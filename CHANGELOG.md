# Changelog

## [0.1.1] - 2022-07-18
### Added
- added dynamic container names with two new flags to use custom ones --coordinator-container and --executors-container
### Fixed
- the coordinator would never collect it's logs in kubernetes mode as the container name was incorrect

## [0.1.0] - 2022-07-18
### Added
- able to capture logs, configuration and diagnostic data from dremio clusters deployed on Kubernetes and on-prem
