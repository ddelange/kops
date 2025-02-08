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

package components

import (
	"k8s.io/kops/pkg/apis/kops"
	"k8s.io/kops/pkg/apis/kops/util"
	"k8s.io/kops/upup/pkg/fi"
	"k8s.io/kops/upup/pkg/fi/loader"
)

// ClusterAutoscalerOptionsBuilder adds options for cluster autoscaler to the model
type ClusterAutoscalerOptionsBuilder struct {
	*OptionsContext
}

var _ loader.ClusterOptionsBuilder = &ClusterAutoscalerOptionsBuilder{}

func (b *ClusterAutoscalerOptionsBuilder) BuildOptions(o *kops.Cluster) error {
	clusterSpec := &o.Spec
	cas := clusterSpec.ClusterAutoscaler
	if cas == nil || !fi.ValueOf(cas.Enabled) {
		return nil
	}

	if cas.Image == nil {

		image := ""
		v, err := util.ParseKubernetesVersion(clusterSpec.KubernetesVersion)
		if err == nil {
			switch v.Minor {
			case 27:
				image = "registry.k8s.io/autoscaling/cluster-autoscaler:v1.27.7"
			case 28:
				image = "registry.k8s.io/autoscaling/cluster-autoscaler:v1.28.4"
			case 29:
				image = "registry.k8s.io/autoscaling/cluster-autoscaler:v1.29.2"
			case 30:
				image = "registry.k8s.io/autoscaling/cluster-autoscaler:v1.30.0"
			default:
				image = "registry.k8s.io/autoscaling/cluster-autoscaler:v1.30.0"
			}
		}
		cas.Image = fi.PtrTo(image)
	}

	if cas.Expander == "" {
		cas.Expander = "random"
	}
	if cas.IgnoreDaemonSetsUtilization == nil {
		cas.IgnoreDaemonSetsUtilization = fi.PtrTo(false)
	}
	if cas.ScaleDownUtilizationThreshold == nil {
		cas.ScaleDownUtilizationThreshold = fi.PtrTo("0.5")
	}
	if cas.SkipNodesWithCustomControllerPods == nil {
		cas.SkipNodesWithCustomControllerPods = fi.PtrTo(true)
	}
	if cas.SkipNodesWithLocalStorage == nil {
		cas.SkipNodesWithLocalStorage = fi.PtrTo(true)
	}
	if cas.SkipNodesWithSystemPods == nil {
		cas.SkipNodesWithSystemPods = fi.PtrTo(true)
	}
	if cas.BalanceSimilarNodeGroups == nil {
		cas.BalanceSimilarNodeGroups = fi.PtrTo(false)
	}
	if cas.EmitPerNodegroupMetrics == nil {
		cas.EmitPerNodegroupMetrics = fi.PtrTo(false)
	}
	if cas.AWSUseStaticInstanceList == nil {
		cas.AWSUseStaticInstanceList = fi.PtrTo(false)
	}
	if cas.NewPodScaleUpDelay == nil {
		cas.NewPodScaleUpDelay = fi.PtrTo("0s")
	}
	if cas.ScaleDownDelayAfterAdd == nil {
		cas.ScaleDownDelayAfterAdd = fi.PtrTo("10m0s")
	}
	if cas.ScaleDownUnneededTime == nil {
		cas.ScaleDownUnneededTime = fi.PtrTo("10m0s")
	}
	if cas.ScaleDownUnreadyTime == nil {
		cas.ScaleDownUnreadyTime = fi.PtrTo("20m0s")
	}
	if cas.MaxNodeProvisionTime == "" {
		cas.MaxNodeProvisionTime = "15m0s"
	}
	if cas.Expander == "priority" {
		cas.CreatePriorityExpenderConfig = fi.PtrTo(true)
	}

	return nil
}
