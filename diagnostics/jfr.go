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

// diagnostics contains all the commands that run server diagnostics to find problems on the host
package diagnostics

import "fmt"

/*
Commands used to run a JFR typically
export DREMIO_PID=$(ps ax | grep dremio | grep -v grep | awk '{print $1}')
jcmd $DREMIO_PID VM.unlock_commercial_features
jcmd $DREMIO_PID JFR.start name="DR_JFR" settings=profile maxage=21600s filename=/tmp/coordinator.jfr dumponexit=true
*/

func JfrPid() []string {
	//return []string{"bash -c", "ps ax | grep dremio", "|", "grep", "-v", "grep", "|", "awk", "'{print", "$1}'"}
	return []string{"ps", "ax"}
}
func JfrEnable(pid string) []string {
	return []string{"jcmd", pid, "VM.unlock_commercial_features"}
}

func JfrEnableSudo(sudouser string, pid string) []string {
	return []string{"sudo", "-u", sudouser, "jcmd", pid, "VM.unlock_commercial_features"}
}

func JfrRun(pid string, duration int, jfrname string, jfrpath string) []string {
	return []string{"jcmd", pid, "JFR.start", "name=" + jfrname, "settings=profile", "maxage=" + fmt.Sprint(duration) + "s", "filename=" + jfrpath, "dumponexit=true"}
}

func JfrRunSudo(sudouser string, pid string, duration int, jfrname string, jfrpath string) []string {
	return []string{"sudo", "-u", sudouser, "jcmd", pid, "JFR.start", "name=" + jfrname, "settings=profile", "maxage=" + fmt.Sprint(duration) + "s", "filename=" + jfrpath, "dumponexit=true"}
}

func JfrCheck(pid string) []string {
	return []string{"jcmd", pid, "JFR.check"}
}

func JfrCheckSudo(sudouser string, pid string) []string {
	return []string{"sudo", "-u", sudouser, "jcmd", pid, "JFR.check"}
}
