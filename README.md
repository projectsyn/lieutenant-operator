# Project Syn: Lieutenant Operator

Kubernetes Operator which implements the backend for [Lieutenant API](https://github.com/projectsyn/lieutenant-api).

The operator keeps inventory about all the tenants and clusters in a SYN managed k8s cluster.

It also handles the management of some requirements like Git repositories and secret management. It can automatically populate Git repositories with template files when a new cluster is added. It will also generate a token to be used by Steward.


**Please note that this project is in it's early stages and under active development**.

This repository is part of Project Syn.
For documentation on Project Syn and this component, see https://syn.tools.

## Documentation

Documentation for this component is written using [Asciidoc][asciidoc] and [Antora][antora].
It is located in the [docs/](docs) folder.
The [Divio documentation structure](https://documentation.divio.com/) is used to organize its content.

You can use the `make docs-serve` command and then browse to http://localhost:2020 to preview the documentation.

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

## Contributing and license

This library is licensed under [BSD-3-Clause](LICENSE).
For information about how to contribute see [CONTRIBUTING](CONTRIBUTING.md).

[commodore]: https://docs.syn.tools/commodore/index.html
[asciidoc]: https://asciidoctor.org/
[antora]: https://antora.org/

