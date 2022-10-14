# Changelog

# Changelog

## [0.1.3] - 2022-10-14
### Added
-  limit log collection by age #10
### Fixed
- block wildcard searches #33
- warn and continue when archiving paths that are nil (i.e. the "..data" paths you get in Dremio k8s) #34

## [0.1.2] - 2022-07-18
### Added
- dynamically find the gc.log #19
- recursively search all log and conf directories #27
### Fixed
- when archiving directory we hit an error #24

## [0.1.1] - 2022-07-18
### Added
- added dynamic container names with two new flags to use custom ones --coordinator-container and --executors-container
### Fixed
- the coordinator would never collect it's logs in kubernetes mode as the container name was incorrect

## [0.1.0] - 2022-07-18
### Added
- able to capture logs, configuration and diagnostic data from dremio clusters deployed on Kubernetes and on-prem
