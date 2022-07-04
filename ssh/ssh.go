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

//ssh package provides functions for collections of logs via scp and ssh
package ssh

import (
	"fmt"
	"strings"

	"github.com/rsvihladremio/dremio-diagnostic-collector/cli"
)

//CmdSSHActions depends on the scp and ssh programs being present and
// then assumes ssh public key auth is in place since it has no support for using
// password based authentication
type CmdSSHActions struct {
	cli cli.CmdExecutor
}

func (c *CmdSSHActions) CopyFromHost(hostName, source, destination string) (string, error) {
	return c.cli.Execute("scp", fmt.Sprintf("%v:%v", hostName, source), destination)
}

func (c *CmdSSHActions) HostExecute(hostName string, arg string) (string, error) {
	sshArgs := []string{"ssh", "-c"}
	sshArgs = append(sshArgs, arg)
	sshArgs = append(sshArgs, hostName)
	return c.cli.Execute(sshArgs...)
}

func (c *CmdSSHActions) FindHosts(searchTerm string) (hosts []string, err error) {
	rawHosts := strings.Split(searchTerm, ",")
	for _, host := range rawHosts {
		if host == "" {
			continue
		}
		hosts = append(hosts, strings.TrimSpace(host))
	}
	return hosts, nil
}
