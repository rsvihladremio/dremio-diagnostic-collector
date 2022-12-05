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

// This module deals with specific k8s cluster level data collection

package collection

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/rsvihladremio/dremio-diagnostic-collector/cli"
	"github.com/rsvihladremio/dremio-diagnostic-collector/helpers"
)

func ClusterK8sExecute(cs CopyStrategy, ddfs helpers.Filesystem, c Collector, k string) error {
	cmds := []string{"nodes", "sc", "pvc", "pv", "service", "endpoints", "pods", "deployments", "statefulsets", "daemonset", "replicaset", "cronjob", "job", "events", "ingress", "limitrange", "resourcequota", "hpa", "pdb", "pc"}
	for _, cmd := range cmds {
		out, err := clusterExecute("default", cmd, c, k)
		if err != nil {
			return fmt.Errorf("ERROR: when getting cluster config, error was %v", err)
		}
		p, err := cs.CreatePath("kubernetes", "dremio-master", "")
		if err != nil {
			return fmt.Errorf("ERROR: trying to construct cluster config path %v", err)
		}
		path := strings.TrimSuffix(p, "dremio-master")
		filename := filepath.Join(path, cmd+".json")
		err = ddfs.WriteFile(filename, out, DirPerms)
		if err != nil {
			return fmt.Errorf("ERROR: trying to write file %v, error was %v", filename, err)
		}

	}
	return nil
}

// Execute commands at the cluster level
// Callsa raw execute function and simply writes out the byte array read from the response
// that comes in directly from kubectl
func clusterExecute(namespace, cmd string, c Collector, k string) ([]byte, error) {
	cli := &cli.Cli{}
	kubectlArgs := []string{k, "-n", namespace, "get"}
	kubectlArgs = append(kubectlArgs, cmd, "-o", "json")
	res, err := cli.Execute2(kubectlArgs...)
	if err != nil {
		log.Printf("ERROR: when getting config %v error returned was %v", cmd, err)
	}
	return res, nil
}

/*
func convertToJSON(res []byte) ([]byte, error) {
	// Clean up returned string
	//trim1 := strings.ReplaceAll(res, `\"`, `"`)
	//trim2 := strings.ReplaceAll(trim1, `\n`, "\n")
	os.WriteFile("/Users/mc/Support/deleteme/test.json", res, 0644)
	b, err := json.MarshalIndent(res, "", "\t")
	if err != nil {
		return nil, err
	}
	return b, nil
}
*/
