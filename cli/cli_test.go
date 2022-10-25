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
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCli(t *testing.T) {

	c := Cli{}
	var err error
	var out string
	var expectedOut string
	if runtime.GOOS == "windows" {
		out, err = c.Execute("cmd.exe", "/c", "dir", "/B", filepath.Join("testdata", "ls"))
		expectedOut = "file1\r\nfile2\r\n"
	} else {
		out, err = c.Execute("ls", "-a", filepath.Join("testdata", "ls"))
		expectedOut = "file1\nfile2\n"
	}
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	// have to use contains because we are getting some extra output
	if !strings.Contains(out, expectedOut) {
		t.Errorf("expected %q but was %q", expectedOut, out)
	}
}

func TestCliWithNoArgsForTheCommand(t *testing.T) {

	c := Cli{}
	var err error
	var out string
	var expectedOut string
	if runtime.GOOS == "windows" {
		out, err = c.Execute("cmd.exe")
		expectedOut = "Microsoft"
	} else {
		out, err = c.Execute("ls")
		expectedOut = "cli.go"
	}
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	// have to use contains because we are getting some extra output
	if !strings.Contains(out, expectedOut) {
		t.Errorf("expected %q but was %q", expectedOut, out)
	}
}

func TestCliWithBadCommand(t *testing.T) {

	c := Cli{}
	_, err := c.Execute("22JIDJMJMHHF")
	if err == nil {
		t.Error("expected error")
	}
	switch v := err.(type) {
	case UnableToStartErr:
		t.Log("expected error is correct")
	default:
		t.Errorf("unexpected error type %T but expected ExecuteCliErr", v)
	}
	expectedErr := "unable to start command '22JIDJMJMHHF' due to error"
	if !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("expected error to contain %v but was %v", expectedErr, err)
	}
}
