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

//cmd package contains all the command line flag and initialization logic for commands
package cmd

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSSHDefault(t *testing.T) {
	sshPath, err := sshDefault()
	if err != nil {
		t.Fatalf("unexpected exception %v", err)
	}

	expectedPath := filepath.Join(".ssh", "id_rsa")
	if !strings.HasSuffix(sshPath, expectedPath) {
		t.Errorf("expected %v but was %v", expectedPath, sshPath)
	}
}
