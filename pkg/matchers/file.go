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
	"fmt"

	"github.com/dremio/dremio-diagnostic-collector/pkg/ddcio"
	"github.com/onsi/gomega/types"
)

// MatchFile checks if a file has the expected content by comparing its content with the provided file.
func MatchFile(expectedFile string) types.GomegaMatcher {
	return &matchFileMatcher{
		expectedFile: expectedFile,
	}
}

type matchFileMatcher struct {
	expectedFile string
}

func (matcher *matchFileMatcher) Match(actual interface{}) (success bool, err error) {
	filePath, ok := actual.(string)
	if !ok {
		return false, fmt.Errorf("MatchFile matcher expects a string (file path) as actual, but got %T", actual)
	}

	areSame, err := ddcio.CompareFiles(matcher.expectedFile, filePath)
	if err != nil {
		return false, err
	}

	return areSame, nil
}

func (matcher *matchFileMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected file %s to match %s", actual, matcher.expectedFile)
}

func (matcher *matchFileMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected file %s not to match %s", actual, matcher.expectedFile)
}
