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
package kubernetes

import (
	"fmt"
	"strings"

	"github.com/rsvihladremio/dremio-diagnostic-collector/cli"
)

type K8sActions interface {
	PodExecute(podName, namespace string, args ...string) (out string, err error)
	PodCopyFromFile(podName, namespace, source, destination string) (out string, err error)
	PodSearch(labelName, namespace string) (podName []string, err error)
}

func NewKubectlK8sActions() *KubectlK8sActions {
	return &KubectlK8sActions{
		cli: &cli.Cli{},
	}
}

type KubectlK8sActions struct {
	cli cli.CmdExecutor
}

func (c *KubectlK8sActions) PodExecute(podName, namespace string, args ...string) (out string, err error) {
	kubectlArgs := []string{"kubectl", "exec", "-it", "-n", namespace, podName}
	kubectlArgs = append(kubectlArgs, args...)
	return c.cli.Execute(kubectlArgs...)
}

func (c *KubectlK8sActions) PodCopyFromFile(podName, namespace, source, destination string) (out string, err error) {
	return c.cli.Execute("kubectl", "cp", "-n", namespace, fmt.Sprintf("%v:%v", podName, source), destination)
}

func (c *KubectlK8sActions) PodSearch(labelName, namespace string) (podName []string, err error) {
	out, err := c.cli.Execute("kubectl", "get", "-n", namespace, "-l", labelName, "-o", "name")
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
