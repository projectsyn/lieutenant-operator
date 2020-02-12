# Project Syn: Lieutenant Operator

Kubernetes Operator which implements the backend for [Lieutenant API](https://github.com/projectsyn/lieutenant-api).

**Please note that this project is in it's early stages and under active development**.

## Deployment

Deployment artifacts are available under `deploy/`. Please amend them to
your environment.

Example:

```
kubectl create ns syn-lieutenant
kubectl -n syn-lieutenant apply -f deploy/crds/
kubectl -n syn-lieutenant apply -f deploy/
```

Some example data to test the operator is available under `examples/`.

## Development

to be written

The Operator is implemented using the [Operator SDK](https://github.com/operator-framework/operator-sdk).
