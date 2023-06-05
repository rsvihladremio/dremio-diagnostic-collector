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

// package autodetect looks at the system configuration and file names and tries to guess at the correct configuration
package autodetect

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/rsvihladremio/dremio-diagnostic-collector/cmd/simplelog"
	"github.com/rsvihladremio/dremio-diagnostic-collector/pkg/ddcio"
)

func GetDremioPID() (int, error) {
	isAWSE, err := IsAWSE()
	if err != nil {
		return -1, fmt.Errorf("failed getting awse status %v", err)
	}
	var procName string
	if isAWSE {
		procName = "AwsDremioDaemon"
	} else {
		procName = "DremioDaemon"
	}
	var jpsOutput bytes.Buffer
	if err := ddcio.Shell(&jpsOutput, "jps"); err != nil {
		simplelog.Warningf("attempting to get full jps output failed: %v", err)
	}
	scanner := bufio.NewScanner(&jpsOutput)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, procName) {
			tokens := strings.Split(line, " ")
			if len(tokens) == 0 {
				return -1, fmt.Errorf("no pid for dremio found in text '%v'", line)
			}
			pidText := tokens[0]
			return strconv.Atoi(pidText)
		}
	}
	return -1, fmt.Errorf("found no matching process named %v therefore cannot get the pid", procName)
}
