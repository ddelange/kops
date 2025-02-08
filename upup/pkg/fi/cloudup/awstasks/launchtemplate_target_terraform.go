/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package awstasks

import (
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"k8s.io/kops/upup/pkg/fi"
	"k8s.io/kops/upup/pkg/fi/cloudup/awsup"
	"k8s.io/kops/upup/pkg/fi/cloudup/terraform"
	"k8s.io/kops/upup/pkg/fi/cloudup/terraformWriter"
	"k8s.io/kops/util/pkg/maps"
)

type terraformLaunchTemplateNetworkInterface struct {
	// AssociatePublicIPAddress associates a public ip address with the network interface. Boolean value.
	AssociatePublicIPAddress *bool `cty:"associate_public_ip_address"`
	// DeleteOnTermination indicates whether the network interface should be destroyed on instance termination.
	DeleteOnTermination *bool `cty:"delete_on_termination"`
	// Ipv6AddressCount is the number of IPv6 addresses to assign with the primary network interface.
	Ipv6AddressCount *int32 `cty:"ipv6_address_count"`
	// SecurityGroups is a list of security group ids.
	SecurityGroups []*terraformWriter.Literal `cty:"security_groups"`
}

type terraformLaunchTemplateMonitoring struct {
	// Enabled indicates that monitoring is enabled
	Enabled *bool `cty:"enabled"`
}

type terraformLaunchTemplatePlacement struct {
	// Affinity is he affinity setting for an instance on a Dedicated Host.
	Affinity *string `cty:"affinity"`
	// AvailabilityZone is the Availability Zone for the instance.
	AvailabilityZone *string `cty:"availability_zone"`
	// GroupName is the name of the placement group for the instance.
	GroupName *string `cty:"group_name"`
	// HostID is the ID of the Dedicated Host for the instance.
	HostID *string `cty:"host_id"`
	// SpreadDomain are reserved for future use.
	SpreadDomain *string `cty:"spread_domain"`
	// Tenancy ist he tenancy of the instance. Can be default, dedicated, or host.
	Tenancy *ec2types.Tenancy `cty:"tenancy"`
}

type terraformLaunchTemplateIAMProfile struct {
	// Name is the name of the profile
	Name *terraformWriter.Literal `cty:"name"`
}

type terraformLaunchTemplateMarketOptionsSpotOptions struct {
	// BlockDurationMinutes is required duration in minutes. This value must be a multiple of 60.
	BlockDurationMinutes *int32 `cty:"block_duration_minutes"`
	// InstanceInterruptionBehavior is the behavior when a Spot Instance is interrupted. Can be hibernate, stop, or terminate
	InstanceInterruptionBehavior *ec2types.InstanceInterruptionBehavior `cty:"instance_interruption_behavior"`
	// MaxPrice is the maximum hourly price you're willing to pay for the Spot Instances
	MaxPrice *string `cty:"max_price"`
	// SpotInstanceType is the Spot Instance request type. Can be one-time, or persistent
	SpotInstanceType *string `cty:"spot_instance_type"`
	// ValidUntil is the end date of the request
	ValidUntil *string `cty:"valid_until"`
}

type terraformLaunchTemplateMarketOptions struct {
	// MarketType is the option type
	MarketType *string `cty:"market_type"`
	// SpotOptions are the set of options
	SpotOptions []*terraformLaunchTemplateMarketOptionsSpotOptions `cty:"spot_options"`
}

type terraformLaunchTemplateBlockDeviceEBS struct {
	// VolumeType is the ebs type to use
	VolumeType *string `cty:"volume_type"`
	// VolumeSize is the volume size
	VolumeSize *int32 `cty:"volume_size"`
	// IOPS is the provisioned IOPS
	IOPS *int32 `cty:"iops"`
	// Throughput is the gp3 volume throughput
	Throughput *int32 `cty:"throughput"`
	// DeleteOnTermination indicates the volume should die with the instance
	DeleteOnTermination *bool `cty:"delete_on_termination"`
	// Encrypted indicates the device should be encrypted
	Encrypted *bool `cty:"encrypted"`
	// KmsKeyID is the encryption key identifier for the volume
	KmsKeyID *string `cty:"kms_key_id"`
}

type terraformLaunchTemplateBlockDevice struct {
	// DeviceName is the name of the device
	DeviceName *string `cty:"device_name"`
	// VirtualName is used for the ephemeral devices
	VirtualName *string `cty:"virtual_name"`
	// EBS defines the ebs spec
	EBS []*terraformLaunchTemplateBlockDeviceEBS `cty:"ebs"`
}

type terraformLaunchTemplateCreditSpecification struct {
	CPUCredits *string `cty:"cpu_credits"`
}

type terraformLaunchTemplateTagSpecification struct {
	// ResourceType is the type of resource to tag.
	ResourceType *string `cty:"resource_type"`
	// Tags are the tags to apply to the resource.
	Tags map[string]string `cty:"tags"`
}

