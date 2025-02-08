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

package model

import (
	"k8s.io/kops/pkg/apis/kops"
)

// UseChallengeCallback is true if we should use a callback challenge during node provisioning with kops-controller.
func UseChallengeCallback(cloudProvider kops.CloudProviderID) bool {
	switch cloudProvider {
	case kops.CloudProviderHetzner:
		return true
	case kops.CloudProviderDO:
		return true
	case kops.CloudProviderScaleway:
		return true
	case kops.CloudProviderAzure:
		return true
	default:
		return false
	}
}

// UseKopsControllerForNodeConfig checks if nodeup should use kops-controller to get nodeup.Config.
func UseKopsControllerForNodeConfig(cluster *kops.Cluster) bool {
	if cluster.UsesLegacyGossip() {
		switch cluster.GetCloudProvider() {
		case kops.CloudProviderGCE:
			// We can use cloud-discovery here.
		case kops.CloudProviderHetzner, kops.CloudProviderScaleway, kops.CloudProviderDO:
			// We don't have a cloud-discovery mechanism implemented in nodeup for many clouds,
			// but we assume that we're using a load balancer with a fixed IP address
		default:
			return false
		}
	}
	return true
}

// UseCiliumEtcd is true if we are using the Cilium etcd cluster.
func UseCiliumEtcd(cluster *kops.Cluster) bool {
	if cluster.Spec.Networking.Cilium == nil {
		return false
	}

	for _, cluster := range cluster.Spec.EtcdClusters {
		if cluster.Name == "cilium" {
			return true
		}
	}

	return false
}

// Configures a Kubelet Credential Provider if Kubernetes is newer than a specific version
func UseExternalKubeletCredentialProvider(k8sVersion *KubernetesVersion, cloudProvider kops.CloudProviderID) bool {
	switch cloudProvider {
	case kops.CloudProviderGCE:
		return k8sVersion.IsGTE("1.29")
	case kops.CloudProviderAWS:
		return k8sVersion.IsGTE("1.27")
	default:
		return false
	}
}
