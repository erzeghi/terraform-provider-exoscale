package exoscale

import (
	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/schema"
)

func SecurityGroupResource() *schema.Resource {
	return &schema.Resource{
		Create: CreateSecurityGroup,
		Read:   ReadSecurityGroup,
		Delete: DeleteSecurityGroup,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},
			"account": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"virtual_machine_count": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"virtual_machine_ids": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Set: schema.HashString,
			},
		},
	}
}

// CreateSecurityGroup creates a security group
func CreateSecurityGroup(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)

	resp, err := client.CreateSecurityGroup(egoscale.SecurityGroupProfile{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
	})
	if err != nil {
		return err
	}

	return applySecurityGroup(resp, d)
}

// ReadSecurityGroup updates the status of the security group from the API
func ReadSecurityGroup(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	resp, err := client.GetSecurityGroupById(d.Id())
	if err != nil {
		// XXX A deleted SecurityGroup returns a 431 error.
		//     we shall use pkg/errors to know more about it
		//     and act accordingly (real timeout error vs not
		//     found error)
		d.SetId("")
		return nil
	}

	return applySecurityGroup(resp, d)
}

// DeleteSecurityGroup deletes a security group
func DeleteSecurityGroup(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	err := client.DeleteSecurityGroup(d.Get("name").(string))
	if err != nil {
		return err
	}

	return nil
}

// applySecurityGroup combines the CloudStack and TerraForm worlds
func applySecurityGroup(securityGroup *egoscale.SecurityGroup, d *schema.ResourceData) error {
	d.SetId(securityGroup.Id)
	d.Set("name", securityGroup.Name)
	d.Set("description", securityGroup.Description)
	d.Set("virtual_machine_count", securityGroup.VirtualMachineCount)
	d.Set("virtual_machine_ids", securityGroup.VirtualMachineIds)

	return nil
}
