[![Build Status](https://github.com/rsvihladremio/dremio-diagnostic-collector/actions/workflows/go.yml/badge.svg)](https://github.com/rsvihladremio/dremio-diagnostic-collector/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/rsvihladremio/dremio-diagnostic-collector)](https://goreportcard.com/report/github.com/rsvihladremio/dremio-diagnostic-collector)
![Coverage Status](https://img.shields.io/endpoint?url=https%3A%2F%2Fgist.githubusercontent.com%2Frsvihladremio%2Fbaa5764cecee421db0f533239258c064%2Fraw%2Fdremio-diagnostic-collector-go-coverage.json)

# dremio-diagnostic-collector

collect logs of dremio for analysis


### Install

Download the [latest release binaries](https://github.com/rsvihladremio/dremio-diagnostic-collector/releases/latest):

1. unzip the binary
2. open a terminal
3. change to the directory where you unzip your binary
4. run the command `./ddc -h` if you get the help for the command you are good to go.


### ddc.yaml

The ddc.yaml file is located next to your ddc binary. The defaults for the ddc.yaml are optimized for k8s and the dremio helm charts. However, if you have a custom installation or are using this with SSH you will likely have to edit the the ddc.yaml file included. 

```yaml
verbose: vvvv
collect-acceleration-log: false
collect-access-log: false
dremio-gclogs-dir: "" # if left blank detection is used to find the gc log dir
dremio-log-dir: "/opt/dremio/data/log" # where the dremio log is located
dremio-conf-dir: "/opt/dremio/conf" #where the dremio conf files are located
dremio-rocksdb-dir: /opt/dremio/data/db # used for locating Dremio's KV Metastore
number-threads: 2 #number of threads to use for collection
dremio-endpoint: "http://localhost:9047" # dremio endpoint on each node to use for collecting Workload Manager, KV Report and Job Profiles
dremio-username: "dremio" # dremio user to for collecting Workload Manager, KV Report and Job Profiles 
dremio-pat-token: "" # when set will attempt to collect Workload Manager, KV report and Job Profiles. Dremio PATs can be enabled by the support key auth.personal-access-tokens.enabled
collect-dremio-configuration: true # will collect dremio.conf, dremio-env, logback.xml and logback-access.xml
number-job-profiles: 25000 # need to have the dremio-pat-token set to work
capture-heap-dump: false # when true a heap dump will be captured on each node that the collector is run against
accept-collection-consent: true # when true you accept consent to collect data on each node, if false collection will fail
```
After you have adjusted the yaml to your liking run ddc with either the k8s or on prem options

### dremio on k8s

Just need to specify the namespace and labels of the coordinators and the executors, next you can specify an output file with -o flag
.tgz, .zip, and .tar.gz are supported

```sh
/bin/ddc -k -n default -e app=dremio-executor -c app=dremio-coordinator - -o ~/Downloads/k8s-diag.tgz
```

If you have issues consult the [k8s docs](docs/k8s.md)

### dremio on prem

specific executors that you want to collect from with the -e flag and coordinators with the -c flag. Specify ssh user, and ssh key to use.

```sh
/bin/ddc -e 192.168.1.12,192.168.1.13 -c 192.168.1.19,192.168.1.2  --ssh-user ubuntu --ssh-key ~/.ssh/id_rsa -o ~/Downloads/k8s-diag.tgz
```

If you have issues consult the [ssh docs](docs/ssh.md)

## What is collected?

As of the today the following is collected

### By default

* Linux perf metrics (io, cpu, bytes read and written, io wait, etc)
* System disk usage
* Java Flight Recorder recording of 60 seconds
* Jstack thread dump every second for approximately 60 seconds
* server.log and 7 days of archives
* metadata_refresh.log and 7 days of archives
* reflection.log and 7 days of archives
* queries.json and up to 28 days of archives 
* all dremio configurations
* All gc logs if present

### Optionally with the appropriate change to ddc.yaml

* access.log and 7 days of archives
* audit.log and 7 days of archvies
* java heap dump

### Optionally with a Dremio Personal Access Token

* a sampling of job profiles (note 25000 jobs can take 15 minutes to collect)
* dremio key value store report
* dremio work load manager details
* system tables and their details


### Full Help

The help is pretty straight forward and comes with examples

```sh
ddc v0.3.0
ddc connects via ssh or kubectl and collects a series of logs and files for dremio, then puts those collected files in an archive
examples:

ddc --coordinator 10.0.0.19 --executors 10.0.0.20,10.0.0.21,10.0.0.22 --ssh-key $HOME/.ssh/id_rsa_dremio 

ddc --k8s --namespace mynamespace --coordinator app=dremio-coordinator --executors app=dremio-executor

Usage:
  ddc [flags]
  ddc [command]

Available Commands:
  completion    Generate the autocompletion script for the specified shell
  help          Help about any command
  local-collect retrieves all the dremio logs and diagnostics for the local node and saves the results in a compatible format for Dremio support
  version       Print the version number of DDC

Flags:
  -c, --coordinator string             a common separated list of coordinators to connect to for collection. With ssh set a list of ip addresses separated by commas. In K8s use a label that matches to the pod(s).
      --coordinator-container string   for use with -k8s flag: sets the container name to use to retrieve logs in the coordinators (default "dremio-master-coordinator")
  -e, --executors string               a common separated list of executors to connect for collection.  With ssh set a list of ip addresses separated by commas. In K8s use a label that matches to the pod(s).
      --executors-container string     for use with -k8s flag: sets the container name to use to retrieve logs in the executors (default "dremio-executor")
  -h, --help                           help for ddc
  -k, --k8s                            use kubernetes to retrieve the diagnostics instead of ssh, instead of hosts pass in labels to the --coordinator and --executors flags
  -n, --namespace string               namespace to use for kubernetes pods (default "default")
  -s, --ssh-key string                 location of ssh key to use to login
  -u, --ssh-user string                user to use during ssh operations to login
  -b, --sudo-user string               if any diagnostcs commands need a sudo user (i.e. for jcmd)
```



