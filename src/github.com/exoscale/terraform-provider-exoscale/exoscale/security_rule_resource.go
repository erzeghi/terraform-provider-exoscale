package exoscale

import (
	"log"
	"strings"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func SecurityGroupRuleResource() *schema.Resource {
	return &schema.Resource{
		Create: CreateSecurityGroupRule,
		Read:   ReadSecurityGroupRule,
		Delete: DeleteSecurityGroupRule,

		Schema: map[string]*schema.Schema{
			"type": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"ingress", "egress"}, true),
			},
			"security_group_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"security_group_name"},
			},
			"security_group_name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"security_group_id"},
			},
			"cidr": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ValidateFunc:  validation.CIDRNetwork(0, 32),
				ConflictsWith: []string{"user_security_group"},
			},
			"protocol": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "TCP",
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"TCP", "UDP", "ICMP", "AH", "ESP", "GRE"}, true),
			},
			"start_port": {
				Type:          schema.TypeInt,
				Optional:      true,
				ForceNew:      true,
				ValidateFunc:  validation.IntBetween(1, 65535),
				ConflictsWith: []string{"icmp_type", "icmp_code"},
			},
			"end_port": {
				Type:          schema.TypeInt,
				Optional:      true,
				ForceNew:      true,
				ValidateFunc:  validation.IntBetween(1, 65535),
				ConflictsWith: []string{"icmp_type", "icmp_code"},
			},
			"icmp_type": {
				Type:          schema.TypeInt,
				Optional:      true,
				ForceNew:      true,
				ValidateFunc:  validation.IntBetween(0, 255),
				ConflictsWith: []string{"start_port", "end_port"},
			},
			"icmp_code": {
				Type:          schema.TypeInt,
				Optional:      true,
				ForceNew:      true,
				ValidateFunc:  validation.IntBetween(0, 255),
				ConflictsWith: []string{"start_port", "end_port"},
			},
			"user_security_group": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"cidr"},
			},
		},
	}
}

func CreateSecurityGroupRule(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	async := meta.(BaseConfig).async

	securityGroupId := d.Get("security_group_id").(string)
	if securityGroupId == "" {
		securityGroupId = d.Get("security_group_name").(string)
	}
	securityGroup, err := getSecurityGroup(client, securityGroupId)
	if err != nil {
		return err
	}

	securityGroupRuleProfile := egoscale.SecurityGroupRuleProfile{
		SecurityGroupId: securityGroup.Id,
		Cidr:            d.Get("cidr").(string),
		Protocol:        strings.ToUpper(d.Get("protocol").(string)),
		StartPort:       d.Get("start_port").(int),
		EndPort:         d.Get("end_port").(int),
		IcmpType:        d.Get("icmp_type").(int),
		IcmpCode:        d.Get("icmp_code").(int),
	}

	if port, ok := d.GetOkExists("port"); ok && port.(int) > 0 {
		securityGroupRuleProfile.StartPort = port.(int)
		securityGroupRuleProfile.EndPort = port.(int)
	}

	// XXX Within TerraForm, it's easy to manage simple instances
	//     once linked between many others... it's painful.
	//     Hence the slice of 1, below.
	groupName := d.Get("user_security_group").(string)
	if groupName != "" {
		userSecurityGroup, err := getSecurityGroup(client, groupName)
		if err != nil {
			return err
		}

		groupList := make([]*egoscale.UserSecurityGroup, 1)
		groupList[0] = &egoscale.UserSecurityGroup{
			Account: userSecurityGroup.Account,
			Group:   userSecurityGroup.Name,
		}

		securityGroupRuleProfile.UserSecurityGroupList = groupList
	}

	var resp *egoscale.SecurityGroupRule
	kind := strings.ToLower(d.Get("type").(string))
	if kind == "ingress" {
		resp, err = client.CreateIngressRule(securityGroupRuleProfile, async)
	} else {
		resp, err = client.CreateEgressRule(securityGroupRuleProfile, async)
	}

	if err != nil {
		return err
	}

	d.SetId(resp.Id)
	d.Set("security_group_id", securityGroup.Id)
	d.Set("security_group_name", securityGroup.Name)
	return applySecurityGroupRule(resp, d)
}

// ReadSecurityGroupRule update a rule
func ReadSecurityGroupRule(d *schema.ResourceData, meta interface{}) error {
	securityGroupId := d.Get("security_group_id").(string)
	if securityGroupId == "" {
		return nil
	}

	client := GetComputeClient(meta)
	securityGroup, err := getSecurityGroup(client, securityGroupId)
	if err != nil {
		return err
	}

	var rules []*egoscale.SecurityGroupRule
	log.Printf("[DEBUG] sg %#v", securityGroup)
	if d.Get("type") == "ingress" {
		rules = securityGroup.IngressRules
	} else {
		rules = securityGroup.EgressRules
	}

	for _, rule := range rules {
		if rule.Id == d.Id() {
			return applySecurityGroupRule(rule, d)
		}
	}

	d.SetId("")
	return nil
}

// DeleteSecurityGroupRule deletes a rule
func DeleteSecurityGroupRule(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	async := meta.(BaseConfig).async

	kind := strings.ToLower(d.Get("type").(string))
	if kind == "ingress" {
		return client.DeleteIngressRule(d.Id(), async)
	}
	return client.DeleteEgressRule(d.Id(), async)
}

func applySecurityGroupRule(rule *egoscale.SecurityGroupRule, d *schema.ResourceData) error {
	d.Set("cidr", rule.Cidr)
	d.Set("end_port", rule.EndPort)
	d.Set("start_port", rule.StartPort)
	d.Set("icmp_type", rule.IcmpType)
	d.Set("icmp_code", rule.IcmpCode)

	return nil
}
