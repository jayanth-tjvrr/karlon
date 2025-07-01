package appprofile

import (
	"context"
	"fmt"

	argoclient "github.com/argoproj/argo-cd/v2/pkg/apiclient"
	argoapp "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	clusterpkg "github.com/argoproj/argo-cd/v2/pkg/apiclient/cluster"
	argoappapi "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"

	//appset "github.com/argoproj/argo-cd/v2/pkg/apis/applicationset/v1alpha1"
	"strings"
	"sync"

	kkarlonv1 "github.com/jayanth-tjvrr/karlon/pkg/api/v1"
	kkarlonapp "github.com/jayanth-tjvrr/karlon/pkg/app"
	kkarlonclusters "github.com/jayanth-tjvrr/karlon/pkg/cluster"
	sets "github.com/deckarep/golang-set/v2"
	"github.com/go-logr/logr"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/json"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	mtx sync.Mutex
)

func Reconcile(
	ctx context.Context,
	cli client.Client,
	argocli argoclient.Client,
	req ctrl.Request,
	log logr.Logger,
) (ctrl.Result, error) {
	log.Info("reconciling kkarlon appprofile")
	var prof kkarlonv1.AppProfile

	if err := cli.Get(ctx, req.NamespacedName, &prof); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("appprofile is gone -- ok")
		} else {
			log.Info(fmt.Sprintf("unable to get appprofile (%s) ... requeuing", err))
			return ctrl.Result{Requeue: true}, nil
		}
	}
	return ReconcileEverything(ctx, cli, argocli, log)
}

