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

//package cli provides wrapper support for executing commands, this is so
// we can test the rest of the implementations quickly.
package cli

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

type CmdExecutor interface {
	Execute(args ...string) (out string, err error)
}

type UnableToStartErr struct {
	Err error
	Cmd string
}

func (u UnableToStartErr) Error() string {
	return fmt.Sprintf("unable to start command '%v' due to error '%v'", u.Cmd, u.Err)
}

type ExecuteCliErr struct {
	Err error
	Cmd string
}

func (u ExecuteCliErr) Error() string {
	return fmt.Sprintf("during execution the command '%v' failed due to error '%v'", u.Cmd, u.Err)
}

// Cli
type Cli struct {
}

func (c *Cli) Execute(args ...string) (string, error) {
	log.Printf("args %v", strings.Join(args, " "))
	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.Output()
	if err != nil {
		return string(output), UnableToStartErr{Err: err, Cmd: strings.Join(args, " ")}
	}
	return string(output), nil
}
