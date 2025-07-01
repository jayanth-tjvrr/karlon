package cluster

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// SimpleClusterInput holds the minimal required fields for manifest generation
// Add more fields if needed for convention
type SimpleClusterInput struct {
	ClusterName              string
	Namespace                string
	HCPName                  string
	ControlPlaneEndpointHost string
	OIDCConfigMap            string
	OIDCConfigMapCP          string
	Replicas                 int
	K8sVersion               string
}

// generateClusterManifest creates a multi-document YAML for the cluster
func GenerateClusterManifest(clusterName, namespace, _hcpName, _endpointHost, _oidcConfigMap, _oidcConfigMapCP string, replicas int, k8sVersion string) (string, error) {
	// Use conventions for names and suffixes
	hcpName := clusterName + "cp"
	endpointHost := hcpName + ".default-service.k8s.byo-upg-3.app.dev-pcd.platform9.com"
	oidcConfigMap := "oidc-auth-" + clusterName
	oidcConfigMapCP := "oidc-auth-" + hcpName

	// Ensure 'v' prefix is present in k8sVersion
	if len(k8sVersion) > 0 && k8sVersion[0] != 'v' {
		k8sVersion = "v" + k8sVersion
	}

	apiServerPort := 443
	podCIDR := "10.244.0.0/16"
	serviceCIDR := "10.96.0.0/16"
	capiVersion := "v1.32"
	metricsServer := "0.6.4"
	bundleRepo := "quay.io/platform9"
	bundleType := "k8s"
	byoClusterAPIVersion := "infrastructure.cluster.x-k8s.io/v1beta1"
	clusterAPIVersion := "cluster.x-k8s.io/v1beta1"
	controlPlaneAPIVersion := "controlplane.platform9.io/v1alpha1"
	bootstrapAPIVersion := "bootstrap.cluster.x-k8s.io/v1beta1"

	// Cluster resource
	cluster := map[string]interface{}{
		"apiVersion": clusterAPIVersion,
		"kind":       "Cluster",
		"metadata": map[string]interface{}{
			"name":      clusterName,
			"namespace": namespace,
			"labels": map[string]interface{}{
				"cas-capi-version": capiVersion,
				"core-addons":      "enabled",
				"metrics-server":   metricsServer,
				"proxy-addons":     "enabled",
			},
		},
		"spec": map[string]interface{}{
			"clusterNetwork": map[string]interface{}{
				"apiServerPort": apiServerPort,
				"pods":          map[string]interface{}{"cidrBlocks": []string{podCIDR}},
				"services":      map[string]interface{}{"cidrBlocks": []string{serviceCIDR}},
			},
			"controlPlaneRef": map[string]interface{}{
				"apiVersion": controlPlaneAPIVersion,
				"kind":       "HostedControlPlane",
				"name":       hcpName,
				"namespace":  namespace,
			},
			"infrastructureRef": map[string]interface{}{
				"apiVersion": byoClusterAPIVersion,
				"kind":       "ByoCluster",
				"name":       clusterName,
				"namespace":  namespace,
			},
		},
	}
	// ByoCluster
	byoCluster := map[string]interface{}{
		"apiVersion": byoClusterAPIVersion,
		"kind":       "ByoCluster",
		"metadata": map[string]interface{}{
			"name":      clusterName,
			"namespace": namespace,
		},
		"spec": map[string]interface{}{
			"bundleLookupBaseRegistry": bundleRepo,
			"controlPlaneEndpoint": map[string]interface{}{
				"host": endpointHost,
				"port": apiServerPort,
			},
		},
	}
	// HostedControlPlane
	// HostedControlPlane expects version WITHOUT 'v' prefix
	hcpVersion := k8sVersion
	if len(hcpVersion) > 0 && hcpVersion[0] == 'v' {
		hcpVersion = hcpVersion[1:]
	}
	hcp := map[string]interface{}{
		"apiVersion": controlPlaneAPIVersion,
		"kind":       "HostedControlPlane",
		"metadata": map[string]interface{}{
			"name":      hcpName,
			"namespace": namespace,
		},
		"spec": map[string]interface{}{
			"addons": map[string]interface{}{
				"coreDNS": map[string]interface{}{},
				"konnectivity": map[string]interface{}{
					"agent": map[string]interface{}{
						"extraArgs": []string{
							"--proxy-server-host=konnectivity-" + hcpName + ".default-service.k8s.byo-upg-3.app.dev-pcd.platform9.com",
							"--proxy-server-port=443",
						},
						"image":       "registry.k8s.io/kas-network-proxy/proxy-agent",
						"tolerations": []map[string]interface{}{{"key": "CriticalAddonsOnly", "operator": "Exists"}},
						"version":     "v0.0.32",
					},
					"server": map[string]interface{}{
						"image":   "registry.k8s.io/kas-network-proxy/proxy-server",
						"port":    8132,
						"version": "v0.0.32",
					},
				},
			},
			"apiServer": map[string]interface{}{
				"extraArgs": []string{
					"--cloud-provider=external",
					"--cors-allowed-origins=https://byo-upg-3-regionone.app.dev-pcd.platform9.com,https://byo-upg-3-regionone.app.dev-pcd.platform9.com/,https://byo-upg-3-regionone.app.dev-pcd.platform9.com,https://byo-upg-3-regionone.app.dev-pcd.platform9.com/",
					"--advertise-address=10.96.0.40",
					"--authentication-config=/etc/kubernetes/oidc-auth/oidc.yaml",
				},
				"extraVolumeMounts": []map[string]interface{}{
					{"mountPath": "/etc/ssl/host-certs", "name": "etc-ssl-certs-from-host", "readOnly": true},
					{"mountPath": "/etc/kubernetes/oidc-auth", "name": "oidc-auth", "readOnly": true},
				},
				"resources": map[string]interface{}{},
			},
			"controllerManager": map[string]interface{}{
				"extraArgs": []string{"--cloud-provider=external"},
				"resources": map[string]interface{}{},
			},
			"extraCertSANs": []string{
				"konnectivity-" + hcpName + ".default-service.k8s.byo-upg-3.app.dev-pcd.platform9.com",
				hcpName + ".default-service.k8s.byo-upg-3.app.dev-pcd.platform9.com",
			},
			"extraVolumes": []map[string]interface{}{
				{"configMap": map[string]interface{}{"name": oidcConfigMap}, "name": "oidc-auth"},
				{"configMap": map[string]interface{}{"name": oidcConfigMapCP}, "name": "oidc-auth"},
			},
			"hcpClass": "default",
			"hostname": hcpName + ".default-service.k8s.byo-upg-3.app.dev-pcd.platform9.com",
			"kubelet": map[string]interface{}{
				"cgroupfs":              "systemd",
				"preferredAddressTypes": []string{"InternalIP", "ExternalIP", "Hostname"},
			},
			"scheduler": map[string]interface{}{"resources": map[string]interface{}{}},
			"version":   k8sVersion,
		},
	}
	// ByoMachineTemplate
	byoMachineTemplate := map[string]interface{}{
		"apiVersion": byoClusterAPIVersion,
		"kind":       "ByoMachineTemplate",
		"metadata": map[string]interface{}{
			"name":      clusterName,
			"namespace": namespace,
		},
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"spec": map[string]interface{}{
					"installerRef": map[string]interface{}{
						"apiVersion": byoClusterAPIVersion,
						"kind":       "K8sInstallerConfigTemplate",
						"name":       clusterName,
						"namespace":  namespace,
					},
				},
			},
		},
	}
	// KubeadmConfigTemplate
	kubeadmConfigTemplate := map[string]interface{}{
		"apiVersion": bootstrapAPIVersion,
		"kind":       "KubeadmConfigTemplate",
		"metadata": map[string]interface{}{
			"name":      clusterName,
			"namespace": namespace,
		},
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"spec": map[string]interface{}{},
			},
		},
	}
	// MachineDeployment
	// MachineDeployment expects version WITH 'v' prefix
	mdVersion := k8sVersion
	if len(mdVersion) > 0 && mdVersion[0] != 'v' {
		mdVersion = "v" + mdVersion
	}
	machineDeployment := map[string]interface{}{
		"apiVersion": clusterAPIVersion,
		"kind":       "MachineDeployment",
		"metadata": map[string]interface{}{
			"name":      clusterName,
			"namespace": namespace,
		},
		"spec": map[string]interface{}{
			"clusterName": clusterName,
			"replicas":    replicas,
			"template": map[string]interface{}{
				"spec": map[string]interface{}{
					"bootstrap": map[string]interface{}{
						"configRef": map[string]interface{}{
							"apiVersion": bootstrapAPIVersion,
							"kind":       "KubeadmConfigTemplate",
							"name":       clusterName,
						},
					},
					"clusterName": clusterName,
					"infrastructureRef": map[string]interface{}{
						"apiVersion": byoClusterAPIVersion,
						"kind":       "ByoMachineTemplate",
						"name":       clusterName,
					},
					"version": mdVersion,
				},
			},
		},
	}
	// K8sInstallerConfigTemplate
	installerConfigTemplate := map[string]interface{}{
		"apiVersion": byoClusterAPIVersion,
		"kind":       "K8sInstallerConfigTemplate",
		"metadata": map[string]interface{}{
			"name":      clusterName,
			"namespace": namespace,
		},
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"spec": map[string]interface{}{
					"bundleRepo": bundleRepo,
					"bundleType": bundleType,
				},
			},
		},
	}
	// Marshal all resources to YAML
	resources := []interface{}{
		cluster, byoCluster, hcp, byoMachineTemplate, kubeadmConfigTemplate, machineDeployment, installerConfigTemplate,
	}
	docs := [][]byte{}
	for _, obj := range resources {
		b, err := yaml.Marshal(obj)
		if err != nil {
			return "", fmt.Errorf("failed to marshal manifest: %w", err)
		}
		docs = append(docs, b)
	}
	result := string(docs[0])
	for i := 1; i < len(docs); i++ {
		result += "\n---\n" + string(docs[i])
	}
	return result, nil
}