type terraformLaunchTemplateInstanceMetadata struct {
	// HTTPEndpoint enables or disables the HTTP metadata endpoint on instances.
	HTTPEndpoint *string `cty:"http_endpoint"`
	// HTTPPutResponseHopLimit is the desired HTTP PUT response hop limit for instance metadata requests.
	HTTPPutResponseHopLimit *int32 `cty:"http_put_response_hop_limit"`
	// HTTPTokens is the state of token usage for your instance metadata requests.
	HTTPTokens *ec2types.LaunchTemplateHttpTokensState `cty:"http_tokens"`
	// HTTPProtocolIPv6 enables the IPv6 instance metadata endpoint
	HTTPProtocolIPv6 *ec2types.LaunchTemplateInstanceMetadataProtocolIpv6 `cty:"http_protocol_ipv6"`
}

type terraformLaunchTemplate struct {
	// Name is the name of the launch template
	Name *string `cty:"name"`
	// Lifecycle is the terraform lifecycle
	Lifecycle *terraform.Lifecycle `cty:"lifecycle"`

	// BlockDeviceMappings is the device mappings
	BlockDeviceMappings []*terraformLaunchTemplateBlockDevice `cty:"block_device_mappings"`
	// CreditSpecification is the credit option for CPU Usage on some instance types
	CreditSpecification *terraformLaunchTemplateCreditSpecification `cty:"credit_specification"`
	// EBSOptimized indicates if the root device is ebs optimized
	EBSOptimized *bool `cty:"ebs_optimized"`
	// IAMInstanceProfile is the IAM profile to assign to the nodes
	IAMInstanceProfile []*terraformLaunchTemplateIAMProfile `cty:"iam_instance_profile"`
	// ImageID is the ami to use for the instances
	ImageID *string `cty:"image_id"`
	// InstanceType is the type of instance
	InstanceType *ec2types.InstanceType `cty:"instance_type"`
	// KeyName is the ssh key to use
	KeyName *terraformWriter.Literal `cty:"key_name"`
	// MarketOptions are the spot pricing options
	MarketOptions []*terraformLaunchTemplateMarketOptions `cty:"instance_market_options"`
	// MetadataOptions are the instance metadata options.
	MetadataOptions *terraformLaunchTemplateInstanceMetadata `cty:"metadata_options"`
	// Monitoring are the instance monitoring options
	Monitoring []*terraformLaunchTemplateMonitoring `cty:"monitoring"`
	// NetworkInterfaces are the networking options
	NetworkInterfaces []*terraformLaunchTemplateNetworkInterface `cty:"network_interfaces"`
	// Placement are the tenancy options
	Placement []*terraformLaunchTemplatePlacement `cty:"placement"`
	// Tags is a map of tags applied to the launch template itself
	Tags map[string]string `cty:"tags"`
	// TagSpecifications are the tags to apply to a resource when it is created.
	TagSpecifications []*terraformLaunchTemplateTagSpecification `cty:"tag_specifications"`
	// UserData is the user data for the instances
	UserData *terraformWriter.Literal `cty:"user_data"`
}

// TerraformLink returns the terraform reference
func (t *LaunchTemplate) TerraformLink() *terraformWriter.Literal {
	return terraformWriter.LiteralProperty("aws_launch_template", fi.ValueOf(t.Name), "id")
}

// VersionLink returns the terraform version reference
func (t *LaunchTemplate) VersionLink() *terraformWriter.Literal {
	return terraformWriter.LiteralProperty("aws_launch_template", fi.ValueOf(t.Name), "latest_version")
}

