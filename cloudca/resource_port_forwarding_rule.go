package cloudca

import (
	"fmt"
	"github.com/cloud-ca/go-cloudca"
	"github.com/cloud-ca/go-cloudca/api"
	"github.com/cloud-ca/go-cloudca/services/cloudca"
	"github.com/hashicorp/terraform/helper/schema"
	"strconv"
)

func resourceCloudcaPortForwardingRule() *schema.Resource {
	return &schema.Resource{
		Create: createPortForwardingRule,
		Read:   readPortForwardingRule,
		Delete: deletePortForwardingRule,

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
				Description: "Name of environment where port forwarding rule should be created",
			},
			"public_ip_id": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The public IP to which these rules should be applied",
			},
			"private_ip_id": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the private IP to bind to",
			},
			"protocol": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The protocol that this rule should use (eg. TCP, UDP)",
			},
			"private_port_start": &schema.Schema{
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The start of the private port range for this rule",
			},
			"private_port_end": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Computed:    true,
				Description: "The end of the private port range for this rule",
			},
			"public_port_start": &schema.Schema{
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The start of the public port range for this rule",
			},
			"public_port_end": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Computed:    true,
				Description: "The end of the public port range for this rule",
			},
			"public_ip": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"private_ip": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"instance_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func createPortForwardingRule(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cca.CcaClient)
	resources, err := client.GetResources(d.Get("service_code").(string), d.Get("environment_name").(string))
	ccaResources := resources.(cloudca.Resources)

	if err != nil {
		return err
	}

	pfr := cloudca.PortForwardingRule{
		PublicIpId:       d.Get("public_ip_id").(string),
		Protocol:         d.Get("protocol").(string),
		PublicPortStart:  strconv.Itoa(d.Get("public_port_start").(int)),
		PrivateIpId:      d.Get("private_ip_id").(string),
		PrivatePortStart: strconv.Itoa(d.Get("private_port_start").(int)),
	}

	if _, ok := d.GetOk("public_port_end"); ok {
		pfr.PublicPortEnd = strconv.Itoa(d.Get("public_port_end").(int))
	}

	if _, ok := d.GetOk("private_port_end"); ok {
		pfr.PrivatePortEnd = strconv.Itoa(d.Get("private_port_end").(int))
	}

	newPfr, err := ccaResources.PortForwardingRules.Create(pfr)
	if err != nil {
		return err
	}

	d.SetId(newPfr.Id)
	return readPortForwardingRule(d, meta)
}

func readPortForwardingRule(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cca.CcaClient)
	resources, _ := client.GetResources(d.Get("service_code").(string), d.Get("environment_name").(string))
	ccaResources := resources.(cloudca.Resources)

	pfr, err := ccaResources.PortForwardingRules.Get(d.Id())
	if err != nil {
		return handleNotFoundError(err, d)
	}

	d.Set("public_ip_id", pfr.PublicIpId)
	d.Set("private_ip_id", pfr.PrivateIpId)
	d.Set("instance_id", pfr.InstanceId)
	d.Set("protocol", pfr.Protocol)
	d.Set("public_port_start", pfr.PublicPortStart)
	d.Set("public_port_end", pfr.PublicPortEnd)
	d.Set("private_port_start", pfr.PrivatePortStart)
	d.Set("private_port_end", pfr.PrivatePortEnd)
	d.Set("private_ip", pfr.PrivateIp)
	d.Set("public_ip", pfr.PublicIp)

	return nil
}

func deletePortForwardingRule(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cca.CcaClient)
	resources, _ := client.GetResources(d.Get("service_code").(string), d.Get("environment_name").(string))
	ccaResources := resources.(cloudca.Resources)

	if _, err := ccaResources.PortForwardingRules.Delete(d.Id()); err != nil {
		return handleNotFoundError(err, d)
	}
	return nil
}

func handleNotFoundError(err error, d *schema.ResourceData) error {
	if ccaError, ok := err.(api.CcaErrorResponse); ok {
		if ccaError.StatusCode == 404 {
			fmt.Errorf("Port forwarding rule with id %s no longer exists", d.Id())
			d.SetId("")
			return nil
		}
	}

	return err
}
