[![Build Status](https://github.com/rsvihladremio/dremio-diagnostic-collector/actions/workflows/go.yml/badge.svg)](https://github.com/rsvihladremio/dremio-diagnostic-collector/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/rsvihladremio/dremio-diagnostic-collector)](https://goreportcard.com/report/github.com/rsvihladremio/dremio-diagnostic-collector)
[![Coverage Status](https://coveralls.io/repos/github/rsvihladremio/dremio-diagnostic-collector/badge.svg?branch=main)](https://coveralls.io/github/rsvihladremio/dremio-diagnostic-collector?branch=main)


# dremio-diagnostic-collector

collect logs of dremio for analysis

## Install

[Binaries are here](https://github.com/rsvihladremio/dremio-diagnostic-collector/releases)

### Mac

Use hombrew

```sh
brew tap rsvihladremio/ddc
brew install ddc
```

### Linux

Use shell script

```sh
/bin/bash -c "$(curl https://raw.githubusercontent.com/rsvihladremio/dremio-diagnostic-collector/main/script/install)"
```

### Windows

Use powershell script

```pwsh
Set-ExecutionPolicy RemoteSigned -Scope CurrentUser # Optional: Needed to run a remote script the first time
irm https://raw.githubusercontent.com/rsvihladremio/dremio-diagnostic-collector/main/script/install.ps1  | iex 
```
## Collection

As of the today the following is collected

* All top level logs in the specified log folder which is the -l or the --dremio-log-dir flag
* All top level conf files in the specified conf folder which is the -C or --dremio-conf-dir flag
* iostat diagnostic if it is available

### To collect from Kubernetes deployed clusters

Just need to specify the namespace and labels of the coordinators and the executors, next you can specify an output file with -o flag
.tgz, .zip, and .tar.gz are supported

```sh
/bin/ddc -k -e default:app=dremio-executor -c default:app=dremio-coordinator -o ~/Downloads/k8s-diag.tgz
```

### To collect from on-prem

This feature relies in ssh and [ssh public authentication](https://www.ssh.com/academy/ssh/public-key-authentication).
This is well documented in [Windows](https://docs.microsoft.com/en-us/windows-server/administration/openssh/openssh_keymanagement),
[Linux](https://www.redhat.com/sysadmin/key-based-authentication-ssh), and [Mac](https://www.linode.com/docs/guides/connect-to-server-over-ssh-on-mac/)

You must specify a --ssh-user and an -ssh-key the key must be configured to access the servers and not require a prompt to use (if encrypted using ssh agent will allow it to work).
The -e and -c flags will take a comma separated list of hosts

```sh
/bin/ddc -e 192.168.1.12,192.168.1.13 -c 192.168.1.19,192.168.1.2  --ssh-user ubuntu --ssh-key ~/.ssh/id_rsa -o ~/Downloads/k8s-diag.tgz
```

### more help

The help is pretty straight forward and comes with examples

```sh
ddc -h
ddc main-7375a13
ddc connects via ssh or kubectl and collects a series of logs and files for dremio, then puts those collected files in an archive
examples:

ddc --coordinator 10.0.0.19 --executors 10.0.0.20,10.0.0.21,10.0.0.22 --ssh-key $HOME/.ssh/id_rsa_dremio --output diag.zip

ddc --k8s --kubectl-path /opt/bin/kubectl --coordinator default:role=coordinator-dremio --executors default:role=executor-dremio --output diag.tar.gz

Usage:
  ddc [flags]

Flags:
  -c, --coordinator string                    coordinator node to connect to for collection
      --coordinator-container string          for use with -k8s flag: sets the container name to use to retrieve logs in the coordinators (default "dremio-master-coordinator")
  -d, --diag-tooling-collection-seconds int   the duration to run diagnostic collection tools like iostat, jstack etc (default 60)
  -C, --dremio-conf-dir string                directory where to find the configuration files for kubernetes this defaults to /opt/dremio/conf and for ssh this defaults to /etc/dremio/
  -l, --dremio-log-dir string                 directory where to find the logs (default "/var/log/dremio")
  -e, --executors string                      either a common separated list or a ip range of executors nodes to connect to
      --executors-container string            for use with -k8s flag: sets the container name to use to retrieve logs in the executors (default "dremio-executor")
  -h, --help                                  help for ddc
  -k, --k8s                                   use kubernetes to retrieve the diagnostics instead of ssh, instead of hosts pass in labels to the --cordinator and --executors flags
  -p, --kubectl-path string                   where to find kubectl (default "kubectl")
  -o, --output string                         filename of the resulting archived (tar) and compressed (gzip) file (default "diag.tgz")
  -s, --ssh-key string                        location of ssh key to use to login
  -u, --ssh-user string                       user to use during ssh operations to login
  -a, --log-age int                           the maximum number of days to go back for log retreival
  
```


## Developing

On Linux, Mac, and WSL there are some shell scripts modeled off the [GitHub ones](https://github.com/github/scripts-to-rule-them-all)

to get started run

```sh
./script/bootstrap
```

after a pull it is a good idea to run

```sh
./script/update
```

tests

```sh
./script/test
```

before checkin run

```sh
./script/cibuild
```

to cut a release do the following

```sh
#dont forget to update changelog.md with the release notes
git tag v0.1.1
git push origin v0.1.1
./script/release v0.1.1
gh repo view -w
# review the draft and when done set it to publish
```
### Windows
Similarly on Windows there are powershell scripts of the same design

to get started run

```powershell
.\script\bootstrap.ps1
```

after a pull it is a good idea to run

```powershell
.\script\update.ps1
```

tests

```powershell
.\script\test.ps1
```

before checkin run

```powershell
.\script\cibuild.ps1
```

to cut a release do the following

```powershell
#dont forget to update changelog.md with the release notes
git tag v0.1.1
.\script\release.ps1 v0.1.1
gh repo view -w
# review the draft and when done set it to publish
```

