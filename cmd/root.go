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

//cmd package contains all the command line flag and initialization logic for commands
package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/rsvihladremio/dremio-diagnostic-collector/collection"
	"github.com/rsvihladremio/dremio-diagnostic-collector/kubernetes"
	"github.com/spf13/cobra"
)

var coordinatorStr string
var executorsStr string
var sshKeyLoc string
var outputLoc string
var kubectlPath string
var isK8s bool

//var isEmbeddedK8s bool
//var isEmbeddedSSH bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "dremio-diagnostic-collector",
	Short: "connects via to dremio servers collects logs into a zip file",
	Long: `connects via ssh or kubectl and collects a series of logs and files for dremio, then puts them in a zip folder for easy uploads
examples:

ddc --coordinator 10.0.0.10 --executors 10.0.0.20-10.0.0.30 --ssh-key $HOME/.ssh/id_rsa_dremio --output diag.zip

ddc --k8s --kubectl-path /opt/bin/kubectl --coordinator coordinator-dremio --executors executor-dremio --output diag.zip
`,
	Run: func(cmd *cobra.Command, args []string) {
		if sshKeyLoc == "" {
			sshDefault, err := sshDefault()
			if err != nil {
				log.Fatalf("unexpected error getting ssh directory '%v'. This is a critical error and should result in a bug report.", err)
			}
			sshKeyLoc = sshDefault
		}
		logOutput := os.Stdout
		var collectorStrategy collection.Collector
		if isK8s {
			log.Print("using Kubernetes kubectl based collection")
			collectorStrategy = kubernetes.NewKubectlK8sActions(kubectlPath)
		} else {
			log.Print("using SSH based collection")
		}
		err := collection.Execute(collectorStrategy, coordinatorStr, executorsStr, outputLoc, logOutput)
		if err != nil {
			log.Fatalf("unexpected error running kubernetes collection '%v'", err)
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

	rootCmd.Flags().StringVarP(&coordinatorStr, "coordinator", "c", "", "coordinator node to connect to for colleciton")
	rootCmd.Flags().StringVarP(&executorsStr, "executors", "e", "", "either a common separated list or a ip range of executors nodes to connect to")
	rootCmd.Flags().StringVarP(&sshKeyLoc, "ssh-key", "s", "", "location of ssh key to use to login to the hosts specified")
	rootCmd.Flags().StringVarP(&outputLoc, "output", "o", "diag.zip", "either a common separated list or a ip range of executors nodes to connect to")
	rootCmd.Flags().StringVarP(&kubectlPath, "kubectl-path", "p", "kubectl", "where to find kubectl")
	rootCmd.Flags().BoolVarP(&isK8s, "k8s", "k", false, "use kubernetes to retrieve the diagnostics instead of ssh, instead of hosts pass in labels to the --cordinator and --executors flags")
	// TODO implement embedded k8s and ssh support using go libs
	//rootCmd.Flags().BoolVar(&isEmbeddedK8s, "embedded-k8s", false, "use embedded k8s client in place of kubectl binary")
	//rootCmd.Flags().BoolVar(&isEmbeddedSSH, "embedded-ssh", false, "use embedded ssh go client in place of ssh and scp binary")
}
