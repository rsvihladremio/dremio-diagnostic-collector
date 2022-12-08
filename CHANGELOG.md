# Changelog

## [0.2.0] - 2022-12-08
### Added
- Support for healthcheck format
- Support for basic format and other future formats
- Kubernetes cluster level config collection

### Fixed
- Issues with windows path formatting

## [0.1.5] - 2022-11-10
### Added
- Support for JFR collection
- Ability to exclude files (exact match)
- Flag to specify GC log location

### Fixed
- Collected file timestamps being reset to 1970 dates

## [0.1.4] - 2022-10-26
### Added
- testing for windows environments
### Fixed
- newer version of golang ci
- moved errors to defer calls
- running ddc from windows sent the wrong path seperator to pods
- running ddc from windows causes syntaxual issues with drive letter notation


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
