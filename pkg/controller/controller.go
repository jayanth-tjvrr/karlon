package controller

import (
	argoapp "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	//appset "github.com/argoproj/argo-cd/v2/pkg/apis/applicationset/v1alpha1"
	"os"

	kkarlonv1 "github.com/jayanth-tjvrr/karlon/pkg/api/v1"
	"github.com/jayanth-tjvrr/karlon/pkg/controllers"
	"github.com/jayanth-tjvrr/karlon/pkg/argocd"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(kkarlonv1.AddToScheme(scheme))
	utilruntime.Must(argoapp.AddToScheme(scheme))
	utilruntime.Must(argoapp.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func NewClient(config *rest.Config) (client.Client, error) {
	return client.New(config, client.Options{
		Scheme: scheme,
	})
}

func StartController(argocdConfigPath string, metricsAddr string, probeAddr string, enableLeaderElection bool) {
	argocdClient := argocd.NewArgocdClientOrDie(argocdConfigPath)
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "d4242dee.kkarlon.io",
		// Disable caching for secret objects, because the controller reads them
		// in a particular namespace. Caching requires RBAC to be setup for
		// cluster-wide List access, as opposed to the more secure
		// namespace-scoped access.
		// Update: need to do the same for ServiceAccount
		ClientDisableCacheFor: []client.Object{
			&corev1.Secret{},
			&corev1.ServiceAccount{},
		},
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.ClusterRegistrationReconciler{
		Client:       mgr.GetClient(),
		Scheme:       mgr.GetScheme(),
		ArgocdClient: argocdClient,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ClusterRegistration")
		os.Exit(1)
	}
	if err = (&controllers.CallHomeConfigReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CallHomeConfig")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func StartCallHomeController(metricsAddr string, probeAddr string, enableLeaderElection bool) {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "d5252dee.kkarlon.io",
		// Disable caching for secret objects, because the controller reads them
		// in a particular namespace. Caching requires RBAC to be setup for
		// cluster-wide List access, as opposed to the more secure
		// namespace-scoped access.
		ClientDisableCacheFor: []client.Object{
			&corev1.Secret{},
			&corev1.ServiceAccount{},
		},
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.CallHomeConfigReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ClusterRegistration")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func StartAppProfileController(argocdConfigPath string, metricsAddr string, probeAddr string, enableLeaderElection bool) {
	argocdClient := argocd.NewArgocdClientOrDie(argocdConfigPath)
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "d5252dee.kkarlon.io",
		// Disable caching for secret objects, because the controller reads them
		// in a particular namespace. Caching requires RBAC to be setup for
		// cluster-wide List access, as opposed to the more secure
		// namespace-scoped access.
		// Update: need to do the same for ServiceAccount
		/*
			ClientDisableCacheFor: []client.Object{
				&corev1.Secret{},
				&corev1.ServiceAccount{},
			},
		*/
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.AppProfileReconciler{
		Client:       mgr.GetClient(),
		Scheme:       mgr.GetScheme(),
		ArgocdClient: argocdClient,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to set up controller", "controller", "AppProfile")
		os.Exit(1)
	}
	if err = (&controllers.ApplicationSetReconciler{
		Client:       mgr.GetClient(),
		Scheme:       mgr.GetScheme(),
		ArgocdClient: argocdClient,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to set up controller", "controller", "ApplicationSet")
		os.Exit(1)
	}
	if err = (&controllers.ApplicationReconciler{
		Client:       mgr.GetClient(),
		Scheme:       mgr.GetScheme(),
		ArgocdClient: argocdClient,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to set up controller", "controller", "Application")
		os.Exit(1)
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func StartClusterController(
	config *rest.Config,
	argocdConfigPath string,
	metricsAddr string,
	probeAddr string,
	enableLeaderElection bool,
) {
	argocdClient := argocd.NewArgocdClientOrDie(argocdConfigPath)
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "d5252dee.kkarlon.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.ClusterReconciler{
		Client:       mgr.GetClient(),
		Scheme:       mgr.GetScheme(),
		ArgocdClient: argocdClient,
		Config:       config,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to set up controller", "controller", "AppProfile")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
