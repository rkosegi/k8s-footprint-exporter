# Kubernetes footprint exporter

<img src="icon.png" style="width: 64px" alt="icon">

This is Prometheus exporter for K8s API resources footprint.
It provides metrics about estimated size of API resources stored in etcd.
Size is estimated based off size of object serialized into JSON byte slice.
This might not be 100% accurate, but it's good enough for purpose of tracking size over time.

## Installation

### Using helm chart

- prepare `my-values.yaml` with metrics definition, see examples bellow.
- run `helm upgrade --install my-release ./chart --values my-values.yaml`
- forward service port `kubectl port-forward service/k8sfootprint-exporter 8889:80`
- check output `curl http://localhost:8889/metrics`

## Building

### Using docker

- run `make build-docker` (requires docker and GNU Make)

### Directly

- run `make build-local` (requires Go SDK)

## Example metrics config

```yaml
cms_and_secrets:
  apiVersion: v1
  kinds:
    configmaps:
      nameLabel: true
      includeOnly: cm.*
      size: true
      count: true
    secret:
      size: false
      count: true
```
Config above will produce following metrics (assuming there are 2 configmaps cm1 and cm2 and one secret sec1 in namespace):

```prometheus_metrics
k8sfootprint_resources_size{resource_set="cms_and_secrets",resource_name="cm1",api_version="v1",kind="configmap"} 237
k8sfootprint_resources_size{resource_set="cms_and_secrets",resource_name="cm2",apiVersion="v1",kind="configmap"} 319
k8sfootprint_resources_count{resource_set="cms_and_secrets",resource_name="*",apiVersion="v1",kind="configmap"} 2
k8sfootprint_resources_size{resource_set="cms_and_secrets",resource_name="*",apiVersion="v1",kind="secret"} 341
k8sfootprint_resources_count{resource_set="cms_and_secrets",resource_name="*",apiVersion="v1",kind="secret"} 1
```

In order for this exporter to have access to those resource in cluster, following RBAC rules must be part of role associated with its serviceaccount:

```yaml
rules:
  - verbs:
      - get
      - list
    apiGroups:
      - ''
    resources:
      - configmaps
      - secrets
```

If used with helm chart from this repo, this is done out-of-box with these values:

```yaml
rbac:
  enabled: true
  fromMetrics: true
```


### More examples

 - [Cluster-wide RBAC](examples/cluster_rbac.yaml)
 - [ConfigMaps and Secrets](examples/configmaps_and_secrets.yaml)
