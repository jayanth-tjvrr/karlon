package cluster

import (
	"context"
	"fmt"

	argoapp "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application"
	argoappv1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateClusterApp creates a cluster-app that accompanies an kkarlon-app for gen2 clusters
func CreateClusterApp(
	appIf argoapp.ApplicationServiceClient,
	argocdNs string,
	clusterName string,
	baseClusterName string,
	repoUrl string, // source repo
	repoRevision string, // source revision
	repoPath string, // source path
	createInArgoCd bool,
	overridden bool,
) (*argoappv1.Application, error) {
	app := constructClusterApp(argocdNs, clusterName, baseClusterName,
		repoUrl, repoRevision, repoPath, overridden)
	if createInArgoCd {
		appCreateRequest := argoapp.ApplicationCreateRequest{
			Application: app,
		}
		_, err := appIf.Create(context.Background(), &appCreateRequest)
		if err != nil {
			return nil, fmt.Errorf("failed to create cluster application: %s", err)
		}
	}
	return app, nil
}

// constructClusterApp returns a cluster-app that accompanies an kkarlon-app for gen2 clusters
func constructClusterApp(
	argocdNs string,
	clusterName string,
	baseClusterName string,
	repoUrl string, // source repo
	repoRevision string, // source revision
	repoPath string, // source path
	overridden bool,
) *argoappv1.Application {
	clusterOverridden := fmt.Sprintf("%v", overridden)
	app := &argoappv1.Application{
		TypeMeta: v1.TypeMeta{
			Kind:       application.ApplicationKind,
			APIVersion: application.Group + "/v1alpha1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      clusterName,
			Namespace: argocdNs,
			Labels: map[string]string{
				"managed-by":    "kkarlon",
				"kkarlon-type":    "cluster-app",
				"kkarlon-cluster": clusterName,
			},
			Annotations: map[string]string{
				baseClusterNameAnnotation:         baseClusterName,
				baseClusterRepoUrlAnnotation:      repoUrl,
				baseClusterRepoRevisionAnnotation: repoRevision,
				baseClusterRepoPathAnnotation:     repoPath,
				baseClusterOverridden:             clusterOverridden,
			},
			Finalizers: []string{argoappv1.ForegroundPropagationPolicyFinalizer},
		},
	}
	var ignoreDiffs []argoappv1.ResourceIgnoreDifferences
	// If used, cluster autoscaler will change replicas so ignore it
	ignoreDiffs = append(ignoreDiffs, argoappv1.ResourceIgnoreDifferences{
		Group:        "cluster.x-k8s.io",
		Kind:         "MachineDeployment",
		JSONPointers: []string{"/spec/replicas"},
	})
	// Ignore CAPI EKS control plane's spec.version because the AWS controller(s)
	// appear to update it with a value that is less precise than the requested
	// one, for e.g. the spec might specify v1.18.16, and get updated with v1.18,
	// causing ArgoCD to report the resource as OutOfSync
	ignoreDiffs = append(ignoreDiffs, argoappv1.ResourceIgnoreDifferences{
		Group:        "controlplane.cluster.x-k8s.io",
		Kind:         "AWSManagedControlPlane",
		JSONPointers: []string{"/spec/version"},
	})

	ignoreDiffs = append(ignoreDiffs, argoappv1.ResourceIgnoreDifferences{
		Group:        "infrastructure.cluster.x-k8s.io",
		Kind:         "AWSMachineTemplate",
		JSONPointers: []string{"/spec"},
	})
	app.Spec.IgnoreDifferences = ignoreDiffs
	app.Spec.Source.Kustomize = &argoappv1.ApplicationSourceKustomize{
		NamePrefix: clusterName + "-",
	}
	app.Spec.SyncPolicy = &argoappv1.SyncPolicy{
		Automated: &argoappv1.SyncPolicyAutomated{
			Prune: true,
		},
		SyncOptions: []string{"Prune=true", "RespectIgnoreDifferences=true"},
	}
	finalRepoPath := repoPath
	if overridden {
		finalRepoPath = repoPath + "/" + clusterName
	}
	app.Spec.Source.RepoURL = repoUrl
	app.Spec.Source.TargetRevision = repoRevision
	app.Spec.Source.Path = finalRepoPath
	app.Spec.Destination.Server = "https://kubernetes.default.svc"
	app.Spec.Destination.Namespace = clusterName
	app.Spec.SyncPolicy = &argoappv1.SyncPolicy{
		Automated: &argoappv1.SyncPolicyAutomated{
			Prune: true,
		},
		SyncOptions: []string{"Prune=true"},
	}
	return app
}
