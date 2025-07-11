package appprofile

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	argoclient "github.com/argoproj/argo-cd/v2/pkg/apiclient"
	applicationpkg "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	clusterpkg "github.com/argoproj/argo-cd/v2/pkg/apiclient/cluster"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"

	argoapp "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	kkarlonv1 "github.com/jayanth-tjvrr/karlon/pkg/api/v1"
	kkarlonapp "github.com/jayanth-tjvrr/karlon/pkg/app"
	sets "github.com/deckarep/golang-set/v2"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type mockArgoClient struct {
	argoclient.Client
}

type mockIoCloser struct {
	io.Closer
}

func (mic *mockIoCloser) Close() error {
	return nil
}

type mockClusterSvcClient struct {
	clusterpkg.ClusterServiceClient
}

func (mac *mockArgoClient) NewClusterClient() (io.Closer, clusterpkg.ClusterServiceClient, error) {
	return &mockIoCloser{}, &mockClusterSvcClient{}, nil
}

type mockApplicationSvcClient struct {
	applicationpkg.ApplicationServiceClient
}

func (mac *mockArgoClient) NewApplicationClient() (io.Closer, applicationpkg.ApplicationServiceClient, error) {
	return &mockIoCloser{}, &mockApplicationSvcClient{}, nil
}

var (
	gClusterList        *v1alpha1.ClusterList
	gApplicationList    *v1alpha1.ApplicationList
	gApplicationSetList *argoapp.ApplicationSetList
	gProfileList        *kkarlonv1.AppProfileList
)

