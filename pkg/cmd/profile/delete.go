package profile

import (
	"fmt"
	"github.com/jayanth-tjvrr/karlon/pkg/controller"
	"github.com/jayanth-tjvrr/karlon/pkg/profile"
	"github.com/spf13/cobra"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

import "github.com/argoproj/argo-cd/v2/util/cli"

func deleteProfileCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var ns string
	command := &cobra.Command{
		Use:   "delete",
		Short: "Delete profile",
		Long:  "Delete profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			profileName := args[0]
			err = deleteProfile(config, ns, profileName)
			if err != nil {
				if !apierr.IsNotFound(err) {
					return err
				}
				fmt.Printf("%s not found, assuming legacy profile\n", profileName)
				// try deleting the legacy profile
				errLegacy := deleteProfileLegacy(config, ns, profileName)
				if errLegacy != nil {
					return errLegacy
				}
			}
			return nil
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&ns, "ns", "kkkarlon", "the kkkarlon namespace")
	return command
}

func deleteProfile(config *restclient.Config, ns string, profileName string) error {
	ctrl, err := controller.NewClient(config)
	if err != nil {
		return fmt.Errorf("failed to get controller client: %w", err)
	}
	return profile.Delete(ctrl, ns, profileName)

}

func deleteProfileLegacy(config *restclient.Config, ns string, profileName string) error {
	kubeClient := kubernetes.NewForConfigOrDie(config)
	return profile.DeleteLegacy(kubeClient, ns, profileName)
}
