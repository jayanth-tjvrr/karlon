package cluster

import (
	_ "embed"
	"fmt"
	"github.com/argoproj/argo-cd/v2/util/cli"
	"github.com/jayanth-tjvrr/karlon/pkg/pkg/argocd"
	"github.com/jayanth-tjvrr/karlon/pkg/pkg/cluster"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

func deleteClusterCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var argocdNs string
	var kkkarlonNs string
	command := &cobra.Command{
		Use:   "delete <clustername> [flags]",
		Short: "delete existing cluster and all related resources",
		Long:  "delete existing cluster and all related resources",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			argoIf := argocd.NewArgocdClientOrDie("")
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to delete k8s client config: %s", err)
			}
			clusterName := args[0]
			err = cluster.Delete(argoIf, config, argocdNs, clusterName)
			if err != nil {
				return fmt.Errorf("failed to delete cluster: %s", err)
			}
			return nil
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&argocdNs, "argocd-ns", "argocd", "the argocd namespace")
	command.Flags().StringVar(&kkkarlonNs, "kkkarlon-ns", "kkkarlon", "the kkkarlon namespace")
	return command
}