func ReconcileEverything(
	ctx context.Context,
	cli client.Client,
	argocli argoclient.Client,
	log logr.Logger,
) (ctrl.Result, error) {
	mtx.Lock()
	defer mtx.Unlock()
	log.V(1).Info("--- global reconciliation begin ---")
	// Get ArgoCD clusters
	conn, clApi, err := argocli.NewClusterClient()
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get argocd clusters client: %s", err)
	}
	defer conn.Close()
	argoClusters, err := clApi.List(ctx, &clusterpkg.ClusterQuery{})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to list argocd clusters: %s", err)
	}

	// Get kkarlon clusters (argocd applications)
	conn2, appApi, err := argocli.NewApplicationClient()
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get argocd application client: %s", err)
	}
	defer conn2.Close()
	query := kkarlonclusters.ArlonGen2ClusterLabelQueryOnArgoApps
	kkarlonClusters, err := appApi.List(ctx, &argoapp.ApplicationQuery{Selector: &query})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to list argocd applications: %s", err)
	}
	kkarlonClusterMap := make(map[string]argoappapi.Application)
	for _, kkarlonClust := range kkarlonClusters.Items {
		kkarlonClusterMap[kkarlonClust.Name] = kkarlonClust
	}

	// Get applications (applicationsets managed by Arlon)
	var appList argoappapi.ApplicationSetList
	rqmt, err := labels.NewRequirement("kkarlon-type", selection.In, []string{"application"})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create requirement: %s", err)
	}
	sel := labels.NewSelector().Add(*rqmt)
	err = cli.List(ctx, &appList, &client.ListOptions{
		Namespace:     "argocd",
		LabelSelector: sel,
	})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to list applicationsets: %s", err)
	}
	validAppNames := sets.NewSet[string]()
	for _, item := range appList.Items {
		validAppNames.Add(item.Name)
	}
	log.V(1).Info("apps counted", "count", len(appList.Items))

	// Get profiles
	var profList kkarlonv1.AppProfileList
	err = cli.List(ctx, &profList)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to list app profiles: %s", err)
	}
	log.V(1).Info("app profiles counted", "count", len(profList.Items))

	// Reconcile clusters
	profileToClusters := make(map[string]sets.Set[string])
	clustNameToServer := make(map[string]string)
	for _, argoClust := range argoClusters.Items {
		if argoClust.Annotations == nil {
			argoClust.Annotations = make(map[string]string)
		}
		clustNameToServer[argoClust.Name] = argoClust.Server
		argoClustAnnotation := argoClust.Annotations[kkarlonapp.ProfilesAnnotationKey]
		dirty := false
		kkarlonClust, ok := kkarlonClusterMap[argoClust.Name]
		if !ok {
			// No corresponding kkarlon cluster. Could be an "external" cluster,
			// so allow the annotation to be managed independently.
			log.V(1).Info("argo cluster has no corresponding kkarlon cluster, skipping",
				"argoClusterName", argoClust.Name)
			updateProfileToClustersMap(argoClust.Name, argoClustAnnotation, profileToClusters)
			continue
		}
		// Arlon cluster exists. Ensure argocd cluster is annotationed identically
		if kkarlonClust.Annotations == nil {
			kkarlonClust.Annotations = make(map[string]string)
		}
		kkarlonClustAnnotation := kkarlonClust.Annotations[kkarlonapp.ProfilesAnnotationKey]
		if kkarlonClustAnnotation != "" {
			// Arlon cluster has annotation
			if argoClustAnnotation != kkarlonClustAnnotation {
				log.Info("updating annotation on argo cluster to match kkarlon cluster's",
					"clustName", kkarlonClust.Name,
					"annotationValue", kkarlonClustAnnotation)
				argoClust.Annotations[kkarlonapp.ProfilesAnnotationKey] = kkarlonClustAnnotation
				dirty = true
			} else {
				log.V(1).Info("argo and kkarlon clusters already in sync",
					"clustName", kkarlonClust.Name)
			}
		} else if argoClustAnnotation != "" {
			// Arlon cluster has no annotation but argo cluster has one
			log.Info("removing annotation from argo cluster because kkarlon cluster has none",
				"argoClusterName", argoClust.Name)
			delete(argoClust.Annotations, kkarlonapp.ProfilesAnnotationKey)
			dirty = true
		} else {
			log.V(1).Info("argo & kkarlon cluster have no annotation, skipping",
				"clustName", kkarlonClust.Name)
			continue
		}
		if dirty {
			_, err = clApi.Update(context.Background(), &clusterpkg.ClusterUpdateRequest{
				Cluster:       &argoClust,
				UpdatedFields: []string{"annotations"},
			})
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to update argo cluster: %s", err)
			}
		}
		updateProfileToClustersMap(kkarlonClust.Name, kkarlonClustAnnotation, profileToClusters)
	}

	// Reconcile profiles
	profNames := sets.NewSet[string]()
	appToClusters := make(map[string]sets.Set[string])
	for _, prof := range profList.Items {
		profNames.Add(prof.Name)
		dirty := false
		beforeInvalidNames := sets.NewSet[string](prof.Status.InvalidAppNames...)
		afterInvalidNames := sets.NewSet[string]()
		clustersUsingThisProfile := profileToClusters[prof.Name]
		for _, appName := range prof.Spec.AppNames {
			if !validAppNames.Contains(appName) {
				afterInvalidNames.Add(appName)
			} else {
				if appToClusters[appName] == nil {
					appToClusters[appName] = sets.NewSet[string]()
				}
				if clustersUsingThisProfile != nil {
					// Add cluster set to this app's overall cluster set
					appToClusters[appName] = appToClusters[appName].Union(clustersUsingThisProfile)
				}
			}
		}
		if !beforeInvalidNames.Equal(afterInvalidNames) {
			prof.Status.InvalidAppNames = afterInvalidNames.ToSlice()
			dirty = true
		}
		beforeHealth := prof.Status.Health
		var afterHealth string
		if len(prof.Status.InvalidAppNames) > 0 {
			afterHealth = "degraded"
		} else {
			afterHealth = "healthy"
		}
		if beforeHealth != afterHealth {
			prof.Status.Health = afterHealth
			dirty = true
		}
		if dirty {
			// update profile status
			log.Info("updating app profile", "profileName", prof.Name)
			err = cli.Status().Update(ctx, &prof)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to update app profile: %s", err)
			}
		}
	}

	// Reconcile apps
	for _, app := range appList.Items {
		if app.Spec.Generators == nil || len(app.Spec.Generators) != 1 {
			log.Info("invalid application set, has no generators",
				"appSetName", app.Name)
			continue
		}
		clustGen := app.Spec.Generators[0].List
		if clustGen == nil {
			log.Info("invalid application set, generator is not of type 'list'",
				"appSetName", app.Name)
			continue
		}
		elems := clustGen.Elements
		beforeClusters := sets.NewSet[string]()
		for _, elem := range elems {
			var element map[string]interface{}
			err = json.Unmarshal(elem.Raw, &element)
			if err != nil {
				log.Error(err, "error decoding json element", "appSetName", app.Name)
				break
			}
			for key, value := range element {
				if key == "cluster_name" {
					clust, ok := value.(string)
					if !ok {
						log.Info("value of cluster key is not a string", "appSetName", app.Name)
						continue
					}
					beforeClusters.Add(clust)
				}
			}
		}
		afterClusters := appToClusters[app.Name]
		if afterClusters == nil {
			afterClusters = sets.NewSet[string]()
		}
		if afterClusters.Equal(beforeClusters) {
			continue // no update needed
		}
		// Update applicationset's generator with new element list
		newElems := []apiextensionsv1.JSON{}
		for clustName := range afterClusters.Iter() {
			jsonStr := fmt.Sprintf(`{"cluster_name":"%s", "cluster_server":"%s"}`,
				clustName, clustNameToServer[clustName])
			newElems = append(newElems, apiextensionsv1.JSON{Raw: []byte(jsonStr)})
		}
		app.Spec.Generators[0].List.Elements = newElems
		log.Info("updating app's list generator's elements",
			"app", app.Name, "elements", newElems)
		err = cli.Update(ctx, &app)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update applicationset: %s", err)
		}
	}
	log.V(1).Info("--- global reconciliation end ---")
	return ctrl.Result{}, nil
}

func updateProfileToClustersMap(
	clustName string,
	commaSeparatedProfileNames string,
	profileToClusters map[string]sets.Set[string],
) {
	if commaSeparatedProfileNames == "" {
		return
	}
	profiles := strings.Split(commaSeparatedProfileNames, ",")
	for _, profile := range profiles {
		if profileToClusters[profile] == nil {
			profileToClusters[profile] = sets.NewSet[string]()
		}
		profileToClusters[profile].Add(clustName)
	}
}
