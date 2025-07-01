/*
Copyright 2021.

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

package controllers

import (
	"context"
	"fmt"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient"
	argoapp "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/argoproj/argo-cd/v2/util/io"
	kkarlonv1 "github.com/jayanth-tjvrr/karlon/pkg/api/v1"
	corev1 "github.com/jayanth-tjvrr/karlon/pkg/api/v1"
	kkarlonapp "github.com/jayanth-tjvrr/karlon/pkg/app"
	"github.com/jayanth-tjvrr/karlon/pkg/argocd"
	bcl "github.com/jayanth-tjvrr/karlon/pkg/basecluster"
	"github.com/jayanth-tjvrr/karlon/pkg/cluster"
	"github.com/go-logr/logr"
	grpccodes "google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

var retryDelayAsResult = ctrl.Result{RequeueAfter: time.Second * 10}

// Default git location of Helm chart for Arlon app (for a cluster)
var defaultArlonChart = kkarlonv1.RepoSpec{
	Url:      "https://github.com/kkarlonproj/kkarlon.git",
	Path:     "pkg/cluster/manifests",
	Revision: "v0.10.0",
}

// ClusterReconciler reconciles a Cluster object
type ClusterReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	ArgocdClient apiclient.Client
	Config       *restclient.Config
	ArgoCdNs     string
	ArlonNs      string
}

//+kubebuilder:rbac:groups=core.kkarlon.io,resources=clusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.kkarlon.io,resources=clusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.kkarlon.io,resources=clusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Cluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.V(1).Info("kkarlon Cluster")
	var cl kkarlonv1.Cluster

	if err := r.Get(ctx, req.NamespacedName, &cl); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("cluster is gone -- ok")
			return ctrl.Result{}, nil
		}
		log.Info(fmt.Sprintf("unable to get cluster (%s) ... requeuing", err))
		return ctrl.Result{Requeue: true}, nil
	}
	// Initialize the patch helper. It stores a "before" copy of the current object.
	patchHelper, err := patch.NewHelper(&cl, r.Client)
	if err != nil {
		log.Error(err, "Failed to configure the patch helper")
		return ctrl.Result{Requeue: true}, nil
	}

	conn, appIf, err := r.ArgocdClient.NewApplicationClient()
	if err != nil {
		msg := fmt.Sprintf("failed to get argocd application client: %s", err)
		return r.UpdateState(ctx, log, &cl, "retrying", msg, retryDelayAsResult)
	}
	defer io.Close(conn)

	if !cl.ObjectMeta.DeletionTimestamp.IsZero() {
		// Handle deletion reconciliation loop.
		return r.reconcileDelete(ctx, log, &cl, patchHelper, appIf)
	}

	// Add finalizer first if not exist to avoid the race condition between init and delete
	if !controllerutil.ContainsFinalizer(&cl, kkarlonv1.ClusterFinalizer) {
		controllerutil.AddFinalizer(&cl, kkarlonv1.ClusterFinalizer)
		// patch and return right away instead of reusing the main defer,
		// because the main defer may take too much time to get cluster status
		// Patch ObservedGeneration only if the reconciliation completed successfully
		patchOpts := []patch.Option{patch.WithStatusObservedGeneration{}}
		if err := patchHelper.Patch(ctx, &cl, patchOpts...); err != nil {
			log.Error(err, "Failed to patch cluster to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	ctmpl := &cl.Spec.ClusterTemplate
	repoUrl := ctmpl.Url
	repoRevision := ctmpl.Revision
	repoPath := ctmpl.Path
	if cl.Status.InnerClusterName == "" {
		log.Info("validating cluster template ...")
		_, creds, err := argocd.GetKubeclientAndRepoCreds(r.Config, r.ArgoCdNs,
			repoUrl)
		if err != nil {
			msg := fmt.Sprintf("failed to get repo creds: %s", err)
			return r.UpdateState(ctx, log, &cl, "retrying", msg, retryDelayAsResult)
		}
		innerClusterName, err := bcl.ValidateGitDir(creds, repoUrl, repoRevision, repoPath)
		if err != nil {
			msg := fmt.Sprintf("failed to validate cluster template: %s", err)
			return r.UpdateState(ctx, log, &cl, "retrying", msg, retryDelayAsResult)
		}
		cl.Status.InnerClusterName = innerClusterName
		return r.UpdateState(ctx, log, &cl, "template-validated",
			"cluster template validation successful", ctrl.Result{})
	}

	ovr := cl.Spec.Override
	overridden := ovr != nil
	if overridden {
		if !cl.Status.OverrideSuccessful {
			// Handle override
			err = cluster.CreatePatchDir(r.Config, cl.Name, ovr.Repo.Url, r.ArgoCdNs,
				ovr.Repo.Path, ovr.Repo.Revision,
				repoRevision, []byte(ovr.Patch), repoUrl, repoPath)
			if err != nil {
				msg := fmt.Sprintf("failed to create override patch in git: %s", err)
				return r.UpdateState(ctx, log, &cl, "retrying", msg, retryDelayAsResult)
			}
			cl.Status.OverrideSuccessful = true
			return r.UpdateState(ctx, log, &cl, "override-created",
				"override patch creation successful", ctrl.Result{})
		}
		// Point the cluster to the override instead of cluster template
		repoUrl = ovr.Repo.Url
		repoRevision = ovr.Repo.Revision
		repoPath = ovr.Repo.Path
	}

	// Check if kkarlon app already exists
	aan := kkarlonAppName(cl.Name)
	_, err = appIf.Get(ctx, &argoapp.ApplicationQuery{Name: &aan})
	if err != nil {
		grpcStatus, ok := grpcstatus.FromError(err)
		if !ok {
			return r.UpdateState(ctx, log, &cl, "retrying",
				"failed to get grpc status from argocd API", retryDelayAsResult)
		}
		if grpcStatus.Code() != grpccodes.NotFound {
			return r.UpdateState(ctx, log, &cl, "retrying",
				fmt.Sprintf("unexpected grpc status: %d", grpcStatus.Code()),
				retryDelayAsResult)
		}
		casMgmtClusterHost := ""
		innerClusterName := cl.Status.InnerClusterName
		gen2CASEnabled := cl.Spec.Autoscaler != nil
		if gen2CASEnabled {
			casMgmtClusterHost = cl.Spec.Autoscaler.MgmtClusterHost
		}
		kkarlonHelmChart := cl.Spec.ArlonHelmChart
		if kkarlonHelmChart == nil {
			kkarlonHelmChart = &defaultArlonChart
		}
		_, err = cluster.Create(appIf, r.Config, r.ArgoCdNs, r.ArlonNs,
			cl.Name, innerClusterName, kkarlonHelmChart.Url, kkarlonHelmChart.Revision,
			kkarlonHelmChart.Path, "",
			nil, true, casMgmtClusterHost, gen2CASEnabled)
		if err != nil {
			msg := fmt.Sprintf("failed to create kkarlon application: %s", err)
			return r.UpdateState(ctx, log, &cl, "retrying", msg, retryDelayAsResult)
		}
	}
	// Check if cluster app already exists
	clusterApp, err := appIf.Get(ctx, &argoapp.ApplicationQuery{Name: &cl.Name})
	if err != nil {
		grpcStatus, ok := grpcstatus.FromError(err)
		if !ok {
			return r.UpdateState(ctx, log, &cl, "retrying",
				"failed to get grpc status from argocd API", retryDelayAsResult)
		}
		if grpcStatus.Code() != grpccodes.NotFound {
			return r.UpdateState(ctx, log, &cl, "retrying",
				fmt.Sprintf("unexpected grpc status: %d", grpcStatus.Code()),
				retryDelayAsResult)
		}
		// Create cluster app
		_, err = cluster.CreateClusterApp(appIf, r.ArgoCdNs,
			cl.Name, cl.Status.InnerClusterName, repoUrl, repoRevision,
			repoPath, true, overridden)
		if err != nil {
			msg := fmt.Sprintf("failed to create cluster application: %s", err)
			return r.UpdateState(ctx, log, &cl, "retrying", msg, retryDelayAsResult)
		}
		return r.UpdateState(ctx, log, &cl, "created",
			"cluster app creation successful", ctrl.Result{})
	}

	// Sync profile annotation from Cluster to cluster app if necessary
	sync := false
	if cl.Annotations != nil && (clusterApp.Annotations == nil ||
		cl.Annotations[kkarlonapp.ProfilesAnnotationKey] !=
			clusterApp.Annotations[kkarlonapp.ProfilesAnnotationKey]) {
		if clusterApp.Annotations == nil {
			clusterApp.Annotations = make(map[string]string)
		}
		clusterApp.Annotations[kkarlonapp.ProfilesAnnotationKey] = cl.Annotations[kkarlonapp.ProfilesAnnotationKey]
		sync = true
	} else if cl.Annotations == nil && clusterApp.Annotations != nil {
		clusterApp.Annotations = nil
		sync = true
	}
	if sync {
		log.Info("updating profiles annotation of cluster app")
		_, err = appIf.Update(ctx, &argoapp.ApplicationUpdateRequest{
			Application: clusterApp,
		})
		if err != nil {
			msg := fmt.Sprintf("failed to update cluster application: %s", err)
			return r.UpdateState(ctx, log, &cl, "retrying", msg, retryDelayAsResult)
		}
	}
	if cl.Status.State != "created" {
		return r.UpdateState(ctx, log, &cl, "created",
			"cluster app already exists but state needs updating -- ok", ctrl.Result{})
	}
	return ctrl.Result{}, nil
}

func (r *ClusterReconciler) UpdateState(
	ctx context.Context,
	log logr.Logger,
	cr *kkarlonv1.Cluster,
	state string,
	msg string,
	result ctrl.Result,
) (ctrl.Result, error) {
	cr.Status.State = state
	cr.Status.Message = msg
	log.Info(fmt.Sprintf("%s ... setting state to '%s'", msg, cr.Status.State))
	if err := r.Status().Update(ctx, cr); err != nil {
		log.Error(err, "unable to update clusterregistration status")
		return ctrl.Result{}, err
	}
	return result, nil
}

func (r *ClusterReconciler) reconcileDelete(
	ctx context.Context,
	log logr.Logger,
	cr *kkarlonv1.Cluster,
	patchHelper *patch.Helper,
	appIf argoapp.ApplicationServiceClient,
) (ctrl.Result, error) {
	// Check if cluster app exists
	clusterApp, err := appIf.Get(ctx, &argoapp.ApplicationQuery{Name: &cr.Name})
	if err == nil {
		// Delete override if necessary
		if cr.Spec.Override != nil && cr.Status.OverrideSuccessful {
			kubeClient, err := kubernetes.NewForConfig(r.Config)
			if err != nil {
				msg := fmt.Sprintf("failed to get kubeclient: %s", err)
				return r.UpdateState(ctx, log, cr, "error-deleting-override",
					msg, retryDelayAsResult)
			}
			err = cluster.DeleteOverridesDir(clusterApp, kubeClient, r.ArgoCdNs,
				clusterApp.Name)
			if err != nil {
				msg := fmt.Sprintf("failed to delete overrides directory: %s", err)
				return r.UpdateState(ctx, log, cr, "error-deleting-override",
					msg, retryDelayAsResult)
			}
			// Flip this flag to indicate override no longer needs deletion
			cr.Status.OverrideSuccessful = false
			return r.UpdateState(ctx, log, cr, "override-deleted",
				"override deleted successfully", ctrl.Result{})
		}

		if !clusterApp.DeletionTimestamp.IsZero() {
			log.Info("cluster app deletion already pending -- will check again later")
			return retryDelayAsResult, nil
		}
		// Delete it
		cascade := true
		_, err = appIf.Delete(ctx, &argoapp.ApplicationDeleteRequest{
			Name:    &cr.Name,
			Cascade: &cascade,
		})
		if err != nil {
			msg := fmt.Sprintf("failed to delete cluster app: %s", err)
			return r.UpdateState(ctx, log, cr, "error-deleting-cluster-app",
				msg, retryDelayAsResult)
		}
		return r.UpdateState(ctx, log, cr, "deleting-cluster-app",
			"deleting cluster app", ctrl.Result{})
	}
	grpcStatus, ok := grpcstatus.FromError(err)
	if !ok {
		return r.UpdateState(ctx, log, cr, "delete-retrying",
			"failed to get grpc status from argocd API", retryDelayAsResult)
	}
	if grpcStatus.Code() != grpccodes.NotFound {
		return r.UpdateState(ctx, log, cr, "delete-retrying",
			fmt.Sprintf("unexpected grpc status: %d", grpcStatus.Code()),
			retryDelayAsResult)
	}

	// Check if kkarlon app already exists
	aan := kkarlonAppName(cr.Name)
	kkarlonApp, err := appIf.Get(ctx, &argoapp.ApplicationQuery{Name: &aan})
	if err == nil {
		if !kkarlonApp.DeletionTimestamp.IsZero() {
			log.Info("kkarlon app deletion already pending -- will check again later")
			return retryDelayAsResult, nil
		}
		// Delete it
		cascade := true
		_, err = appIf.Delete(ctx, &argoapp.ApplicationDeleteRequest{
			Name:    &aan,
			Cascade: &cascade,
		})
		if err != nil {
			msg := fmt.Sprintf("failed to delete kkarlon app: %s", err)
			return r.UpdateState(ctx, log, cr, "error-deleting-kkarlon-app",
				msg, retryDelayAsResult)
		}
		return r.UpdateState(ctx, log, cr, "deleting-kkarlon-app",
			"deleting kkarlon app", ctrl.Result{})
	}
	grpcStatus, ok = grpcstatus.FromError(err)
	if !ok {
		return r.UpdateState(ctx, log, cr, "delete-retrying",
			"failed to get grpc status from argocd API", retryDelayAsResult)
	}
	if grpcStatus.Code() != grpccodes.NotFound {
		return r.UpdateState(ctx, log, cr, "delete-retrying",
			fmt.Sprintf("unexpected grpc status: %d", grpcStatus.Code()),
			retryDelayAsResult)
	}
	controllerutil.RemoveFinalizer(cr, kkarlonv1.ClusterFinalizer)
	if err := patchHelper.Patch(ctx, cr); err != nil {
		log.Info(fmt.Sprintf("failed to remove finalizer from cluster: %s", err))
		return ctrl.Result{}, err
	}
	log.Info(fmt.Sprintf("removed finalizer from cluster '%s'",
		cr.Name))
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Cluster{}).
		Complete(r)
}

func kkarlonAppName(clusterName string) string {
	return fmt.Sprintf("%s-kkarlon", clusterName)
}
