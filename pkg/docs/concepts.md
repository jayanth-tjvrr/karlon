# Concepts

## Management cluster

This Kubernetes cluster hosts the following components:

- ArgoCD
- Arlon
- Cluster management stacks e.g. Cluster API and/or Crossplane

The Arlon state and controllers reside in the karlon namespace.

## Workload Cluster

An Arlon workload cluster is a Kubernetes cluster
that Arlon creates and manages via a git directory structure stored in
the workspace repository.

The new way of provisioning workload clusters in Arlon since v0.9 is gen2 clusters using *cluster template* that replace gen1 clusters using *cluster spec*
The most significant change in gen2 clusters is the *Cluster Template* construct, which replaces the older cluster spec from gen1 clusters.
To distinguish them from the older gen1 clusters, the ones deployed from a cluster template are called next-gen clusters or gen2 clusters.

## Cluster Template 

A cluster template is a base cluster manifest that can be "cloned" to produce
one or more identical or similar workload clusters.
To create a cluster template, you first compose a manifest containing one or more
declarative resources that define the kind and shape of cluster that you desire.
You then store the manifest in its own directoy somewhere in git.
You then instruct Arlon to "prep" the directory to promote it to cluster template.
To know more about cluster template for Arlon gen2 clusters including the difference
with cluster spec and the process to create gen2 clusters; read the document [cluster template](./clustertemplate.md)


## Application (App)

An Arlon Application (or "App" for short) defines a source of Kubernetes
manifests that can be applied/deployed to a workload cluster. It can
take the form of raw YAML files, a Helm chart, or a Kustomize directory.
The source resides in a git repository.
Arlon represents an App as a specialized ArgoCD ApplicationSet resource.
An App is not limited traditional "applications" and can refer to any set of
deployable resources, for e.g. Kubernetes RBAC rules or other types
of configurations.
For more details about Apps, refer to [AppProfiles article](./appprofiles.md)

## Application Profile (AppProfile)

An AppProfile is simply a set of App names referring to Arlon Apps.
You use AppProfile resources to define common groupings of apps, for example
"monitoring-stack", or "security-policies-1".
It is perfectly legal for multiple AppProfiles to refer to some common App names,
meaning they can overlap.
For more details about AppProfiles, refer to [AppProfiles article](./appprofiles.md)

## Deploying apps to workload clusters

You deploy apps to workload clusters by annotating a workload cluster
with the desired AppProfile(s). The union of all Apps referenced by those
AppProfiles is deployed to the cluster.
For more details about annotating/targeting clusters,
refer to [AppProfiles article](./appprofiles.md)

