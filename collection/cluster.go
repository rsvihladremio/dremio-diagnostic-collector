//	Copyright 2023 Dremio Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This module deals with specific k8s cluster level data collection

package collection

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/rsvihladremio/dremio-diagnostic-collector/cli"
	"github.com/rsvihladremio/dremio-diagnostic-collector/cmd/simplelog"
	"github.com/rsvihladremio/dremio-diagnostic-collector/helpers"
	"github.com/rsvihladremio/dremio-diagnostic-collector/pkg/masking"
)

func ClusterK8sExecute(namespace string, cs CopyStrategy, ddfs helpers.Filesystem, c Collector, k string) error {
	cmds := []string{"nodes", "sc", "pvc", "pv", "service", "endpoints", "pods", "deployments", "statefulsets", "daemonset", "replicaset", "cronjob", "job", "events", "ingress", "limitrange", "resourcequota", "hpa", "pdb", "pc"}
	for _, cmd := range cmds {
		out, err := clusterExecute(namespace, cmd, c, k)
		if err != nil {
			simplelog.Errorf("when getting cluster config, error was %v", err)
			continue
		}
		text, err := masking.RemoveSecretsFromK8sJSON(string(out))
		if err != nil {
			simplelog.Errorf("unable to mask secrets for %v in namespace %v returning am empty text due to error '%v'", k, namespace, err)
			continue
		}
		p, err := cs.CreatePath("kubernetes", "dremio-master", "")
		if err != nil {
			simplelog.Errorf("trying to construct cluster config path %v with error %v", p, err)
			continue
		}
		path := strings.TrimSuffix(p, "dremio-master")
		filename := filepath.Join(path, cmd+".json")
		err = ddfs.WriteFile(filename, []byte(text), DirPerms)
		if err != nil {
			simplelog.Errorf("trying to write file %v, error was %v", filename, err)
			continue
		}

	}
	return nil
}

// Execute commands at the cluster level
// Calls a raw execute function and simply writes out the byte array read from the response
// that comes in directly from kubectl
func clusterExecute(namespace, cmd string, _ Collector, k string) ([]byte, error) {
	cli := &cli.Cli{}
	kubectlArgs := []string{k, "-n", namespace, "get"}
	kubectlArgs = append(kubectlArgs, cmd, "-o", "json")
	res, err := cli.ExecuteBytes(kubectlArgs...)
	if err != nil {
		return []byte(""), fmt.Errorf("when getting config %v error returned was %v", cmd, err)
	}
	return res, nil
}
