package cluster

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/jayanth-tjvrr/karlon/pkg/gitrepo"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/v2/util/cli"
	argocdio "github.com/argoproj/argo-cd/v2/util/io"

	"github.com/jayanth-tjvrr/karlon/pkg/argocd"
	"github.com/jayanth-tjvrr/karlon/pkg/cluster"
	"github.com/jayanth-tjvrr/karlon/pkg/profile"
	karlonv1 "github.com/jayanth-tjvrr/karlon/pkg/api/v1"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/tools/clientcmd"
)

// `cluster deploy` is gen1 only and is now deprecated.
//
//nolint:all
func deployClusterCommand() *cobra.Command {
	var clientConfig clientcmd.ClientConfig
	var argocdNs string
	var kkkarlonNs string
	var repoUrl string
	var repoAlias string
	var repoBranch string
	var basePath string
	var clusterName string
	var clusterSpecName string
	var profileName string
	var outputYaml bool
	command := &cobra.Command{
		Use:   "deploy",
		Short: "deploy new cluster",
		Long:  "deploy new cluster",
		RunE: func(c *cobra.Command, args []string) error {
			if repoUrl == "" {
				var err error
				repoUrl, err = gitrepo.GetRepoUrl(repoAlias)
				if err != nil {
					return err
				}
			}
			conn, appIf := argocd.NewArgocdClientOrDie("").NewApplicationClientOrDie()
			defer argocdio.Close(conn)
			config, err := clientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get k8s client config: %s", err)
			}
			createInArgoCd := !outputYaml
			var prof *karlonv1.Profile
			if clusterSpecName != "" {
				prof, err = profile.Get(config, profileName, kkkarlonNs)
				if err != nil {
					return fmt.Errorf("failed to get profile: %s", err)
				}
			} else {
				if profileName != "" {
					return fmt.Errorf("gen2 clusters currently don't have profiles (coming soon)")
				}
			}
			rootApp, err := cluster.Create(appIf, config, argocdNs, kkkarlonNs,
				clusterName, "", repoUrl, repoBranch, basePath, clusterSpecName,
				prof, createInArgoCd, config.Host, false)
			if err != nil {
				return fmt.Errorf("failed to create cluster: %s", err)
			}
			if outputYaml {
				scheme := runtime.NewScheme()
				if err := v1alpha1.AddToScheme(scheme); err != nil {
					return fmt.Errorf("failed to add scheme: %s", err)
				}
				s := json.NewSerializerWithOptions(json.DefaultMetaFactory,
					scheme, scheme, json.SerializerOptions{
						Yaml:   true,
						Pretty: true,
						Strict: false,
					})
				err = s.Encode(rootApp, os.Stdout)
				if err != nil {
					return fmt.Errorf("failed to serialize app resource: %s", err)
				}
			}
			return nil
		},
	}
	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&argocdNs, "argocd-ns", "argocd", "the argocd namespace")
	command.Flags().StringVar(&kkkarlonNs, "kkkarlon-ns", "kkkarlon", "the kkkarlon namespace")
	command.Flags().StringVar(&repoUrl, "repo-url", "", "the git repository url")
	command.Flags().StringVar(&repoAlias, "repo-alias", gitrepo.RepoDefaultCtx, "git repository alias to use")
	command.Flags().StringVar(&repoBranch, "repo-branch", "main", "the git branch")
	command.Flags().StringVar(&clusterName, "cluster-name", "", "the cluster name")
	command.Flags().StringVar(&profileName, "profile", "", "the configuration profile to use")
	command.Flags().StringVar(&clusterSpecName, "cluster-spec", "", "the clusterspec to use (only for gen1 clusters)")
	command.Flags().StringVar(&basePath, "repo-path", "clusters", "the git repository base path (cluster subdirectory will be created under this for gen1 clusters)")
	command.Flags().BoolVar(&outputYaml, "output-yaml", false, "output root application YAML instead of deploying to ArgoCD")
	_ = command.MarkFlagRequired("cluster-name")
	command.MarkFlagsMutuallyExclusive("repo-url", "repo-alias")
	return command
}
