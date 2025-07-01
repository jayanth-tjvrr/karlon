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

package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/jayanth-tjvrr/karlon/pkg/cmd/clustercontroller"
	"os"

	"go.uber.org/zap/zapcore"

	"github.com/jayanth-tjvrr/karlon/pkg/cmd/app"
	"github.com/jayanth-tjvrr/karlon/pkg/cmd/appprofile"
	"github.com/jayanth-tjvrr/karlon/pkg/cmd/appprofilecontroller"
	"github.com/jayanth-tjvrr/karlon/pkg/cmd/webhook"

	apppkg "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/argoproj/argo-cd/v2/util/io"
	"github.com/spf13/cobra"

	"github.com/jayanth-tjvrr/karlon/pkg/cmd/basecluster"
	"github.com/jayanth-tjvrr/karlon/pkg/cmd/bundle"
	"github.com/jayanth-tjvrr/karlon/pkg/cmd/callhomecontroller"
	"github.com/jayanth-tjvrr/karlon/pkg/cmd/cluster"
	"github.com/jayanth-tjvrr/karlon/pkg/cmd/controller"
	"github.com/jayanth-tjvrr/karlon/pkg/cmd/gitrepo"
	"github.com/jayanth-tjvrr/karlon/pkg/cmd/initialize"
	"github.com/jayanth-tjvrr/karlon/pkg/cmd/install"
	"github.com/jayanth-tjvrr/karlon/pkg/cmd/list_clusters"
	"github.com/jayanth-tjvrr/karlon/pkg/cmd/profile"
	"github.com/jayanth-tjvrr/karlon/pkg/cmd/verify"
	"github.com/jayanth-tjvrr/karlon/pkg/cmd/version"
	"github.com/jayanth-tjvrr/karlon/pkg/pkg/argocd"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
)

func main() {
	fmt.Fprintf(os.Stderr, "[DEBUG] main.go: program entry\n")
	command := &cobra.Command{
		Use:               "kkarlon",
		Short:             "Run the Arlon program",
		Long:              "Run the Arlon program",
		DisableAutoGenTag: true,
		PersistentPreRun:  checkForArgocd,
		Run: func(c *cobra.Command, args []string) {
			c.Println(c.UsageString())
		},
	}
	// don't display usage upon error
	command.SilenceUsage = true
	command.AddCommand(controller.NewCommand())
	command.AddCommand(callhomecontroller.NewCommand())
	command.AddCommand(appprofilecontroller.NewCommand())
	command.AddCommand(clustercontroller.NewCommand())
	command.AddCommand(list_clusters.NewCommand())
	command.AddCommand(bundle.NewCommand())
	command.AddCommand(profile.NewCommand())
	// `clusterspec` is gen-1 only
	//command.AddCommand(clusterspec.NewCommand())
	command.AddCommand(cluster.NewCommand())
	command.AddCommand(webhook.NewCommand())
	command.AddCommand(basecluster.NewCommand())
	command.AddCommand(gitrepo.NewCommand())
	command.AddCommand(verify.NewCommand())
	command.AddCommand(install.NewCommand())
	command.AddCommand(app.NewCommand())
	command.AddCommand(appprofile.NewCommand())
	command.AddCommand(initialize.NewCommand())
	command.AddCommand(version.NewCommand())

	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.RFC3339NanoTimeEncoder,
	}
	opts.BindFlags(flag.CommandLine)
	// override default log level, which is initially set to 'debug'
	_ = flag.Set("zap-log-level", "info")
	flag.Parse()
	logger := zap.New(zap.UseFlagOptions(&opts))
	ctrl.SetLogger(logger)
	args := flag.Args()
	command.SetArgs(args)
	fmt.Fprintf(os.Stderr, "[DEBUG] main.go: before command.Execute\n")
	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "[DEBUG] main.go: command.Execute returned error: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "[DEBUG] main.go: after command.Execute\n")
}

func checkForArgocd(cmd *cobra.Command, args []string) {
	fmt.Fprintf(os.Stderr, "[DEBUG] checkForArgocd: called for command %s\n", cmd.Name())
	if cmd.Name() == "cluster" || cmd.Name() == "listclusters" {
		conn, appIf := argocd.NewArgocdClientOrDie("").NewApplicationClientOrDie()
		defer io.Close(conn)
		query := "managed-by=kkarlon,kkarlon-type=cluster"
		_, err := appIf.List(context.Background(), &apppkg.ApplicationQuery{Selector: &query})
		if err != nil {
			fmt.Println("ArgoCD auth token has expired....Login to ArgoCD again")
			os.Exit(1)
		}
	}
}
