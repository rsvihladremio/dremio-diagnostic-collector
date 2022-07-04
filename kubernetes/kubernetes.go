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
	"strings"

	"github.com/rsvihladremio/dremio-diagnostic-collector/cli"
)

func NewKubectlK8sActions(kubectlPath string) *KubectlK8sActions {
	return &KubectlK8sActions{
		cli:         &cli.Cli{},
		kubectlPath: kubectlPath,
	}
}

type KubectlK8sActions struct {
	cli         cli.CmdExecutor
	kubectlPath string
}

func (c *KubectlK8sActions) HostExecute(hostString string, args ...string) (out string, err error) {
	tokens := strings.Split(hostString, ":")
	namespace := tokens[0]
	podName := tokens[1]
	kubectlArgs := []string{c.kubectlPath, "exec", "-it", "-n", namespace, podName}
	kubectlArgs = append(kubectlArgs, args...)
	return c.cli.Execute(kubectlArgs...)
}

func (c *KubectlK8sActions) CopyFromHost(hostString, source, destination string) (out string, err error) {
	tokens := strings.Split(hostString, ":")
	namespace := tokens[0]
	podName := tokens[1]
	return c.cli.Execute(c.kubectlPath, "cp", "-n", namespace, fmt.Sprintf("%v:%v", podName, source), destination)
}

func (c *KubectlK8sActions) FindHosts(searchTerm string) (podName []string, err error) {
	tokens := strings.Split(searchTerm, ":")
	namespace := tokens[0]
	labelName := tokens[1]
	out, err := c.cli.Execute(c.kubectlPath, "get", "-n", namespace, "-l", labelName, "-o", "name")
	if err != nil {
		return []string{}, err
	}
	rawPods := strings.Split(out, "\n")
	var pods []string
	for _, pod := range rawPods {
		if pod == "" {
			continue
		}
		pods = append(pods, strings.TrimSpace(pod))
	}
	return pods, nil
}
