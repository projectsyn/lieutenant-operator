# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased
### Added
- Docs from the SDD
- Doc generator from CRDs

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
- Gitlab subgroup handling
- GitRepos properly updated from Clusters and Tenants

## v0.0.3 - 2020-02-10
### Added
- Gitlab implementation for managing git repos
- Changelog
- RBAC management for clusters
