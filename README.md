[![Build Status](https://github.com/rsvihladremio/dremio-diagnostic-collector/actions/workflows/go.yml/badge.svg)](https://github.com/rsvihladremio/dremio-diagnostic-collector/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/rsvihladremio/dremio-diagnostic-collector)](https://goreportcard.com/report/github.com/rsvihladremio/dremio-diagnostic-collector)
[![Coverage Status](https://coveralls.io/repos/github/rsvihladremio/dremio-diagnostic-collector/badge.svg?branch=main)](https://coveralls.io/github/rsvihladremio/dremio-diagnostic-collector?branch=main)


# dremio-diagnostic-collector

collect logs of dremio for analysis


## Quick Start

On Linux, Mac and WSL run the following script

```sh
/bin/bash -c "$(curl https://raw.githubusercontent.com/rsvihladremio/dremio-diagnostic-collector/main/script/install)"
```

For all other platforms consult the [installation options](docs/install.md)

### dremio on k8s

Just need to specify the namespace and labels of the coordinators and the executors, next you can specify an output file with -o flag
.tgz, .zip, and .tar.gz are supported

```sh
/bin/ddc -k -n default -e app=dremio-executor -c app=dremio-coordinator -o ~/Downloads/k8s-diag.tgz
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

* All top level logs in the specified log folder which is the -l or the --dremio-log-dir flag
* All top level conf files in the specified conf folder which is the -C or --dremio-conf-dir flag
* iostat diagnostic if it is available
* Java flight recorder (JFR) if required

### Flight recorder (JFR)

If you are running JFRs then please see [jfr docs](docs/JFR.md)

### more help

The help is pretty straight forward and comes with examples

```sh
ddc v0.2.1
ddc connects via ssh or kubectl and collects a series of logs and files for dremio, then puts those collected files in an archive
examples:

ddc --coordinator 10.0.0.19 --executors 10.0.0.20,10.0.0.21,10.0.0.22 --ssh-key $HOME/.ssh/id_rsa_dremio --output diag.zip

ddc --k8s --kubectl-path /opt/bin/kubectl --namespace mynamespace --coordinator app=dremio-coordinator --executors app=dremio-executor --output diag.zip

Usage:
  ddc [flags]

Flags:
  -c, --coordinator string                    coordinator to connect to for collection. With ssh set a list of ip addresses separated by commas. In K8s use a label that matches to the pod(s).
      --coordinator-container string          for use with -k8s flag: sets the container name to use to retrieve logs in the coordinators (default "dremio-master-coordinator")
  -d, --diag-tooling-collection-seconds int   the duration to run diagnostic collection tools like iostat, jstack etc (default 60)
  -C, --dremio-conf-dir string                directory where to find the configuration files for kubernetes this defaults to /opt/dremio/conf and for ssh this defaults to /etc/dremio/
  -g, --dremio-gc-dir string                  directory where to find the GC logs (default "/var/log/dremio")
  -l, --dremio-log-dir string                 directory where to find the logs (default "/var/log/dremio")
  -x, --exclude-files strings                 comma seperated list of file names to exclude (default [*jfr])
  -e, --executors string                      either a common separated list or a ip range of executors nodes to connect to. With ssh set a list of ip addresses separated by commas. In K8s use a label that matches to the pod(s).
      --executors-container string            for use with -k8s flag: sets the container name to use to retrieve logs in the executors (default "dremio-executor")
  -f, --format string                         format for output, (choices are "healthcheck" (default) and "basic" (default "healthcheck")
  -h, --help                                  help for ddc
  -j, --jfr int                               enables collection of java flight recorder (jfr), time specified in seconds
  -k, --k8s                                   use kubernetes to retrieve the diagnostics instead of ssh, instead of hosts pass in labels to the --cordinator and --executors flags
  -p, --kubectl-path string                   where to find kubectl (default "kubectl")
  -a, --log-age int                           the maximum number of days to go back for log retreival (default is no filter and will retrieve all logs)
  -n, --namespace string                      namespace to use for kubernetes pods (default "default")
  -o, --output string                         filename of the resulting archived (tar) and compressed (gzip) file (default "diag.tgz")
  -s, --ssh-key string                        location of ssh key to use to login
  -u, --ssh-user string                       user to use during ssh operations to login
  -b, --sudo-user string                      if any diagnostcs commands need a sudo user (i.e. for jcmd)
```



