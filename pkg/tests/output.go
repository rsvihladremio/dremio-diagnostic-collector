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

// package tests provides helper functions and mocks for running tests
package tests

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/dremio/dremio-diagnostic-collector/cmd/simplelog"
)

// CaptureOutput captures standard output for a function f and returns the output string
func CaptureOutput(f func()) (string, error) {
	// Keep the original stdout and stderr.
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}

	os.Stdout = w
	os.Stderr = w

	outC := make(chan string)

	go func() {
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, r); err != nil {
			panic(err)
		}
		outC <- buf.String()
	}()

	f()

	if err := w.Close(); err != nil {
		return "", fmt.Errorf("unable to capture output due to error %v", err)
	}

	out := <-outC

	return out, nil
}

// Tree prints the directory structure in a tree-like format.
func Tree(dir string) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		simplelog.Errorf("Failed to get absolute path: %v with error %v", dir, err)
		return
	}

	fmt.Println(dir)
	PrintTree(dir, "")
}

func TreeToString(dir string) string {
	out, err := CaptureOutput(func() {
		Tree(dir)
	})
	if err != nil {
		simplelog.Errorf("unable to capture output for tree due to error %v", err)
		return ""
	}
	return out
}

// PrintTree is a helper function to print directory structure recursively.
func PrintTree(dir string, indent string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		simplelog.Errorf("Failed to read directory: %v with error %v", dir, err)
		return
	}

	for _, entry := range entries {
		fmt.Println(indent+"|--", entry.Name())
		if entry.IsDir() {
			PrintTree(filepath.Join(dir, entry.Name()), indent+"|   ")
		}
	}
}
