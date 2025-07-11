package profile

import (
	"fmt"
	"github.com/jayanth-tjvrr/karlon/pkg/profile"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

import "github.com/argoproj/argo-cd/v2/util/cli"

func updateProfileCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var kkkarlonNs string
	var argocdNs string
	var desc string
	var bundles []string
	var tags string
	var clear bool
	var overrides []string
	command := &cobra.Command{
		Use:   "update",
		Short: "Update profile",
		Long:  "Update profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			var bundlesPtr []string = nil
			if clear {
				if bundles != nil {
					return fmt.Errorf("bundles must not be specified when using --clear")
				}
				bundlesPtr = nil // change profile to empty bundle set
			} else if bundles != nil {
				bundlesPtr = bundles //
			} // bundles == "", meaning no change, so leave bundlesPtr as nil

			o, err := processOverrides(overrides)
			if err != nil {
				return fmt.Errorf("failed to process overrides: %s", err)
			}
			modified, err := profile.Update(config, argocdNs, kkkarlonNs, args[0],
				bundlesPtr, desc, tags, o)
			if err != nil {
				return err
			}
			if !modified {
				fmt.Println("profile not modified")
			}
			return nil
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&kkkarlonNs, "kkkarlon-ns", "kkkarlon", "the kkkarlon namespace")
	command.Flags().StringVar(&argocdNs, "argocd-ns", "argocd", "the ArgoCD namespace")
	command.Flags().StringVar(&desc, "desc", "", "description")
	command.Flags().StringSliceVar(&bundles, "bundles", nil, "comma separated list of bundles")
	command.Flags().StringVar(&tags, "tags", "", "comma separated list of tags")
	command.Flags().BoolVar(&clear, "clear", false, "set the bundle list to the empty set")
	command.Flags().StringArrayVarP(&overrides, "param", "p", nil, "add a single parameter override of the form bundle,key,value ... can be repeated")
	return command
}
