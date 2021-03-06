package cloudca

import (
	"fmt"
	"github.com/cloud-ca/go-cloudca"
	"github.com/cloud-ca/go-cloudca/api"
	"github.com/cloud-ca/go-cloudca/services/cloudca"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
	"strings"
)

func resourceCloudcaVpc() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudcaVpcCreate,
		Read:   resourceCloudcaVpcRead,
		Update: resourceCloudcaVpcUpdate,
		Delete: resourceCloudcaVpcDelete,

		Schema: map[string]*schema.Schema{
			"service_code": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "A cloudca service code",
			},
			"environment_name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of environment where VPC should be created",
			},
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of VPC",
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Description of VPC",
			},
			"vpc_offering": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name or id of the VPC offering",
				StateFunc: func(val interface{}) string {
					return strings.ToLower(val.(string))
				},
			},
			"network_domain": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				ForceNew:    true,
				Description: "A custom DNS suffix at the level of a network",
			},
		},
	}
}

func resourceCloudcaVpcCreate(d *schema.ResourceData, meta interface{}) error {
	ccaClient := meta.(*cca.CcaClient)
	resources, _ := ccaClient.GetResources(d.Get("service_code").(string), d.Get("environment_name").(string))
	ccaResources := resources.(cloudca.Resources)

	vpcOfferingId, cerr := retrieveVpcOfferingID(&ccaResources, d.Get("vpc_offering").(string))

	if cerr != nil {
		return cerr
	}

	vpcToCreate := cloudca.Vpc{
		Name:          d.Get("name").(string),
		Description:   d.Get("description").(string),
		VpcOfferingId: vpcOfferingId,
	}

	if networkDomain, ok := d.GetOk("network_domain"); ok {
		vpcToCreate.NetworkDomain = networkDomain.(string)
	}

	newVpc, err := ccaResources.Vpcs.Create(vpcToCreate)
	if err != nil {
		return fmt.Errorf("Error creating the new VPC %s: %s", vpcToCreate.Name, err)
	}
	d.SetId(newVpc.Id)

	return resourceCloudcaVpcRead(d, meta)
}

func resourceCloudcaVpcRead(d *schema.ResourceData, meta interface{}) error {
	ccaClient := meta.(*cca.CcaClient)
	resources, _ := ccaClient.GetResources(d.Get("service_code").(string), d.Get("environment_name").(string))
	ccaResources := resources.(cloudca.Resources)

	// Get the vpc details
	vpc, err := ccaResources.Vpcs.Get(d.Id())
	if err != nil {
		if ccaError, ok := err.(api.CcaErrorResponse); ok {
			if ccaError.StatusCode == 404 {
				fmt.Errorf("VPC %s does no longer exist", d.Get("name").(string))
				d.SetId("")
				return nil
			}
		}
		return err
	}

	vpcOffering, offErr := ccaResources.VpcOfferings.Get(vpc.VpcOfferingId)
	if offErr != nil {
		if ccaError, ok := offErr.(api.CcaErrorResponse); ok {
			if ccaError.StatusCode == 404 {
				fmt.Errorf("VPC offering id=%s does no longer exist", vpc.VpcOfferingId)
				d.SetId("")
				return nil
			}
		}
		return offErr
	}

	// Update the config
	d.Set("name", vpc.Name)
	d.Set("description", vpc.Description)
	setValueOrID(d, "vpc_offering", strings.ToLower(vpcOffering.Name), vpc.VpcOfferingId)
	d.Set("network_domain", vpc.NetworkDomain)

	return nil
}

func resourceCloudcaVpcUpdate(d *schema.ResourceData, meta interface{}) error {
	ccaClient := meta.(*cca.CcaClient)
	resources, _ := ccaClient.GetResources(d.Get("service_code").(string), d.Get("environment_name").(string))
	ccaResources := resources.(cloudca.Resources)

	if d.HasChange("name") || d.HasChange("description") {
		newName := d.Get("name").(string)
		newDescription := d.Get("description").(string)
		log.Printf("[DEBUG] Details have changed updating VPC.....")
		_, err := ccaResources.Vpcs.Update(cloudca.Vpc{Id: d.Id(), Name: newName, Description: newDescription})
		if err != nil {
			return err
		}
	}

	return nil
}

func resourceCloudcaVpcDelete(d *schema.ResourceData, meta interface{}) error {
	ccaClient := meta.(*cca.CcaClient)
	resources, _ := ccaClient.GetResources(d.Get("service_code").(string), d.Get("environment_name").(string))
	ccaResources := resources.(cloudca.Resources)

	fmt.Println("[INFO] Destroying VPC: %s", d.Get("name").(string))
	if _, err := ccaResources.Vpcs.Destroy(d.Id()); err != nil {
		if ccaError, ok := err.(api.CcaErrorResponse); ok {
			if ccaError.StatusCode == 404 {
				fmt.Errorf("VPC %s does no longer exist", d.Get("name").(string))
				d.SetId("")
				return nil
			}
		}
		return err
	}

	return nil
}

func retrieveVpcOfferingID(ccaRes *cloudca.Resources, name string) (id string, err error) {
	if isID(name) {
		return name, nil
	}

	vpcOfferings, err := ccaRes.VpcOfferings.List()
	if err != nil {
		return "", err
	}
	for _, offering := range vpcOfferings {
		if strings.EqualFold(offering.Name, name) {
			log.Printf("Found vpc offering: %+v", offering)
			return offering.Id, nil
		}
	}

	return "", fmt.Errorf("VPC offering with name %s not found", name)
}
