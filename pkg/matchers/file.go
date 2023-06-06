package matchers

import (
	"fmt"

	"github.com/onsi/gomega/types"
	"github.com/rsvihladremio/dremio-diagnostic-collector/pkg/ddcio"
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
