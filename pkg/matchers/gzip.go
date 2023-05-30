package matchers

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
)

// ContainFileInGzip checks if a file exists within a gzip archive.
func ContainFileInGzip(expectedFile string) types.GomegaMatcher {
	return &containFileInGzipMatcher{
		expectedFile: expectedFile,
	}
}

type containFileInGzipMatcher struct {
	expectedFile string
}

func (matcher *containFileInGzipMatcher) Match(actual interface{}) (success bool, err error) {
	gzipFile, ok := actual.(string)
	if !ok {
		return false, fmt.Errorf("ContainFileInGzip matcher expects a string (gzip file path) as actual, but got %T", actual)
	}

	// Open the gzip file
	file, err := os.Open(gzipFile)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// Create a gzip reader
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return false, err
	}
	defer gzipReader.Close()

	// Create a tar reader
	tarReader := tar.NewReader(gzipReader)

	// Iterate over each file in the gzip archive
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // Reached the end of the archive
		} else if err != nil {
			return false, err
		}

		// Get the file name
		fileName := header.Name

		// Check if the file name matches the expected file
		if strings.Contains(fileName, matcher.expectedFile) {
			return true, nil
		}
	}

	return false, nil
}

func (matcher *containFileInGzipMatcher) FailureMessage(actual interface{}) (message string) {
	return format.Message(actual, fmt.Sprintf("to contain file %s", matcher.expectedFile))
}

func (matcher *containFileInGzipMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return format.Message(actual, fmt.Sprintf("not to contain file %s", matcher.expectedFile))
}
