# Arlon Design and Concepts

## Management cluster

This Kubernetes cluster hosts the following components:

- ArgoCD
- Arlon
- Cluster management stacks e.g. Cluster API and/or Crossplane

The Arlon state and controllers reside in the karlon namespace.

## Configuration bundle

A configuration bundle (or just "bundle") is grouping of data files that
produce a set of Kubernetes manifests via a *tool*. This closely follows ArgoCD's
definition of *tool types*. Consequently, the list of supported bundle
types mirrors ArgoCD's supported set of manifest-producing tools.
Each bundle is defined using a Kubernetes ConfigMap resource in the arlo namespace.
Additionally, a bundle can embed the data itself ("static bundle"), or contain a reference
to the data ("dynamic bundle"). A reference can be a URL, GitHub location, or Helm repo location.
The current list of supported bundle types is:

- manifest_inline: a single manifest yaml file embedded in the resource
- manifest_ref: a reference to a single manifest yaml file
- dir_inline: an embedded tarball that expands to a directory of YAML files
- helm_inline: an embedded Helm chart package
- helm_ref: an external reference to a Helm chart

### Bundle purpose

Bundles can specify an optional *purpose* to help classify and organize them.
In the future, Arlon may order bundle installation by purpose order (for e.g.
install bundles with purpose=*networking* before others) but that is not the
case today. The currently *suggested* purpose values are:

- networking
- add-on
- data-service
- application

## Profile

A profile expresses a desired configuration for a Kubernetes cluster.
It is composed of

- An optional clusterspec. If specified, it allows the profile
  to be used to create new clusters.
  If absent, the profile can only be applied to existing clusters.
- A list of bundles specifying the configuration to apply onto the cluster
  once it is operational
- An optional list of `values.yaml` settings for any Helm Chart type bundle
  in the bundle list

## Cluster

### Cluster Specification/ Metadata

A Cluster Specification contains desired settings when creating a new cluster. These settings are the values that define the shape and the configurations of the cluster.

There is a difference in the cluster specification for gen1 (deprecated) and gen2 clusters. The main difference in these cluster specifications is that gen2 Cluster Specification allow users to deploy arbitrarily complex clusters using the full Cluster API feature set. This is also closer to the gitops and declarative style of cluster creation and gives users more control over the cluster that they deploy.

The Cluster Specification of gen-2 clusrers is called the cluster template, which is described in detail [here](https://github.com/karlonproj/karlon/blob/main/docs/clustertemplate.md).

A cluster template manifest consists of:

- A predefined list of Cluster API objects: Cluster, Machines, Machine Deployments, etc. to be deployed in the current namespace
- The specific infrastructure provider to be used (e.g aws).ß
- Kubernetes version
- Cluster templates/ flavors that need to be used for creating the cluster manifest (e.g eks, eks-managedmachinepool)

### Cluster Preparation

Once these cluster specifications are created successfully, the next step is to prepare the cluster for deployment.

Once the cluster template manifest is created, the next step is to preare the workspace repository directory in which this cluster template manifest is present. This is explained in detail [here](https://github.com/karlonproj/karlon/blob/main/docs/clustertemplate.md#preparation)

### Cluster Creation

Now, all the prerequisites for creating a cluster are completed and the cluster can be created/deployed.

#### Cluster Chart

The cluster chart is a Helm chart that creates (and optionally applies) the manifests necessary to create a cluster and deploy desired configurations and applications to it as a part of cluster creation, the following resources are created: The profile's Cluster Specification, bundle list and other settings are used to generate values for the cluster chart, and the chart is deployed as a Helm release into the *karlon* namespace in the management cluster.

Here is a summary of the kinds of resources generated and deployed by the chart:

- A unique namespace with a name based on the cluster's name. All subsequent
  resources below are created inside that namespace.
- The stack-specific resources to create the cluster (for e.g. Cluster API resources)
- A ClusterRegistration to automatically register the cluster with ArgoCD
- A GitRepoDir to automatically create a git repo and/or directory to host a copy
  of the expanded bundles. Every bundle referenced by the profile is
  copied/unpacked into its own subdirectory.
- One ArgoCD Application resource for each bundle.

cluster template creation is explained [here](https://github.com/karlonproj/karlon/blob/main/docs/clustertemplate.md#creation)
