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

package components

import (
	"k8s.io/kops/pkg/apis/kops"
	"k8s.io/kops/upup/pkg/fi"
	"k8s.io/kops/upup/pkg/fi/loader"
)

// KubeSchedulerOptionsBuilder adds options for kube-scheduler to the model
type KubeSchedulerOptionsBuilder struct {
	*OptionsContext
}

var _ loader.ClusterOptionsBuilder = &KubeSchedulerOptionsBuilder{}

func (b *KubeSchedulerOptionsBuilder) BuildOptions(o *kops.Cluster) error {
	clusterSpec := &o.Spec
	if clusterSpec.KubeScheduler == nil {
		clusterSpec.KubeScheduler = &kops.KubeSchedulerConfig{}
	}

	config := clusterSpec.KubeScheduler

	if config.LogLevel == 0 {
		// TODO: No way to set to 0?
		config.LogLevel = 2
	}

	if config.Image == "" {
		image, err := Image("kube-scheduler", clusterSpec, b.AssetBuilder)
		if err != nil {
			return err
		}
		config.Image = image
	}

	if config.LeaderElection == nil {
		//  Doesn't seem to be any real downside to always doing a leader election
		config.LeaderElection = &kops.LeaderElectionConfiguration{
			LeaderElect: fi.PtrTo(true),
		}
	}

	if clusterSpec.CloudProvider.AWS != nil {

		if config.FeatureGates == nil {
			config.FeatureGates = make(map[string]string)
		}

		if _, found := config.FeatureGates["InTreePluginAWSUnregister"]; !found && b.ControlPlaneKubernetesVersion().IsLT("1.31") {
			config.FeatureGates["InTreePluginAWSUnregister"] = "true"
		}

		if _, found := config.FeatureGates["CSIMigrationAWS"]; !found && b.ControlPlaneKubernetesVersion().IsLT("1.27") {
			config.FeatureGates["CSIMigrationAWS"] = "true"
		}
	}
	return nil
}
