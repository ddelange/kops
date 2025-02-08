/*
Copyright 2017 The Kubernetes Authors.

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

package openstacktasks

import (
	"context"
	"fmt"
	"sort"

	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/listeners"
	"k8s.io/klog/v2"
	"k8s.io/kops/upup/pkg/fi"
	"k8s.io/kops/upup/pkg/fi/cloudup/openstack"
)

// +kops:fitask
type LBListener struct {
	ID           *string
	Name         *string
	Port         *int
	Pool         *LBPool
	Lifecycle    fi.Lifecycle
	AllowedCIDRs []string
}

// GetDependencies returns the dependencies of the Instance task
func (e *LBListener) GetDependencies(tasks map[string]fi.CloudupTask) []fi.CloudupTask {
	var deps []fi.CloudupTask
	for _, task := range tasks {
		if _, ok := task.(*LB); ok {
			deps = append(deps, task)
		}
		if _, ok := task.(*LBPool); ok {
			deps = append(deps, task)
		}
	}
	return deps
}

var _ fi.CompareWithID = &LBListener{}

func (s *LBListener) CompareWithID() *string {
	return s.ID
}

func NewLBListenerTaskFromCloud(cloud openstack.OpenstackCloud, lifecycle fi.Lifecycle, listener *listeners.Listener, find *LBListener) (*LBListener, error) {
	// sort for consistent comparison
	sort.Strings(listener.AllowedCIDRs)
	listenerTask := &LBListener{
		ID:           fi.PtrTo(listener.ID),
		Name:         fi.PtrTo(listener.Name),
		Port:         fi.PtrTo(listener.ProtocolPort),
		AllowedCIDRs: listener.AllowedCIDRs,
		Lifecycle:    lifecycle,
	}

	if len(listener.Pools) > 0 {
		for _, pool := range listener.Pools {
			poolTask, err := NewLBPoolTaskFromCloud(cloud, lifecycle, &pool, find.Pool)
			if err != nil {
				return nil, fmt.Errorf("NewLBListenerTaskFromCloud: Failed to create new LBListener task for pool %s: %v", pool.Name, err)
			} else {
				listenerTask.Pool = poolTask
				// TODO: Support Multiple?
				break
			}
		}
	} else {
		pool, err := cloud.GetPool(listener.DefaultPoolID)
		if err != nil {
			return nil, fmt.Errorf("Fail to get pool with ID: %s: %v", listener.DefaultPoolID, err)
		}
		poolTask, err := NewLBPoolTaskFromCloud(cloud, lifecycle, pool, find.Pool)
		if err != nil {
			return nil, fmt.Errorf("NewLBListenerTaskFromCloud: Failed to create new LBListener task for pool %s: %v", pool.Name, err)
		}
		listenerTask.Pool = poolTask
	}
	if find != nil {
		// Update all search terms
		find.ID = listenerTask.ID
		find.Name = listenerTask.Name
		find.Pool = listenerTask.Pool
	}
	return listenerTask, nil
}

func (s *LBListener) Find(context *fi.CloudupContext) (*LBListener, error) {
	if s.Name == nil {
		return nil, nil
	}

	cloud := context.T.Cloud.(openstack.OpenstackCloud)
	listenerList, err := cloud.ListListeners(listeners.ListOpts{
		ID:   fi.ValueOf(s.ID),
		Name: fi.ValueOf(s.Name),
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to list loadbalancer listeners for name %s: %v", fi.ValueOf(s.Name), err)
	}
	if len(listenerList) == 0 {
		return nil, nil
	}
	if len(listenerList) > 1 {
		return nil, fmt.Errorf("Multiple listeners found with name %s", fi.ValueOf(s.Name))
	}

	return NewLBListenerTaskFromCloud(cloud, s.Lifecycle, &listenerList[0], s)
}

func (s *LBListener) Run(context *fi.CloudupContext) error {
	return fi.CloudupDefaultDeltaRunMethod(s, context)
}

func (_ *LBListener) CheckChanges(a, e, changes *LBListener) error {
	if a == nil {
		if e.Name == nil {
			return fi.RequiredField("Name")
		}
	} else {
		if changes.ID != nil {
			return fi.CannotChangeField("ID")
		}
		if changes.Name != nil {
			return fi.CannotChangeField("Name")
		}
	}
	return nil
}

func (_ *LBListener) RenderOpenstack(t *openstack.OpenstackAPITarget, a, e, changes *LBListener) error {
	useVIPACL, err := t.Cloud.UseLoadBalancerVIPACL()
	if err != nil {
		return err
	}

	if a == nil {
		klog.V(2).Infof("Creating LB with Name: %q", fi.ValueOf(e.Name))
		listeneropts := listeners.CreateOpts{
			Name:           fi.ValueOf(e.Name),
			DefaultPoolID:  fi.ValueOf(e.Pool.ID),
			LoadbalancerID: fi.ValueOf(e.Pool.Loadbalancer.ID),
			Protocol:       listeners.ProtocolTCP,
			ProtocolPort:   fi.ValueOf(e.Port),
		}

		if useVIPACL && (fi.ValueOf(e.Pool.Loadbalancer.Provider) != "ovn") {
			listeneropts.AllowedCIDRs = e.AllowedCIDRs
		}

		listener, err := t.Cloud.CreateListener(listeneropts)
		if err != nil {
			return fmt.Errorf("error creating LB listener: %v", err)
		}
		e.ID = fi.PtrTo(listener.ID)
		return nil
	} else if len(changes.AllowedCIDRs) > 0 {
		if useVIPACL && (fi.ValueOf(a.Pool.Loadbalancer.Provider) != "ovn") {
			opts := listeners.UpdateOpts{
				AllowedCIDRs: &changes.AllowedCIDRs,
			}
			_, err := listeners.Update(context.TODO(), t.Cloud.LoadBalancerClient(), fi.ValueOf(a.ID), opts).Extract()
			if err != nil {
				return fmt.Errorf("error updating LB listener: %v", err)
			}
		} else {
			klog.V(2).Infof("Openstack Octavia VIPACLs not supported")
		}
		return nil
	}
	klog.V(2).Infof("Openstack task LB::RenderOpenstack did nothing")
	return nil
}