// RenderTerraform is responsible for rendering the terraform json
func (t *LaunchTemplate) RenderTerraform(target *terraform.TerraformTarget, a, e, changes *LaunchTemplate) error {
	var err error

	cloud := target.Cloud.(awsup.AWSCloud)

	var image *string
	if e.ImageID != nil {
		im, err := cloud.ResolveImage(fi.ValueOf(e.ImageID))
		if err != nil {
			return err
		}
		image = im.ImageId
	}

	tf := terraformLaunchTemplate{
		Name:         e.Name,
		EBSOptimized: e.RootVolumeOptimization,
		ImageID:      image,
		InstanceType: e.InstanceType,
		Lifecycle:    &terraform.Lifecycle{CreateBeforeDestroy: fi.PtrTo(true)},
		MetadataOptions: &terraformLaunchTemplateInstanceMetadata{
			// See issue https://github.com/hashicorp/terraform-provider-aws/issues/12564.
			HTTPEndpoint:            fi.PtrTo("enabled"),
			HTTPTokens:              e.HTTPTokens,
			HTTPPutResponseHopLimit: e.HTTPPutResponseHopLimit,
			HTTPProtocolIPv6:        e.HTTPProtocolIPv6,
		},
		NetworkInterfaces: []*terraformLaunchTemplateNetworkInterface{
			{
				AssociatePublicIPAddress: e.AssociatePublicIP,
				DeleteOnTermination:      fi.PtrTo(true),
				Ipv6AddressCount:         e.IPv6AddressCount,
			},
		},
	}

	if fi.ValueOf(e.SpotPrice) != "" {
		marketSpotOptions := terraformLaunchTemplateMarketOptionsSpotOptions{
			BlockDurationMinutes:         e.SpotDurationInMinutes,
			InstanceInterruptionBehavior: e.InstanceInterruptionBehavior,
			MaxPrice:                     e.SpotPrice,
		}
		tf.MarketOptions = []*terraformLaunchTemplateMarketOptions{
			{
				MarketType:  fi.PtrTo("spot"),
				SpotOptions: []*terraformLaunchTemplateMarketOptionsSpotOptions{&marketSpotOptions},
			},
		}
	}
	if fi.ValueOf(e.CPUCredits) != "" {
		tf.CreditSpecification = &terraformLaunchTemplateCreditSpecification{
			CPUCredits: e.CPUCredits,
		}
	}
	for _, x := range e.SecurityGroups {
		tf.NetworkInterfaces[0].SecurityGroups = append(tf.NetworkInterfaces[0].SecurityGroups, x.TerraformLink())
	}
	if e.SSHKey != nil {
		tf.KeyName = e.SSHKey.TerraformLink()
	}
	if e.Tenancy != nil {
		tf.Placement = []*terraformLaunchTemplatePlacement{{Tenancy: e.Tenancy}}
	}
	if e.InstanceMonitoring != nil {
		tf.Monitoring = []*terraformLaunchTemplateMonitoring{
			{Enabled: e.InstanceMonitoring},
		}
	}
	if e.IAMInstanceProfile != nil {
		tf.IAMInstanceProfile = []*terraformLaunchTemplateIAMProfile{
			{Name: e.IAMInstanceProfile.TerraformLink()},
		}
	}
	if e.UserData != nil {
		d, err := fi.ResourceAsBytes(e.UserData)
		if err != nil {
			return err
		}
		if d != nil {
			tf.UserData, err = target.AddFileBytes("aws_launch_template", fi.ValueOf(e.Name), "user_data", d, true)
			if err != nil {
				return err
			}
		}
	}

	devices, err := e.buildRootDevice(cloud)
	if err != nil {
		return err
	}

	devicesKeys := maps.SortedKeys(devices)
	for _, key := range devicesKeys {
		terraformLaunchTemplateBlockDevice := createTerraformLaunchTemplateBlockDevice(key, devices[key])
		tf.BlockDeviceMappings = append(tf.BlockDeviceMappings, terraformLaunchTemplateBlockDevice)
	}

	additionals, err := buildAdditionalDevices(e.BlockDeviceMappings)
	if err != nil {
		return err
	}

	additionalsKeys := maps.SortedKeys(additionals)
	for _, key := range additionalsKeys {
		terraformLaunchTemplateBlockDevice := createTerraformLaunchTemplateBlockDevice(key, additionals[key])
		tf.BlockDeviceMappings = append(tf.BlockDeviceMappings, terraformLaunchTemplateBlockDevice)
	}

	devices, err = buildEphemeralDevices(cloud, fi.ValueOf(e.InstanceType))
	if err != nil {
		return err
	}

	devicesKeys = maps.SortedKeys(devices)
	for _, key := range devicesKeys {
		tf.BlockDeviceMappings = append(tf.BlockDeviceMappings, &terraformLaunchTemplateBlockDevice{
			VirtualName: devices[key].VirtualName,
			DeviceName:  fi.PtrTo(key),
		})
	}

	if e.Tags != nil {
		tf.TagSpecifications = append(tf.TagSpecifications, &terraformLaunchTemplateTagSpecification{
			ResourceType: fi.PtrTo("instance"),
			Tags:         e.Tags,
		})
		tf.TagSpecifications = append(tf.TagSpecifications, &terraformLaunchTemplateTagSpecification{
			ResourceType: fi.PtrTo("volume"),
			Tags:         e.Tags,
		})
		tf.Tags = e.Tags
	}

	return target.RenderResource("aws_launch_template", fi.ValueOf(e.Name), tf)
}

func createTerraformLaunchTemplateBlockDevice(deviceName string, v *BlockDeviceMapping) *terraformLaunchTemplateBlockDevice {
	return &terraformLaunchTemplateBlockDevice{
		DeviceName: fi.PtrTo(deviceName),
		EBS: []*terraformLaunchTemplateBlockDeviceEBS{
			{
				DeleteOnTermination: fi.PtrTo(true),
				Encrypted:           v.EbsEncrypted,
				KmsKeyID:            v.EbsKmsKey,
				IOPS:                v.EbsVolumeIops,
				Throughput:          v.EbsVolumeThroughput,
				VolumeSize:          v.EbsVolumeSize,
				VolumeType:          fi.PtrTo(string(v.EbsVolumeType)),
			},
		},
	}
}
