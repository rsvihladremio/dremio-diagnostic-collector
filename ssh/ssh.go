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

// ssh package provides functions for collections of logs via scp and ssh
package ssh

import (
	"fmt"
	"strings"

	"github.com/rsvihladremio/dremio-diagnostic-collector/cli"
)

func NewCmdSSHActions(sshKey, sshUser string) *CmdSSHActions {
	return &CmdSSHActions{
		cli:     &cli.Cli{},
		sshKey:  sshKey,
		sshUser: sshUser,
	}
}

// CmdSSHActions depends on the scp and ssh programs being present and
// then assumes ssh public key auth is in place since it has no support for using
// password based authentication
type CmdSSHActions struct {
	cli     cli.CmdExecutor
	sshKey  string
	sshUser string
}

func (c *CmdSSHActions) CopyFromHost(hostName string, isCoordinator bool, source, destination string) (string, error) {
	return c.cli.Execute("scp", "-i", c.sshKey, "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf("%v@%v:%v", c.sshUser, hostName, source), destination)
}

func (c *CmdSSHActions) CopyFromHostSudo(hostName string, isCoordinator bool, sudoUser, source, destination string) (string, error) {
	return c.cli.Execute("scp", "-i", c.sshKey, "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no", fmt.Sprintf("%v@%v:%v", c.sshUser, hostName, source), destination)
}

func (c *CmdSSHActions) HostExecute(hostName string, isCoordinator bool, args ...string) (string, error) {
	sshArgs := []string{"ssh", "-i", c.sshKey, "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no"}
	sshArgs = append(sshArgs, fmt.Sprintf("%v@%v", c.sshUser, hostName))
	sshArgs = append(sshArgs, strings.Join(args, " "))
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
