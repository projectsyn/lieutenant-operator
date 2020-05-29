# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## unreleased
### Added
- Ability to configure sync interval

## v0.1.3 - 2020-05-15
### Added
- Create a `common.yml` class for each tenant

## v0.1.2 - 2020-05-11
### Fixed
- Reconcile status

## v0.1.1 - 2020-05-08
### Added
- Docs from the SDD
- Doc generator from CRDs
- GitRepo file templates
- Add an empty file for each cluster to the tenant git repo
- Add the cluster service account token to Vault
- Implement DisplayName for GitRepo objects
### Fixed
- A race condition in the reconcile loop

## v0.0.5 - 2020-02-27
### Fixed
- Key comparison issues when using multiline strings in YAML
- GitRepoURL not being set for clusters/tenants

## v0.0.4 - 2020-02-17
### Added
- Example CRs
- Documentation how to deploy
### Changed
- Implement git host keys
- Only update status if GitRepo was alredy created
- Fix token handling
- Add Age column to CRDs
- Use local namespace as default for secretRef
- Add tenant label to GitRepos
- Reuse object names for GitRepo names
### Fixed
- GitLab subgroup handling
- GitRepos properly updated from Clusters and Tenants

## v0.0.3 - 2020-02-10
### Added
- GitLab implementation for managing git repos
- Changelog
- RBAC management for clusters
