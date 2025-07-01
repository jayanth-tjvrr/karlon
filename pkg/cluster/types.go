package cluster

type Cluster struct {
	Name            string
	ClusterSpecName string           // empty for external clusters
	BaseCluster     *BaseClusterInfo // gen2 only
	ProfileName     string           // gen1 profile
	IsExternal      bool
	SecretName      string   // The corresponding argocd secret. Empty for non-external clusters.
	AppProfiles     []string // gen2 profiles
}

type BaseClusterInfo struct {
	Name         string
	RepoUrl      string
	RepoRevision string
	RepoPath     string
	Overridden   string
}

const clusterTypeLabelKey = "kkarlon.io/cluster-type"
const externalClusterTypeLabel = "kkarlon.io/cluster-type=external"
const argoClusterSecretTypeLabel = "argocd.argoproj.io/secret-type=cluster"

const baseClusterNameAnnotation = "kkarlon.io/basecluster-name"
const baseClusterRepoUrlAnnotation = "kkarlon.io/basecluster-repo-url"
const baseClusterRepoRevisionAnnotation = "kkarlon.io/basecluster-repo-revision"
const baseClusterRepoPathAnnotation = "kkarlon.io/basecluster-repo-path"
const baseClusterOverridden = "kkarlon.io/basecluster-overridden"

const ArlonGen1ClusterLabelQueryOnArgoApps = "managed-by=kkarlon,kkarlon-type=cluster"
const ArlonGen2ClusterLabelQueryOnArgoApps = "managed-by=kkarlon,kkarlon-type=cluster-app"
