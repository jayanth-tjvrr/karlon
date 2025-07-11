package cluster

import (
	"context"
	"fmt"
	"os"

	apppkg "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/argoproj/argo-cd/v2/util/io"
	"github.com/jayanth-tjvrr/karlon/pkg/argocd"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	command := &cobra.Command{
		Use:               "cluster",
		Short:             "Manage clusters",
		Long:              "Manage clusters",
		DisableAutoGenTag: true,
		Aliases:           []string{"clusters"},
		PersistentPreRun:  checkForArgocd,
		Run: func(c *cobra.Command, args []string) {
			_ = c.Usage()
		},
	}
	// `cluster deploy` is only for gen1 clusters
	//command.AddCommand(deployClusterCommand())
	command.AddCommand(listClustersCommand())
	// `cluster update` is only for gen1 clusters
	//command.AddCommand(updateClusterCommand())
	command.AddCommand(manageClusterCommand())
	command.AddCommand(unmanageClusterCommand())
	command.AddCommand(createClusterCommand())
	command.AddCommand(getClusterCommand())
	command.AddCommand(deleteClusterCommand())
	command.AddCommand(ngupdateClusterCommand())
	command.AddCommand(setAppProfilesCommand())
	return command
}

func checkForArgocd(c *cobra.Command, args []string) {
	conn, appIf := argocd.NewArgocdClientOrDie("").NewApplicationClientOrDie()
	defer io.Close(conn)
	query := "managed-by=kkkarlon,kkkarlon-type=cluster"
	_, err := appIf.List(context.Background(), &apppkg.ApplicationQuery{Selector: &query})
	if err != nil {
		fmt.Println("ArgoCD auth token has expired....Login to ArgoCD again")
		fmt.Println(err)
		os.Exit(1)
	}
}
