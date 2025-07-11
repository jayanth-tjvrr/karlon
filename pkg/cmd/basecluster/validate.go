package basecluster

import (
	"fmt"
	bcl "github.com/jayanth-tjvrr/karlon/pkg/basecluster"
	"github.com/spf13/cobra"
	// "k8s.io/client-go/tools/clientcmd"
)

func validateBaseClusterCommand() *cobra.Command {
	/*
		var clientConfig clientcmd.ClientConfig
		var argocdNs string
		var kkkarlonNs string
	*/
	command := &cobra.Command{
		Use:   "validate <filename> [flags]",
		Short: "validate cluster template files",
		Long:  "validate cluster template files",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			/*
				argoIf := argocd.NewArgocdClientOrDie("")
				config, err := clientConfig.ClientConfig()
				if err != nil {
					return fmt.Errorf("failed to get k8s client config: %s", err)
				}
			*/
			fileName := args[0]
			clusterName, err := bcl.Validate(fileName)
			if err != nil {
				return err
			}
			fmt.Println("validation successful, cluster name:", clusterName)
			return nil
		},
	}
	return command
}
