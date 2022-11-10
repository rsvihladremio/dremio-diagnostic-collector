/*
   Copyright 2022 Ryan SVIHLA

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

// cmd package contains all the command line flag and initialization logic for commands
package cmd

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/rsvihladremio/dremio-diagnostic-collector/collection"
	"github.com/rsvihladremio/dremio-diagnostic-collector/kubernetes"
	"github.com/rsvihladremio/dremio-diagnostic-collector/ssh"
	"github.com/spf13/cobra"
)

var dremioConfDir string
var dremioLogDir string
var dremioGcDir string
var coordinatorContainer string
var executorsContainer string
var coordinatorStr string
var executorsStr string
var sshKeyLoc string
var sshUser string
var outputLoc string
var kubectlPath string
var isK8s bool
var durationDiagnosticTooling int
var logAge int
var jfrduration int
var sudoUser string
var excludeFiles []string
var GitSha = "unknown"
var Version = "dev"

// var isEmbeddedK8s bool
// var isEmbeddedSSH bool
func getVersion() string {
	return fmt.Sprintf("ddc %v-%v\n", Version, GitSha)
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ddc",
	Short: getVersion() + "ddc connects via to dremio servers collects logs into an archive",
	Long: getVersion() + `ddc connects via ssh or kubectl and collects a series of logs and files for dremio, then puts those collected files in an archive
examples:

ddc --coordinator 10.0.0.19 --executors 10.0.0.20,10.0.0.21,10.0.0.22 --ssh-key $HOME/.ssh/id_rsa_dremio --output diag.zip

ddc --k8s --kubectl-path /opt/bin/kubectl --coordinator default:app=dremio-coordinator-dremio --executors default:app=dremio-executor --output diag.tar.gz
`,
	Run: func(cmd *cobra.Command, args []string) {
		if sshKeyLoc == "" {
			sshDefault, err := sshDefault()
			if err != nil {
				log.Fatalf("unexpected error getting ssh directory '%v'. This is a critical error and should result in a bug report.", err)
			}
			sshKeyLoc = sshDefault
		}
		// Update paths to ensure if run on windows that we still use a forward slash "/"
		if dremioConfDir == "" {
			if isK8s {
				dremioConfDir = "/opt/dremio/conf/..data/"
			} else {
				dremioConfDir = "/etc/dremio/"
			}
		}
		if dremioLogDir == "" {
			if isK8s {
				dremioConfDir = "/opt/dremio/data/log/"
			} else {
				dremioConfDir = "/var/log/dremio/"
			}
		}
		logOutput := os.Stdout

		collectionArgs := collection.Args{
			CoordinatorStr:            coordinatorStr,
			ExecutorsStr:              executorsStr,
			OutputLoc:                 filepath.Clean(outputLoc),
			DremioConfDir:             filepath.Clean(dremioConfDir),
			DremioLogDir:              filepath.Clean(dremioLogDir),
			GCLogOverride:             filepath.Clean(dremioGcDir),
			DurationDiagnosticTooling: durationDiagnosticTooling,
			LogAge:                    logAge,
			JfrDuration:               jfrduration,
			SudoUser:                  sudoUser,
			ExcludeFiles:              excludeFiles,
		}

		// All dremio deployments will be Linux based so we have to switch the path seperator on these two elements
		// since https://pkg.go.dev/path/filepath?utm_source=gopls#Clean shows that Clean will replace the slash with OS local seperator
		confdir := collectionArgs.DremioConfDir
		logdir := collectionArgs.DremioLogDir
		collectionArgs.DremioConfDir = strings.Replace(confdir, `\`, `/`, -1)
		collectionArgs.DremioLogDir = strings.Replace(logdir, `\`, `/`, -1)

		err := validateParameters(collectionArgs, sshKeyLoc, sshUser, isK8s)
		if err != nil {
			fmt.Println("COMMAND HELP TEXT:")
			fmt.Println("")
			err := cmd.Help()
			if err != nil {
				log.Fatalf("unable to print help %v", err)
			}
			fmt.Println("")
			fmt.Println("")
			fmt.Printf("Invalid command flag detected: %v\n", err)
			fmt.Println("")
			os.Exit(1)
		}
		fmt.Println(getVersion())
		var collectorStrategy collection.Collector
		if isK8s {
			log.Print("using Kubernetes kubectl based collection")
			collectorStrategy = kubernetes.NewKubectlK8sActions(kubectlPath, coordinatorContainer, executorsContainer)
		} else {
			log.Print("using SSH based collection")
			collectorStrategy = ssh.NewCmdSSHActions(sshKeyLoc, sshUser)
		}
		// Create ref to real file system (since with testing we redirect the argument to a mock object)
		err = collection.Execute(collectorStrategy,
			logOutput,
			collectionArgs,
		)

		if err != nil {
			log.Fatalf("unexpected error running collection '%v'", err)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}

type unableToGetHomeDir struct {
	Err error
}

func (u unableToGetHomeDir) Error() string {
	return fmt.Sprintf("unable to get home dir '%v'", u.Err)
}

// sshDefault returns the default .ssh key typically used on most deployments

func sshDefault() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", unableToGetHomeDir{}
	}
	return filepath.Join(home, ".ssh", "id_rsa"), nil
}

func init() {
	// command line flags

	rootCmd.Flags().StringVar(&coordinatorContainer, "coordinator-container", "dremio-master-coordinator", "for use with -k8s flag: sets the container name to use to retrieve logs in the coordinators")
	rootCmd.Flags().StringVar(&executorsContainer, "executors-container", "dremio-executor", "for use with -k8s flag: sets the container name to use to retrieve logs in the executors")
	rootCmd.Flags().StringVarP(&coordinatorStr, "coordinator", "c", "", "coordinator node to connect to for collection")
	rootCmd.Flags().StringVarP(&executorsStr, "executors", "e", "", "either a common separated list or a ip range of executors nodes to connect to")
	rootCmd.Flags().StringVarP(&sshKeyLoc, "ssh-key", "s", "", "location of ssh key to use to login")
	rootCmd.Flags().StringVarP(&sshUser, "ssh-user", "u", "", "user to use during ssh operations to login")
	rootCmd.Flags().StringVarP(&outputLoc, "output", "o", "diag.tgz", "filename of the resulting archived (tar) and compressed (gzip) file")
	rootCmd.Flags().StringVarP(&kubectlPath, "kubectl-path", "p", "kubectl", "where to find kubectl")
	rootCmd.Flags().BoolVarP(&isK8s, "k8s", "k", false, "use kubernetes to retrieve the diagnostics instead of ssh, instead of hosts pass in labels to the --cordinator and --executors flags")
	rootCmd.Flags().StringVarP(&dremioConfDir, "dremio-conf-dir", "C", "", "directory where to find the configuration files for kubernetes this defaults to /opt/dremio/conf and for ssh this defaults to /etc/dremio/")
	rootCmd.Flags().StringVarP(&dremioLogDir, "dremio-log-dir", "l", "/var/log/dremio", "directory where to find the logs")
	rootCmd.Flags().IntVarP(&durationDiagnosticTooling, "diag-tooling-collection-seconds", "d", 60, "the duration to run diagnostic collection tools like iostat, jstack etc")
	rootCmd.Flags().IntVarP(&logAge, "log-age", "a", 0, "the maximum number of days to go back for log retreival (default is no filter and will retrieve all logs)")
	rootCmd.Flags().StringVarP(&dremioGcDir, "dremio-gc-dir", "g", "/var/log/dremio", "directory where to find the GC logs")
	rootCmd.Flags().IntVarP(&jfrduration, "jfr", "j", 0, "enables collection of java flight recorder (jfr), time specified in seconds")
	rootCmd.Flags().StringVarP(&sudoUser, "sudo-user", "b", "", "if any diagnostcs commands need a sudo user (i.e. for jcmd)")
	rootCmd.Flags().StringSliceVarP(&excludeFiles, "exclude-files", "x", []string{"*jfr"}, "comma seperated list of file names to exclude")

	// TODO implement embedded k8s and ssh support using go libs
	//rootCmd.Flags().BoolVar(&isEmbeddedK8s, "embedded-k8s", false, "use embedded k8s client in place of kubectl binary")
	//rootCmd.Flags().BoolVar(&isEmbeddedSSH, "embedded-ssh", false, "use embedded ssh go client in place of ssh and scp binary")

}

func validateParameters(args collection.Args, sshKeyLoc, sshUser string, isK8s bool) error {
	if args.CoordinatorStr == "" {
		if isK8s {
			return errors.New("the coordinator string was empty you must pass a namespace, a colon and a label that will match your coordinators --coordinator or -c arguments. Example: -c \"default:mylabel=coordinator\"")
		}
		return errors.New("the coordinator string was empty you must pass a single host or a comma separated lists of hosts to --coordinator or -c arguments. Example: -e 192.168.64.12,192.168.65.10")
	}
	if args.ExecutorsStr == "" {
		if isK8s {
			return errors.New("the executor string was empty you must pass a namespace, a colon and a label that will match your executors --executor or -e arguments. Example: -e \"default:mylabel=executor\"")
		}
		return errors.New("the executor string was empty you must pass a single host or a comma separated lists of hosts to --executor or -e arguments. Example: -e 192.168.64.12,192.168.65.10")
	}

	if !isK8s {
		if sshKeyLoc == "" {
			return errors.New("the ssh private key location was empty, pass --ssh-key or -s with the key to get past this error. Example --ssh-key ~/.ssh/id_rsa")
		}
		if sshUser == "" {
			return errors.New("the ssh user was empty, pass --ssh-user or -u with the user name you want to use to get past this error. Example --ssh-user ubuntu")
		}
	}
	return nil
}
