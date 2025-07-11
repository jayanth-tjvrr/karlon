# Application Profiles (new in v0.10.0)

The Application Profiles feature, also known as Gen2 Profiles, is an
addition to Arlon v0.10.0 that provides a new way to describe, group,
and deploy manifests to workload clusters. The new feature introduces
these concepts:

* Arlon Application (or just "App")
* Application Profile (a.k.a. AppProfile)
* Targeting application profiles to workload clusters via annotation

The feature provides an alternative to the Bundle and Profile concepts
("the old way") of earlier versions of Arlon. Specifically,
the Arlon Application can be viewed as a replacement for Bundles,
and AppProfile is a substitute for Profile.
In release v0.10.0, the "old way" continues to be supported, but is deprecated,
meaning it will likely be retired in an upcoming release.

## Arlon Application (a.k.a. "Arlon App")

An Arlon Application is similar to a Dynamic Bundle from earlier releases.
It specifies a source of one or more manifests stored in git in any
["tool" format supported by ArgoCD](https://argo-cd.readthedocs.io/en/stable/user-guide/application_sources/)

Internally, Arlon represents an App as a specialized [ArgoCD ApplicationSet](https://argo-cd.readthedocs.io/en/stable/operator-manual/applicationset/) resource.
This allows you to specify the manifest source in the `spec.template.spec` section,
while Arlon takes care of targeting the deployment to the correct workload cluster(s)
by automatically manipulating the `spec.generators` section.
ApplicationSets managed by Arlon to represent apps are distinguished from
other ApplicationSets via the `kkarlon-type=application` label.
The ApplicationSet's Generators list must contain a single generator of [List](https://argo-cd.readthedocs.io/en/stable/operator-manual/applicationset/Generators-List/)
type. The Arlon AppProfile controller will modify this list in real-time
to deploy the application to the right workload clusters (or no cluster at all).

While it is possible for you create and edit an ApplicationSet resource manifest
satisfying the requirements to be an Arlon App from scratch, Arlon makes
this easier with the `kkarlon app create --output-yaml` command, which outputs an
initial compliant manifest that you can
save to a file and edit to your liking before applying to
the management cluster to actually create the app.
(Without the `--output-yaml` flag, the command will apply the resource for you).

Here's an example of an initial Arlon Application manifest:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  labels:
    kkarlon-type: application
    managed-by: kkarlon
  name: myconfigmap
  namespace: argocd
spec:
  generators:
  - list: {}
  template:
    metadata:
      name: '{{cluster_name}}-app-myconfigmap'
    spec:
      destination:
        namespace: default
        server: '{{cluster_server}}'
      project: default
      source:
        path: apps/my-cfgmap-1
        repoURL: https://github.com/bcle/fleet-infra.git
        targetRevision: HEAD
      syncPolicy:
        automated:
          prune: true
```

The List generator that the AppProfile controller maintains supplies two variables for template substitution:

* `cluster_name`: the name of the target workload cluster
* `cluster_server`: the URL+FQDN of the workload cluster's Kubernetes API endpoint

Notice how the initial manifest takes advantage of those variables to set

* `spec.template.metadata.name` to `{{cluster-name}}-app-myconfigmap`
to ensure that any actual ArgoCD Application resources deployed from the ApplicationSet are uniquely named, by prefixing the cluster name.
* `spec.template.spec.destination.server` to `{{cluster_server}}` to target the correct workload cluster

Arlon apps can be listed in two ways. The first is to use the `kkarlon app list` command. One advantage
is that it's simple to use and also displays additional information about the app, such as which AppProfiles
are currently associated with the app.

Example:

```shell
$ kkarlon app list
NAME         REPO                                     PATH              REVISION  APP_PROFILES
myconfigmap  https://github.com/bcle/fleet-infra.git  apps/my-cfgmap-1  HEAD      [marketing]
```

The second way is to use pure kubectl to list ApplicationSets with the Arlon specific label:

```shell
$ kubectl -n argocd get applicationset -l kkarlon-type=application
NAME          AGE
myconfigmap   21d
```

Similarly, an Arlon app can be deleted in two ways:

- `kkarlon app delete <appName>`
- `kubectl -n argocd delete applicationset <appName>`

## AppProfile

An AppProfile is simply a grouping (or set) of Arlon Apps.
Unlike an Arlon Application (which is represented by an ApplicationSet resource),
an AppProfile is represented by an Arlon-native custom resource.

An AppProfile specifies the apps it is associated with via the `appNames` list.
It is legal for `appNames` to contain names of Arlon Apps that don't exist yet.
To indicate whether some app names are currently invalid, the AppProfile controller
will update the resource's `status` section as follows:

- If all specified app names refer to valid Arlon apps, `status.health`
  is set to `healthy`.
- If one or more specified app names refer to non-existent Arlon apps,
  then `status.health` is set to `degraded`, and `status.invalidAppNames`
  lists the invalid names.

Here is an example of an AppProfile manifest that includes 3 apps, one of which does not exist:

```yaml
apiVersion: core.kkarlon.io/v1
kind: AppProfile
metadata:
  name: marketing
  namespace: kkarlon
spec:
  appNames:
  - myconfigmap
  - wordpress
  - nonexistent-app
status:
  health: degraded
  invalidAppNames:
  - nonexistent-app
```

> Note: AppProfile resources reside in the `kkarlon` namespace.
> This is in contrast to Arlon App resources, which reside in the `argocd` namespace since they are actually ArgoCD ApplicationSet resources.

Since AppProfiles are defined by their own custom resource and are fairly straightforward, their lifecycle
can be managed entirely using `kubectl`. That said, Arlon provides the `kkarlon appprofile list` to display
useful information about current AppProfiles. Example:

```shell
$ kkarlon appprofile list
NAME         APPS                     HEALTH   INVALID_APPS
engineering  [wordpress]              healthy  []
marketing    [myconfigmap wordpress]  degraded [nonexistent-app]
```

As the example illustrates, it is totally legal for two or more AppProfiles to include the same app(s).
When those AppProfiles are targeted to a workload cluster, the cluster receives the union of these app sets.

## Targeting AppProfiles to Workload Clusters

Apps are actually deployed to workload clusters by annotating the desired cluster(s) with `kkarlon.io/profiles=<comma separated list of AppProfiles>`. This is supported only on Arlon Gen2 clusters, which themselves are represented as ArgoCD Application resources.

Example: suppose we have a Gen2 workload cluster named **clust1**.  Since the cluster is represented as an ArgoCD Application resource, targeting the `engineering` and `marketing` app profiles to it is achieved with:

```shell
kubectl -n argocd annotate --overwrite application clust1 kkarlon.io/profiles=engineering,marketing
```

The set of app profiles for a cluster can always be updated that way, in real time. Behind the scenes, Arlon's AppProfile controller
updates the `spec.generators` section of each affected app to target the cluster. The set of apps targeting a cluster is the union
of all the valid appNames from every app profile listed in the annotation.

By default, every ArgoCD Application Resource generated by the app's ApplicationSet is named with the pattern `<clustername>-app-<appname>`.
In the running example, these two ArgoCD application resources will be generated:

* clust1-app-wordpress
* clust1-app-myconfigmap

To detach all profiles from a cluster, simply remove the annotation:

```shell
kubectl -n argocd annotate application clust1 kkarlon.io/profiles-
```

The last command will automatically destroy any ArgoCD application resources generated for that cluster (assuming they originate from Arlon apps).

> Note: users are free to create and maintain their own ApplicationSets not managed by Arlon. Those will work side-by-side with the ones
> managed by Arlon, which is unaware of those other ApplicationSets. This allows the user to take advantage of other
> types of [ApplicationSet generators](https://argo-cd.readthedocs.io/en/stable/operator-manual/applicationset/Generators/).

