package basecluster

import (
	"fmt"
	"github.com/argoproj/argo-cd/v2/util/cli"
	"github.com/jayanth-tjvrr/karlon/pkg/argocd"
	bcl "github.com/jayanth-tjvrr/karlon/pkg/basecluster"
	"github.com/jayanth-tjvrr/karlon/pkg/gitrepo"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

func validateGitBaseClusterCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var argocdNs string
	var repoUrl string
	var repoAlias string
	var repoPath string
	var repoRevision string
	command := &cobra.Command{
		Use:   "validategit --repo-url repoUrl [--repo-revision revision] [--repo-path path]",
		Short: "validate cluster template directory in git",
		Long:  "validate cluster template directory in git",
		RunE: func(c *cobra.Command, args []string) error {
			if repoUrl == "" {
				var err error
				repoUrl, err = gitrepo.GetRepoUrl(repoAlias)
				if err != nil {
					return err
				}
			}
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			_, creds, err := argocd.GetKubeclientAndRepoCreds(config, argocdNs, repoUrl)
			if err != nil {
				return fmt.Errorf("failed to get repository credentials: %s", err)
			}
			clusterName, err := bcl.ValidateGitDir(creds, repoUrl, repoRevision, repoPath)
			if err != nil {
				return err
			}
			fmt.Println("validation successful, cluster name:", clusterName)
			return nil
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&argocdNs, "argocd-ns", "argocd", "the argocd namespace")
	command.Flags().StringVar(&repoUrl, "repo-url", "", "the git repository url for cluster template directory")
	command.Flags().StringVar(&repoAlias, "repo-alias", gitrepo.RepoDefaultCtx, "git repository alias to use")
	command.Flags().StringVar(&repoRevision, "repo-revision", "main", "the git revision for cluster template directory")
	command.Flags().StringVar(&repoPath, "repo-path", "", "the git repository path for cluster template directory")
	command.MarkFlagsMutuallyExclusive("repo-url", "repo-alias")
	return command
}
