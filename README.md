# kubectl extension-versions

This is a kubectl plugin that lists you the installed versions of well-known
Kubernetes extensions/operators (and their subcomponents, if any) on your
cluster.

For example:

```sh
kubectl extension-versions
- istio:
  - pilot: docker.io/istio/pilot:1.0.2
  - sidecar-injector: docker.io/istio/sidecar_injector:1.0.2
  - policy: docker.io/istio/mixer:1.0.2
  - prometheus: (not found)
- knative:
  - build: gcr.io/knative-releases/github.com/knative/build/cmd/controller:v0.4.0
  - serving: gcr.io/knative-releases/github.com/knative/serving/cmd/controller:v0.4.0
  - eventing: gcr.io/knative-releases/github.com/knative/eventing/cmd/controller:v0.4.0
```

## Installation

> :warning::warning: These instructions don't work yet. Just `go build` this and
> place the binary to your `$PATH` as `kubectl-extension_versions` (mind the
> underscore) to get it to work.

1. Install [krew](https://github.com/GoogleContainerTools/krew) plugin manager
   for kubectl.

2. Install this plugin by running:

   kubectl krew install extension-versions

3. Run the plugin by calling it as:

   kubectl extension-versions

## Authors

- Ahmet Alp Balkan [(@ahmetb)](https://twitter.com/ahmetb)

---

This is not an official Google project.
