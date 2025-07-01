package cluster

import (
	"fmt"
	"os"

	"github.com/jayanth-tjvrr/karlon/pkg/pkg/cluster"
	"github.com/jayanth-tjvrr/karlon/pkg/pkg/gitrepo"
	"github.com/spf13/cobra"
)

func createClusterCommand() *cobra.Command {
	var clusterRepoUrl string
	var repoAlias string
	var clusterRepoRevision string
	var clusterRepoPath string
	var clusterName string
	var namespace string
	var k8sVersion string
	var replicas int
	var outputYaml bool
	command := &cobra.Command{
		Use:   "create",
		Short: "create new workload cluster from a cluster template",
		Long:  "create new workload cluster from a cluster template",
		RunE: func(c *cobra.Command, args []string) error {
			fmt.Fprintf(os.Stderr, "[DEBUG] Entered cluster create RunE\n")
			// Only require clusterName, namespace, replicas, k8sVersion
			if clusterName == "" || namespace == "" || k8sVersion == "" {
				return fmt.Errorf("cluster-name, namespace, and k8s-version are required")
			}
			// Derive all other fields using conventions
			hcpName := clusterName + "cp"
			controlPlaneEndpointHost := fmt.Sprintf("%scp.default-service.k8s.byo-upg-3.app.dev-pcd.platform9.com", clusterName)
			oidcConfigMap := fmt.Sprintf("oidc-auth-%s", clusterName)
			oidcConfigMapCP := fmt.Sprintf("oidc-auth-%scp", clusterName)
			// Call existing manifest generation logic (refactor if needed)
			fmt.Fprintf(os.Stderr, "[DEBUG] Generating manifest...\n")
			manifest, err := cluster.GenerateClusterManifest(
				clusterName,
				namespace,
				hcpName,
				controlPlaneEndpointHost,
				oidcConfigMap,
				oidcConfigMapCP,
				replicas,
				k8sVersion,
			)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[DEBUG] Manifest generation failed: %v\n", err)
				return err
			}
			fmt.Fprintf(os.Stderr, "[DEBUG] Manifest generated.\n")
			if outputYaml {
				fmt.Fprintf(os.Stderr, "[DEBUG] Outputting YAML to stdout (no push to git).\n")
				fmt.Println(manifest)
				return nil
			}
			fmt.Fprintf(os.Stderr, "[DEBUG] Preparing to push YAML to git...\n")
			// Push YAML to GitHub repo
			repoUrl := clusterRepoUrl
			if repoUrl == "" {
				var err error
				repoUrl, err = gitrepo.GetRepoUrl(repoAlias)
				if err != nil {
					return fmt.Errorf("failed to resolve repo url: %w", err)
				}
			}
			branch := clusterRepoRevision
			if branch == "" {
				branch = "main"
			}
			path := clusterRepoPath
			if path == "" {
				path = "clusters" // default path in repo
			}
			filePath := fmt.Sprintf("%s/%s-cluster.yaml", path, clusterName)
			err = gitrepo.CommitAndPushFile(repoUrl, branch, filePath, []byte(manifest), fmt.Sprintf("Add cluster manifest for %s", clusterName))
			if err != nil {
				return fmt.Errorf("failed to push manifest to git: %w", err)
			}
			fmt.Printf("Cluster manifest pushed to %s on branch %s at path %s\n", repoUrl, branch, filePath)
			return nil
		},
	}
	command.Flags().StringVar(&clusterRepoUrl, "repo-url", "", "the git repository url for cluster template")
	command.Flags().StringVar(&repoAlias, "repo-alias", gitrepo.RepoDefaultCtx, "git repository alias to use")
	command.Flags().StringVar(&clusterRepoRevision, "repo-revision", "main", "the git revision for cluster template")
	command.Flags().StringVar(&clusterRepoPath, "repo-path", "", "the git repository path for cluster template")
	command.Flags().StringVar(&clusterName, "cluster-name", "", "Cluster name (required)")
	command.Flags().StringVar(&namespace, "namespace", "", "Namespace (required)")
	command.Flags().IntVar(&replicas, "replicas", 1, "Replica count (required)")
	command.Flags().StringVar(&k8sVersion, "k8s-version", "v1.31.2", "Kubernetes version (required)")
	command.Flags().BoolVar(&outputYaml, "output-yaml", false, "Output manifest YAML to stdout instead of pushing to git")
	_ = command.MarkFlagRequired("cluster-name")
	_ = command.MarkFlagRequired("namespace")
	_ = command.MarkFlagRequired("k8s-version")
	command.MarkFlagsMutuallyExclusive("repo-url", "repo-alias")
	return command
}
