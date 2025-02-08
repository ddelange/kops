/*
Copyright 2020 The Kubernetes Authors.

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

package azuretasks

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	compute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"k8s.io/klog/v2"
	"k8s.io/kops/upup/pkg/fi"
	"k8s.io/kops/upup/pkg/fi/cloudup/azure"
)

// Disk is an Azure Managed Disk.
// +kops:fitask
type Disk struct {
	Name      *string
	Lifecycle fi.Lifecycle

	ResourceGroup *ResourceGroup
	SizeGB        *int32
	Tags          map[string]*string
	VolumeType    *compute.DiskStorageAccountTypes
	Zones         []*string
}

var (
	_ fi.CloudupTask          = &Disk{}
	_ fi.CompareWithID        = &Disk{}
	_ fi.CloudupTaskNormalize = &Disk{}
)

// CompareWithID returns the Name of the Disk.
func (d *Disk) CompareWithID() *string {
	return d.Name
}

// Find discovers the Disk in the cloud provider.
func (d *Disk) Find(c *fi.CloudupContext) (*Disk, error) {
	cloud := c.T.Cloud.(azure.AzureCloud)
	l, err := cloud.Disk().List(context.TODO(), *d.ResourceGroup.Name)
	if err != nil {
		return nil, err
	}
	var found *compute.Disk
	for _, v := range l {
		if *v.Name == *d.Name {
			found = v
			break
		}
	}
	if found == nil {
		return nil, nil
	}

	disk := &Disk{
		Name:      d.Name,
		Lifecycle: d.Lifecycle,
		ResourceGroup: &ResourceGroup{
			Name: d.ResourceGroup.Name,
		},
		SizeGB: found.Properties.DiskSizeGB,
		Tags:   found.Tags,
		Zones:  found.Zones,
	}
	if found.SKU != nil && found.SKU.Name != nil {
		disk.VolumeType = found.SKU.Name
	}
	if found.Properties != nil {
		disk.SizeGB = found.Properties.DiskSizeGB
	}

	return disk, nil
}

func (d *Disk) Normalize(c *fi.CloudupContext) error {
	c.T.Cloud.(azure.AzureCloud).AddClusterTags(d.Tags)
	return nil
}

// Run implements fi.Task.Run.
func (d *Disk) Run(c *fi.CloudupContext) error {
	return fi.CloudupDefaultDeltaRunMethod(d, c)
}

// CheckChanges returns an error if a change is not allowed.
func (*Disk) CheckChanges(a, e, changes *Disk) error {
	if a == nil {
		// Check if required fields are set when a new resource is created.
		if e.Name == nil {
			return fi.RequiredField("Name")
		}
		return nil
	}

	// Check if unchangeable fields won't be changed.
	if changes.Name != nil {
		return fi.CannotChangeField("Name")
	}
	return nil
}

// RenderAzure creates or updates a Disk.
func (*Disk) RenderAzure(t *azure.AzureAPITarget, a, e, changes *Disk) error {
	if a == nil {
		klog.Infof("Creating a new Disk with name: %s", fi.ValueOf(e.Name))
	} else {
		klog.Infof("Updating a Disk with name: %s", fi.ValueOf(e.Name))
	}
	name := *e.Name

	disk := compute.Disk{
		Location: to.Ptr(t.Cloud.Region()),
		Properties: &compute.DiskProperties{
			CreationData: &compute.CreationData{
				CreateOption: to.Ptr(compute.DiskCreateOptionEmpty),
			},
			DiskSizeGB: e.SizeGB,
		},
		SKU: &compute.DiskSKU{
			Name: e.VolumeType,
		},
		Tags:  e.Tags,
		Zones: e.Zones,
	}

	_, err := t.Cloud.Disk().CreateOrUpdate(
		context.TODO(),
		*e.ResourceGroup.Name,
		name,
		disk)

	return err
}
