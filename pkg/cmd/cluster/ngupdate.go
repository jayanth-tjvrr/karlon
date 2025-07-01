package cluster

import (
	"context"
	"fmt"

	argoapp "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/argoproj/argo-cd/v2/util/cli"
	"github.com/argoproj/argo-cd/v2/util/io"
	"github.com/jayanth-tjvrr/karlon/pkg/argocd"
	"github.com/jayanth-tjvrr/karlon/pkg/cluster"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

func ngupdateClusterCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var argocdNs string
	var kkkarlonNs string
	var profileName string
	var deleteProfileName bool
	command := &cobra.Command{
		Use:   "ngupdate <clustername> [flags]",
		Short: "update existing next-gen cluster",
		Long:  "update existing next-gen cluster",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			name := args[0]
			conn, appIf := argocd.NewArgocdClientOrDie("").NewApplicationClientOrDie()
			defer io.Close(conn)
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			selector := "kkkarlon-cluster=" + name + ",kkkarlon-type=cluster-app"
			apps, err := appIf.List(context.Background(),
				&argoapp.ApplicationQuery{Selector: &selector})
			if err != nil {
				return fmt.Errorf("failed to list apps related to cluster: %s", err)
			}
			if len(apps.Items) == 0 {
				return fmt.Errorf("failed to get the given cluster")
			}
			if !deleteProfileName {
				_, err = cluster.NgUpdate(appIf, config, argocdNs, kkkarlonNs, name, profileName, true)
				if err != nil {

					return fmt.Errorf("error: %s", err)
				}
			} else {
				err = cluster.DestroyProfileApps(appIf, name)
				if err != nil {
					return fmt.Errorf("failed to delete the profile app")
				}
			}
			return nil
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&argocdNs, "argocd-ns", "argocd", "the argocd namespace")
	command.Flags().StringVar(&kkkarlonNs, "kkkarlon-ns", "kkkarlon", "the kkkarlon namespace")
	command.Flags().StringVar(&profileName, "profile", "", "the configuration profile to use")
	command.Flags().BoolVar(&deleteProfileName, "delete-profile", false, "delete the existing profile app from the cluster")
	return command
}
