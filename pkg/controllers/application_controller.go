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
	argoclient "github.com/argoproj/argo-cd/v2/pkg/apiclient"
	argoapp "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/jayanth-tjvrr/karlon/pkg/appprofile"
	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ApplicationReconciler reconciles a AppProfile object
type ApplicationReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	ArgocdClient apiclient.Client
}

//+kubebuilder:rbac:groups=core.kkarlon.io,resources=appprofiles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.kkarlon.io,resources=appprofiles/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.kkarlon.io,resources=appprofiles/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AppProfile object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *ApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	return reconcileApplication(ctx, r.Client, r.ArgocdClient, req, logger)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&argoapp.Application{}).
		Complete(r)
}

func reconcileApplication(
	ctx context.Context,
	cli client.Client,
	argocli argoclient.Client,
	req ctrl.Request,
	log logr.Logger,
) (ctrl.Result, error) {
	var app argoapp.Application

	if err := cli.Get(ctx, req.NamespacedName, &app); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("application is gone -- ok")
			// no further reconciliation is necessary
			return ctrl.Result{}, nil
		} else {
			log.Info(fmt.Sprintf("unable to get application (%s) ... requeuing", err))
			return ctrl.Result{Requeue: true}, nil
		}
	}
	if app.Labels == nil || app.Labels["kkarlon-type"] != "cluster-app" {
		log.V(1).Info("application is not an kkarlon cluster, skipping...")
		return ctrl.Result{}, nil
	}
	log.V(1).Info("reconciling application implementing kkarlon cluster")
	return appprofile.ReconcileEverything(ctx, cli, argocli, log)
}
