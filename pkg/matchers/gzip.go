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

package matchers

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
)

func ContainThisFileInTheGzip(expectedFilePath string) types.GomegaMatcher {
	return &gzipFileMatcher{
		expectedFilePath: expectedFilePath,
	}
}

type gzipFileMatcher struct {
	expectedFilePath string
}

func (m *gzipFileMatcher) Match(actual interface{}) (success bool, err error) {
	actualFilePath, ok := actual.(string)
	if !ok {
		return false, fmt.Errorf("MatchGzipFileContents matcher expects a string (path to gzip file)")
	}

	gzipFile, err := os.Open(filepath.Clean(actualFilePath))
	if err != nil {
		return false, fmt.Errorf("failed to open gzip file: %v", err)
	}
	defer gzipFile.Close()

	gzipReader, err := gzip.NewReader(gzipFile)
	if err != nil {
		return false, fmt.Errorf("failed to create gzip reader: %v", err)
	}

	actualFileBytes, err := io.ReadAll(gzipReader)
	if err != nil {
		return false, fmt.Errorf("failed to read file from gzip archive: %v", err)
	}

	expectedFileBytes, err := os.ReadFile(filepath.Clean(m.expectedFilePath))
	if err != nil {
		return false, fmt.Errorf("failed to read expected file: %v", err)
	}

	return bytes.Equal(actualFileBytes, expectedFileBytes), nil
}

func (m *gzipFileMatcher) FailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "to match file contents of the file inside the gzip archive with", m.expectedFilePath)
}

func (m *gzipFileMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "not to match file contents of the file inside the gzip archive with", m.expectedFilePath)
}
