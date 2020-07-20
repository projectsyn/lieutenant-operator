# Project Syn: Lieutenant Operator

Kubernetes Operator which implements the backend for [Lieutenant API](https://github.com/projectsyn/lieutenant-api).

The operator keeps inventory about all the tenants and clusters in a SYN managed k8s cluster.

It also handles the management of some requirements like Git repositories and secret management. It can automatically populate Git repositories with template files when a new cluster is added. It will also generate a token to be used by Steward.


**Please note that this project is in it's early stages and under active development**.

## Deployment

A Kustomize setup is available under `deploy/`.

Example:

```
kubectl create ns syn-lieutenant
kubectl -n syn-lieutenant apply -k deploy/crds/
kubectl -n syn-lieutenant apply -k deploy/
```

Some example data to test the operator is available under `examples/`.

## Development

to be written

The Operator is implemented using the [Operator SDK](https://github.com/operator-framework/operator-sdk).
