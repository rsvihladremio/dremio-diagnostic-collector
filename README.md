[![Go Report Card](https://goreportcard.com/badge/github.com/dremio/dremio-diagnostic-collector)](https://goreportcard.com/report/github.com/dremio/dremio-diagnostic-collector)
![Coverage Status](https://img.shields.io/badge/Code_Coverage-71%25-yellow)


Automated log and analytics collection for Dremio clusters

## IMPORTANT LINKS

* Read the [FAQ](FAQ.md) for common questions on setting up DDC
* Read the [ddc.yaml](default-ddc.yaml) for a full, detailed list of customizable collection parameters (optional)
* Read the [official Dremio Support page](https://support.dremio.com/hc/en-us/articles/15560006579739) for more details on the DDC architecture

### Install DDC on your local machine

Download the [latest release binary](https://github.com/dremio/dremio-diagnostic-collector/releases/latest):

1. Unzip the binary
2. Open a terminal and change to the directory where you unzipped your binary
3. Run the command `./ddc -h`. If you see the DDC command help, you are good to go.

### Dremio on Kubernetes

DDC connects via SSH or `kubectl` and collects a series of logs and files for Dremio, then puts those collected files in an archive

For Kubernetes deployments _(Requires `kubectl` cluster access to be configured)_:

##### coordinator only
```bash
ddc --k8s --namespace mynamespace --coordinator app=dremio-coordinator 
```
    
##### coordinator and executors
```bash
ddc --k8s --namespace mynamespace --coordinator app=dremio-coordinator --executors app=dremio-executor 
```
      
##### to collect job profiles, system tables, kv reports and wlm (via REST API)
_Requires Dremio admin privileges. Dremio PATs can be enabled by the support key `auth.personal-access-tokens.enabled`_
```bash
ddc --k8s -n mynamespace -c app=dremio-coordinator -e app=dremio-executor --dremio-pat-prompt
```

### Dremio on-prem

Specific executors that you want to collect from with the `-e` flag and coordinators with the `-c` flag. Specify SSH user, and SSH key to use.


For SSH based communication to VMs or Bare Metal hardware:

##### coordinator only

```bash
ddc --coordinator 10.0.0.19 --ssh-user myuser 
```    
##### coordinator and executors
        
```bash
ddc --coordinator 10.0.0.19 --executors 10.0.0.20,10.0.0.21,10.0.0.22 --ssh-user myuser
```

##### to collect job profiles, system tables, kv reports and wlm (via REST API)
_Requires Dremio admin privileges. Dremio PATs can be enabled by the support key `auth.personal-access-tokens.enabled`_
```bash
ddc --coordinator 10.0.0.19 --executors 10.0.0.20,10.0.0.21,10.0.0.22 --ssh-user myuser --dremio-pat-prompt
```    
    
##### to avoid using the /tmp folder on nodes

```bash
ddc --coordinator 10.0.0.19 --executors 10.0.0.20,10.0.0.21,10.0.0.22 --ssh-user myuser --transfer-dir /mnt/lots_of_storage/
```

### Dremio AWSE

Log-only collection from a Dremio AWSE coordinator is possible via the following command. This will produce a tarball with logs from all nodes.

```bash
./ddc awselogs
```

### Dremio Cloud
To collect job profiles, system tables, and wlm via REST API, specify the following parameters in `ddc.yaml`
```yaml
is-dremio-cloud: true
dremio-endpoint: "[eu.]dremio.cloud"    # Specify whether EU Dremio Cloud or not
dremio-cloud-project-id: "<PROJECT_ID>"
dremio-pat-token: "<DREMIO_PAT>"
tmp-output-dir: /full/path/to/dir       # Specify local target directory
```
and run `./ddc local-collect` from your local machine

### Windows Users

If you are running DDC from Windows, always run in a shell from the `C:` drive prompt. 
This is because of a limitation of kubectl ( see https://github.com/kubernetes/kubernetes/issues/77310 )

### ddc.yaml

The `ddc.yaml` file is located next to your DDC binary and can be edited to fit your environment. The [default-ddc.yaml](default-ddc.yaml) documents the full list of available parameters.

### Flags


```sh
Available Commands:
  awselogs      Log only collect of AWSE from the coordinator node
  completion    Generate the autocompletion script for the specified shell
  help          Help about any command
  local-collect retrieves all the dremio logs and diagnostics for the local node and saves the results in a compatible format for Dremio support
  version       Print the version number of DDC

Flags:
  -c, --coordinator string             coordinator to connect to for collection. With ssh set a list of ip addresses separated by commas. In K8s use a label that matches to the pod(s).
      --coordinator-container string   for use with -k8s flag: sets the container name to use to retrieve logs in the coordinators (default "dremio-master-coordinator,dremio-coordinator")
      --ddc-yaml string                location of ddc.yaml that will be transferred to remote nodes for collection configuration (default "/Users/ryan.svihla/Documents/GitHub/dremio-diagnostic-collector/bin/ddc.yaml")
  -t, --dremio-pat-prompt              Prompt for Dremio Personal Access Token (PAT)
  -e, --executors string               either a common separated list or a ip range of executors nodes to connect to. With ssh set a list of ip addresses separated by commas. In K8s use a label that matches to the pod(s).
      --executors-container string     for use with -k8s flag: sets the container name to use to retrieve logs in the executors (default "dremio-executor")
  -h, --help                           help for ddc
  -k, --k8s                            use kubernetes to retrieve the diagnostics instead of ssh, instead of hosts pass in labels to the --coordinator and --executors flags
  -p, --kubectl-path string            where to find kubectl (default "kubectl")
  -n, --namespace string               namespace to use for kubernetes pods (default "default")
      --output-file string             name of tgz file to save the diagnostic collection to (default "diag.tgz")
  -s, --ssh-key string                 location of ssh key to use to login
  -u, --ssh-user string                user to use during ssh operations to login
  -b, --sudo-user string               if any diagnostics commands need a sudo user (i.e. for jcmd)
      --transfer-dir string            directory to use for communication between the local-collect command and this one (default "/tmp/ddc-20240112141658")

```
