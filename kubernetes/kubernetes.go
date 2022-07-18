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

//kubernetes package provides access to log collections on k8s
package kubernetes

import (
	"fmt"
	"log"
	"strings"

	"github.com/rsvihladremio/dremio-diagnostic-collector/cli"
)

// NewKubectlK8sActions is the only supported way to initialize the KubectlK8sActions struct
// one must pass the path to kubectl
func NewKubectlK8sActions(kubectlPath string) *KubectlK8sActions {
	return &KubectlK8sActions{
		cli:         &cli.Cli{},
		kubectlPath: kubectlPath,
	}
}

//KubectlK8sActions provides a way to collect and copy files using kubectl
type KubectlK8sActions struct {
	cli         cli.CmdExecutor
	kubectlPath string
}

func (c *KubectlK8sActions) getContainerName(isCoordinator bool) string {
	if isCoordinator {
		return "dremio-coordinator"
	}
	return "dremio-executor"
}

func (c *KubectlK8sActions) HostExecute(hostString string, isCoordinator bool, args ...string) (out string, err error) {
	tokens := strings.Split(hostString, ".")
	namespace := tokens[0]
	podName := tokens[1]
	kubectlArgs := []string{c.kubectlPath, "exec", "-it", "-n", namespace, "-c", c.getContainerName(isCoordinator), podName, "--"}
	kubectlArgs = append(kubectlArgs, args...)
	return c.cli.Execute(kubectlArgs...)
}

func (c *KubectlK8sActions) CopyFromHost(hostString string, isCoordinator bool, source, destination string) (out string, err error) {
	tokens := strings.Split(hostString, ".")
	namespace := tokens[0]
	podName := tokens[1]
	return c.cli.Execute(c.kubectlPath, "cp", "-n", namespace, "-c", c.getContainerName(isCoordinator), fmt.Sprintf("%v:%v", podName, source), destination)
}

func (c *KubectlK8sActions) FindHosts(searchTerm string) (podName []string, err error) {
	tokens := strings.Split(searchTerm, ":")
	namespace := tokens[0]
	labelName := tokens[1]
	out, err := c.cli.Execute(c.kubectlPath, "get", "pods", "-n", namespace, "-l", labelName, "-o", "name")
	if err != nil {
		return []string{}, err
	}
	rawPods := strings.Split(out, "\n")
	var pods []string
	for _, pod := range rawPods {
		if pod == "" {
			continue
		}
		rawPod := strings.TrimSpace(pod)
		log.Print(rawPod)
		pod := rawPod[4:]
		log.Print(pod)
		podWithNamespace := fmt.Sprintf("%v.%v", namespace, pod)
		pods = append(pods, podWithNamespace)
	}
	return pods, nil
}
