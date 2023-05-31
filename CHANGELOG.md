# Changelog
## [0.3.0-beta4]

### Changed

* job profiles now threads
* new improved thread pool that executes more consistently instead of in bunches
* support for more than just gzipped archive logs

### Fixed

* archiving of logs now works correctly and will grab several days of logs

## [0.3.0-beta3]

### Added
* added logging for configuration as it has been picked up

### Fixed
* jstack was not pausing long enough between iterations
* dont delete jfr after capturing it. fixes #100
* removed old $ variables fixes #102
* verbosity was broken, now works even if someone asks for more verbosity than we have
* defaults were not set for WLM, export system tables and kvm store report
* inverted check for wlm, kv store, dremio config, disk usage and job profile collections fixes #104 and #98
* command line flags are only bound to viper if they are passed
* command line flag variables are now parsed directly from viper, this allows configuration to override the flags fixed #103
* if archive folder is not available just log an error and skip it
* cli flags take precidence

## [0.3.0-beta2]

### Changed

* will no longer collect job profiles when queries.json is not present
  on a node and will log a warning when job collection is enabled.

### Fixed

* fixes https://github.com/rsvihladremio/dremio-diagnostic-collector/issues/80 now releases that are not linux contain an extra binary and example ddc.yaml file editing the file will change the parameters that run remotely on the server
* fixed  https://github.com/rsvihladremio/dremio-diagnostic-collector/issues/86 local capture metrics were not working
* old thread pool was buggy, wrote new one by hand, works well
* removed a lot of code around capture and collection as it is no longer needed
* logging to two sources, and now includes logs in the final tarball
* local-collect will archive what is had now fixed https://github.com/rsvihladremio/dremio-diagnostic-collector/issues/84
* fixes https://github.com/rsvihladremio/dremio-diagnostic-collector/issues/77 removing k8s from local-collect
* fixes https://github.com/rsvihladremio/dremio-diagnostic-collector/issues/78 consent formatting
* fixes https://github.com/rsvihladremio/dremio-diagnostic-collector/issues/79 consent was backwards now works as expected


## [0.3.0-beta1] - 2023-05-15

### Added

- Local capture command that can run locally on a node. 
- Added configuration file support ddc-capture.yaml in local folder (also supports json, toml, hcl, env, and props file formats). The configuration options are the same name as the flags. Run ddc local-capture --help for more information.

### Changed

- removed all formats except the health check format

## [0.2.2] - 2023-05-15

### Added
- support for sudo


## [0.2.1] - 2023-02-14

### Added

- Easier to use http client

### Fixed

-  Excluding files did not accept wildcards
-  Fix k8s namespace syntax. Use `-n` option instead of `namespace:label`
-  AWSE deployments can have bundle multiple dupe named files under one IP

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

[0.3.0-beta4]: https://github.com/rsvihladremio/dremio-diagnostic-collector/compare/v0.3.0-beta3...v0.3.0-beta4
[0.3.0-beta3]: https://github.com/rsvihladremio/dremio-diagnostic-collector/compare/v0.3.0-beta2...v0.3.0-beta3
[0.3.0-beta2]: https://github.com/rsvihladremio/dremio-diagnostic-collector/compare/v0.3.0-beta1...v0.3.0-beta2
[0.3.0-beta1]: https://github.com/rsvihladremio/dremio-diagnostic-collector/compare/v0.2.2...v0.3.0-beta1
[0.2.2]: https://github.com/rsvihladremio/dremio-diagnostic-collector/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/rsvihladremio/dremio-diagnostic-collector/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/rsvihladremio/dremio-diagnostic-collector/compare/v0.1.5...v0.2.0
[0.1.5]: https://github.com/rsvihladremio/dremio-diagnostic-collector/compare/v0.1.4...v0.1.5
[0.1.4]: https://github.com/rsvihladremio/dremio-diagnostic-collector/compare/v0.1.3...v0.1.4
[0.1.3]: https://github.com/rsvihladremio/dremio-diagnostic-collector/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/rsvihladremio/dremio-diagnostic-collector/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/rsvihladremio/dremio-diagnostic-collector/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/rsvihladremio/dremio-diagnostic-collector/releases/tag/v0.1.0