func init() {
	// ArgoCD clusters
	gClusterList = &v1alpha1.ClusterList{
		Items: []v1alpha1.Cluster{
			{
				Name:   "kkarlon-cluster-1",
				Server: "kkarlon-cluster-1.local",
			},
			{
				Name:   "external-cluster",
				Server: "external-cluster.local",
			},
			{
				Name:   "kkarlon-cluster-2",
				Server: "kkarlon-cluster-2.local",
				Annotations: map[string]string{
					"kkarlon.io/profiles": "marketing,qa",
				},
			},
			{
				Name:   "kkarlon-cluster-3",
				Server: "kkarlon-cluster-3.local",
				Annotations: map[string]string{
					"kkarlon.io/profiles": "engineering,marketing",
				},
			},
		},
	}

	// applications representing Arlon clusters
	gApplicationList = &v1alpha1.ApplicationList{
		Items: []v1alpha1.Application{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kkarlon-cluster-2",
					Labels: map[string]string{
						"kkarlon-type": "cluster-app",
						"managed-by": "kkarlon",
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kkarlon-cluster-1",
					Labels: map[string]string{
						"kkarlon-type": "cluster-app",
						"managed-by": "kkarlon",
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kkarlon-cluster-3",
					Labels: map[string]string{
						"kkarlon-type": "cluster-app",
						"managed-by": "kkarlon",
					},
				},
			},
		},
	}

	// ApplicationSets representing Arlon apps
	gApplicationSetList = &argoapp.ApplicationSetList{
		Items: []argoapp.ApplicationSet{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "wordpress",
					Labels: map[string]string{
						"kkarlon-type": "application",
					},
				},
				Spec: argoapp.ApplicationSetSpec{
					Generators: []argoapp.ApplicationSetGenerator{
						{
							List: &argoapp.ListGenerator{},
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mysql",
					Labels: map[string]string{
						"kkarlon-type": "application",
					},
				},
				Spec: argoapp.ApplicationSetSpec{
					Generators: []argoapp.ApplicationSetGenerator{
						{
							List: &argoapp.ListGenerator{},
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "autocad",
					Labels: map[string]string{
						"kkarlon-type": "application",
					},
				},
				Spec: argoapp.ApplicationSetSpec{
					Generators: []argoapp.ApplicationSetGenerator{
						{
							List: &argoapp.ListGenerator{},
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "teamcity",
					Labels: map[string]string{
						"kkarlon-type": "application",
					},
				},
				Spec: argoapp.ApplicationSetSpec{
					Generators: []argoapp.ApplicationSetGenerator{
						{
							List: &argoapp.ListGenerator{},
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "app-not-used-anywhere",
					Labels: map[string]string{
						"kkarlon-type": "application",
					},
				},
				Spec: argoapp.ApplicationSetSpec{
					Generators: []argoapp.ApplicationSetGenerator{
						{
							List: &argoapp.ListGenerator{},
						},
					},
				},
			},
		},
	}

	// Arlon App Profiles
	gProfileList = &kkarlonv1.AppProfileList{
		Items: []kkarlonv1.AppProfile{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "marketing",
				},
				Spec: kkarlonv1.AppProfileSpec{
					AppNames: []string{
						"wordpress",
						"nonexistent-1",
						"mysql",
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "engineering",
				},
				Spec: kkarlonv1.AppProfileSpec{
					AppNames: []string{
						"mysql",
						"nonexistent-2",
						"autocad",
						"nonexistent-1",
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "qa",
				},
				Spec: kkarlonv1.AppProfileSpec{
					AppNames: []string{
						"mysql",
						"teamcity",
					},
				},
			},
		},
	}
}

func (mcsc *mockClusterSvcClient) List(ctx context.Context,
	in *clusterpkg.ClusterQuery,
	opts ...grpc.CallOption) (*v1alpha1.ClusterList, error) {
	return gClusterList, nil
}

func (mcsc *mockClusterSvcClient) Update(ctx context.Context,
	in *clusterpkg.ClusterUpdateRequest,
	opts ...grpc.CallOption) (*v1alpha1.Cluster, error) {
	clust := lookupArgoCluster(in.Cluster.Name)
	if clust == nil {
		return nil, fmt.Errorf("cluster not found")
	}
	*clust = *in.Cluster
	return nil, nil
}

func (masc *mockApplicationSvcClient) List(ctx context.Context,
	in *applicationpkg.ApplicationQuery,
	opts ...grpc.CallOption) (*v1alpha1.ApplicationList, error) {
	return gApplicationList, nil
}

type mockCtrlRuntClient struct {
	client.Client
}

func (mcrc *mockCtrlRuntClient) List(ctx context.Context,
	list client.ObjectList, opts ...client.ListOption) error {
	if appSetListPtr, ok := list.(*argoapp.ApplicationSetList); ok {
		*appSetListPtr = *gApplicationSetList
	} else {
		profileListPtr := list.(*kkarlonv1.AppProfileList)
		*profileListPtr = *gProfileList
	}
	return nil
}

type mockStatusWriter struct {
	client.StatusWriter
}

func (mcrc *mockCtrlRuntClient) Update(ctx context.Context,
	obj client.Object, opts ...client.UpdateOption) error {
	pAppSet, ok := obj.(*argoapp.ApplicationSet)
	if !ok {
		return fmt.Errorf("can't update any object type other than ApplicationSet")
	}
	pCurrent := lookupApplicationSet(pAppSet.Name)
	if pCurrent == nil {
		return fmt.Errorf("no application set with name %s", pAppSet.Name)
	}
	*pCurrent = *pAppSet
	return nil
}

func (mcrc *mockCtrlRuntClient) Status() client.StatusWriter {
	return &mockStatusWriter{}
}

func (msw *mockStatusWriter) Update(ctx context.Context, obj client.Object,
	opts ...client.UpdateOption) error {
	if pProfile, ok := obj.(*kkarlonv1.AppProfile); ok {
		prof := lookupProfile(pProfile.Name)
		if prof == nil {
			return fmt.Errorf("failed to find profile named %s", pProfile.Name)
		}
		*prof = *pProfile
		return nil
	}
	return fmt.Errorf("updating object of that type not supported")
}

func lookupProfile(name string) *kkarlonv1.AppProfile {
	for i, prof := range gProfileList.Items {
		if prof.Name == name {
			return &gProfileList.Items[i]
		}
	}
	return nil
}

func lookupArgoCluster(name string) *v1alpha1.Cluster {
	for i, clust := range gClusterList.Items {
		if clust.Name == name {
			return &gClusterList.Items[i]
		}
	}
	return nil
}

func lookupApplicationSet(name string) *argoapp.ApplicationSet {
	for i, aps := range gApplicationSetList.Items {
		if aps.Name == name {
			return &gApplicationSetList.Items[i]
		}
	}
	return nil
}

func TestAppProfileReconcileEverything(t *testing.T) {
	log := zap.New(zap.UseFlagOptions(&zap.Options{
		Development: true,
		TimeEncoder: zapcore.RFC3339NanoTimeEncoder,
	}))
	var mcr *mockCtrlRuntClient
	var mac *mockArgoClient

	reconcile(t, mcr, mac, log)
	assert.Equal(t, gProfileList.Items[0].Status.Health, "degraded")
	assert.True(t, stringSetsEqual(gProfileList.Items[0].Status.InvalidAppNames, []string{"nonexistent-1"}))
	assert.Equal(t, gProfileList.Items[1].Status.Health, "degraded")
	assert.True(t, stringSetsEqual(gProfileList.Items[1].Status.InvalidAppNames, []string{"nonexistent-1", "nonexistent-2"}))
	assert.Equal(t, gProfileList.Items[2].Status.Health, "healthy")
	// annotation was removed from clusters 2 and 3 because corresponding
	// kkarlon cluster had none
	assert.True(t, argoClusterHasProfiles(t, "kkarlon-cluster-2", nil))
	assert.True(t, argoClusterHasProfiles(t, "kkarlon-cluster-3", nil))

	// annotate kkarlon cluster 1
	annotateArlonCluster(t, "kkarlon-cluster-1", "foo,marketing")
	reconcile(t, mcr, mac, log)
	assert.True(t, argoClusterHasProfiles(t, "kkarlon-cluster-1", []string{"marketing", "foo"}))
	assert.True(t, kkarlonAppTargetsTheseClusters(t, "wordpress", []string{"kkarlon-cluster-1"}))
	assert.True(t, kkarlonAppTargetsTheseClusters(t, "mysql", []string{"kkarlon-cluster-1"}))
	assert.True(t, kkarlonAppTargetsTheseClusters(t, "autocad", []string{}))
	assert.True(t, kkarlonAppTargetsTheseClusters(t, "teamcity", []string{}))

	// add engineering to kkarlon cluster 1, qa to kkarlon cluster 2
	annotateArlonCluster(t, "kkarlon-cluster-1", "marketing,foo,engineering")
	annotateArlonCluster(t, "kkarlon-cluster-2", "qa")
	reconcile(t, mcr, mac, log)
	assert.True(t, argoClusterHasProfiles(t, "kkarlon-cluster-1", []string{"engineering", "marketing", "foo"}))
	assert.True(t, argoClusterHasProfiles(t, "kkarlon-cluster-2", []string{"qa"}))
	assert.True(t, kkarlonAppTargetsTheseClusters(t, "wordpress", []string{"kkarlon-cluster-1"}))
	assert.True(t, kkarlonAppTargetsTheseClusters(t, "mysql", []string{"kkarlon-cluster-2", "kkarlon-cluster-1"}))
	assert.True(t, kkarlonAppTargetsTheseClusters(t, "autocad", []string{"kkarlon-cluster-1"}))
	assert.True(t, kkarlonAppTargetsTheseClusters(t, "teamcity", []string{"kkarlon-cluster-2"}))

	// add teamcity to engineering and remove mysql from it
	gProfileList.Items[1].Spec.AppNames = []string{"teamcity", "autocad"}
	reconcile(t, mcr, mac, log)
	assert.True(t, argoClusterHasProfiles(t, "kkarlon-cluster-1", []string{"engineering", "marketing", "foo"}))
	assert.True(t, argoClusterHasProfiles(t, "kkarlon-cluster-2", []string{"qa"}))
	assert.True(t, kkarlonAppTargetsTheseClusters(t, "wordpress", []string{"kkarlon-cluster-1"}))
	assert.True(t, kkarlonAppTargetsTheseClusters(t, "mysql", []string{"kkarlon-cluster-2", "kkarlon-cluster-1"}))
	assert.True(t, kkarlonAppTargetsTheseClusters(t, "autocad", []string{"kkarlon-cluster-1"}))
	assert.True(t, kkarlonAppTargetsTheseClusters(t, "teamcity", []string{"kkarlon-cluster-2", "kkarlon-cluster-1"}))

	// remove all profiles from cluster 1, and attach engineering to cluster 3
	annotateArlonCluster(t, "kkarlon-cluster-1", "")
	annotateArlonCluster(t, "kkarlon-cluster-3", "engineering")
	reconcile(t, mcr, mac, log)
	assert.True(t, argoClusterHasProfiles(t, "kkarlon-cluster-1", []string{}))
	assert.True(t, argoClusterHasProfiles(t, "kkarlon-cluster-2", []string{"qa"}))
	assert.True(t, argoClusterHasProfiles(t, "kkarlon-cluster-3", []string{"engineering"}))
	assert.True(t, kkarlonAppTargetsTheseClusters(t, "wordpress", []string{}))
	assert.True(t, kkarlonAppTargetsTheseClusters(t, "mysql", []string{"kkarlon-cluster-2"}))
	assert.True(t, kkarlonAppTargetsTheseClusters(t, "autocad", []string{"kkarlon-cluster-3"}))
	assert.True(t, kkarlonAppTargetsTheseClusters(t, "teamcity", []string{"kkarlon-cluster-2", "kkarlon-cluster-3"}))

	// remove all profiles from cluster 2
	annotateArlonCluster(t, "kkarlon-cluster-2", "")
	reconcile(t, mcr, mac, log)
	assert.True(t, argoClusterHasProfiles(t, "kkarlon-cluster-1", []string{}))
	assert.True(t, argoClusterHasProfiles(t, "kkarlon-cluster-2", []string{}))
	assert.True(t, argoClusterHasProfiles(t, "kkarlon-cluster-3", []string{"engineering"}))
	assert.True(t, kkarlonAppTargetsTheseClusters(t, "wordpress", []string{}))
	assert.True(t, kkarlonAppTargetsTheseClusters(t, "mysql", []string{}))
	assert.True(t, kkarlonAppTargetsTheseClusters(t, "autocad", []string{"kkarlon-cluster-3"}))
	assert.True(t, kkarlonAppTargetsTheseClusters(t, "teamcity", []string{"kkarlon-cluster-3"}))

	// annotate external cluster
	annotateArgoCluster(t, "external-cluster", "engineering,qa,marketing")
	reconcile(t, mcr, mac, log)
	assert.True(t, argoClusterHasProfiles(t, "kkarlon-cluster-1", []string{}))
	assert.True(t, argoClusterHasProfiles(t, "kkarlon-cluster-1", []string{}))
	assert.True(t, argoClusterHasProfiles(t, "kkarlon-cluster-2", []string{}))
	assert.True(t, argoClusterHasProfiles(t, "kkarlon-cluster-3", []string{"engineering"}))
	assert.True(t, argoClusterHasProfiles(t, "external-cluster", []string{"marketing", "qa", "engineering"}))
	assert.True(t, kkarlonAppTargetsTheseClusters(t, "wordpress", []string{"external-cluster"}))
	assert.True(t, kkarlonAppTargetsTheseClusters(t, "mysql", []string{"external-cluster"}))
	assert.True(t, kkarlonAppTargetsTheseClusters(t, "autocad", []string{"external-cluster", "kkarlon-cluster-3"}))
	assert.True(t, kkarlonAppTargetsTheseClusters(t, "teamcity", []string{"external-cluster", "kkarlon-cluster-3"}))

	// remove autocad from all profiles
	gProfileList.Items[1].Spec.AppNames = []string{"teamcity"} // engineering
	reconcile(t, mcr, mac, log)
	assert.True(t, argoClusterHasProfiles(t, "kkarlon-cluster-1", []string{}))
	assert.True(t, argoClusterHasProfiles(t, "kkarlon-cluster-1", []string{}))
	assert.True(t, argoClusterHasProfiles(t, "kkarlon-cluster-2", []string{}))
	assert.True(t, argoClusterHasProfiles(t, "kkarlon-cluster-3", []string{"engineering"}))
	assert.True(t, argoClusterHasProfiles(t, "external-cluster", []string{"marketing", "qa", "engineering"}))
	assert.True(t, kkarlonAppTargetsTheseClusters(t, "wordpress", []string{"external-cluster"}))
	assert.True(t, kkarlonAppTargetsTheseClusters(t, "mysql", []string{"external-cluster"}))
	assert.True(t, kkarlonAppTargetsTheseClusters(t, "autocad", []string{}))
	assert.True(t, kkarlonAppTargetsTheseClusters(t, "teamcity", []string{"external-cluster", "kkarlon-cluster-3"}))

	// remove all apps, ensure invalidAppNames correctly updated
	gApplicationSetList.Items = nil
	reconcile(t, mcr, mac, log)
	ensureProfileInvalidApps(t, "marketing", []string{"wordpress", "nonexistent-1", "mysql"})
	ensureProfileInvalidApps(t, "engineering", []string{"teamcity"})
	ensureProfileInvalidApps(t, "qa", []string{"mysql", "teamcity"})
	dumpProfiles(t)
	dumpClusters(t)
	dumpApplicationSets(t)
}

func reconcile(t *testing.T, mcr *mockCtrlRuntClient, mac *mockArgoClient, log logr.Logger) {
	_, err := ReconcileEverything(context.TODO(), mcr, mac, log)
	if err != nil {
		t.Fatalf("reconcile error: %s", err)
	}
}

func dumpProfiles(t *testing.T) {
	for _, prof := range gProfileList.Items {
		t.Log("profile:", prof)
	}
}

func dumpClusters(t *testing.T) {
	for _, cluster := range gClusterList.Items {
		t.Log("cluster:", cluster)
	}
}

func dumpApplicationSets(t *testing.T) {
	for _, a := range gApplicationSetList.Items {
		t.Log("applicationset:", a)
	}
}

func argoClusterHasProfiles(t *testing.T, clustName string, profiles []string) bool {
	specifiedSet := sets.NewSet[string](profiles...)
	actualSet := sets.NewSet[string]()
	for i, clust := range gClusterList.Items {
		if clust.Name == clustName {
			ann := gClusterList.Items[i].Annotations
			if ann != nil && ann[kkarlonapp.ProfilesAnnotationKey] != "" {
				profNames := strings.Split(ann[kkarlonapp.ProfilesAnnotationKey], ",")
				for _, profName := range profNames {
					actualSet.Add(profName)
				}
			}
			return actualSet.Equal(specifiedSet)
		}
	}
	t.Errorf("failed to find argocd cluster with name %s", clustName)
	return false
}

func annotateArlonCluster(t *testing.T, clustName string, commaSeparatedProfiles string) {
	for i, clust := range gApplicationList.Items {
		if clust.Name == clustName {
			gApplicationList.Items[i].Annotations = make(map[string]string)
			gApplicationList.Items[i].Annotations[kkarlonapp.ProfilesAnnotationKey] = commaSeparatedProfiles
			return
		}
	}
	t.Errorf("failed to find kkarlon cluster with name %s", clustName)
}

func annotateArgoCluster(t *testing.T, clustName string, commaSeparatedProfiles string) {
	for i, clust := range gClusterList.Items {
		if clust.Name == clustName {
			gClusterList.Items[i].Annotations = make(map[string]string)
			gClusterList.Items[i].Annotations[kkarlonapp.ProfilesAnnotationKey] = commaSeparatedProfiles
			return
		}
	}
	t.Errorf("failed to find argo cluster with name %s", clustName)
}

func stringSetsEqual(s1 []string, s2 []string) bool {
	set1 := sets.NewSet[string](s1...)
	set2 := sets.NewSet[string](s2...)
	return set1.Equal(set2)
}

func kkarlonAppTargetsTheseClusters(t *testing.T, appName string, clustNames []string) bool {
	desiredClustNames := sets.NewSet[string](clustNames...)
	for _, app := range gApplicationSetList.Items {
		if app.Name == appName {
			actualClustNames := sets.NewSet[string]()
			for _, elem := range app.Spec.Generators[0].List.Elements {
				var element map[string]interface{}
				if err := json.Unmarshal(elem.Raw, &element); err != nil {
					t.Fatalf("failed to unmarshal json: %s", err)
				}
				val, ok := element["cluster_name"]
				if !ok {
					t.Fatalf("applicationset %s has an element with no cluster key", appName)
				}
				clustName := val.(string)
				actualClustNames.Add(clustName)
			}
			return desiredClustNames.Equal(actualClustNames)
		}
	}
	t.Fatalf("failed to find kkarlon app with name %s", appName)
	return false
}

func ensureProfileInvalidApps(t *testing.T, profName string, desiredInvApps []string) {
	desiredSet := sets.NewSet[string](desiredInvApps...)
	actualSet := sets.NewSet[string]()
	for _, prof := range gProfileList.Items {
		if prof.Name == profName {
			for _, invAppName := range prof.Status.InvalidAppNames {
				actualSet.Add(invAppName)
			}
			assert.True(t, desiredSet.Equal(actualSet))
			expectedHealth := "healthy"
			if len(prof.Status.InvalidAppNames) > 0 {
				expectedHealth = "degraded"
			}
			assert.Equal(t, prof.Status.Health, expectedHealth, "unexpected health status")
		}
	}
}
