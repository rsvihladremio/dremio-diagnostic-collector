package matchers_test

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/rsvihladremio/dremio-diagnostic-collector/pkg/matchers"
)

var _ = Describe("Gzip Matchers", func() {
	Context("ContainFileInGzip", func() {
		It("should contain the expected file", func() {
			gzipFile := filepath.Join("testdata", "file1.txt.gz")
			expectedFile := filepath.Join("testdata", "file1.txt")
			// Expect the gzip file to contain the expected file

			Expect(gzipFile).To(ContainThisFileInTheGzip(expectedFile))
		})

		It("should not contain a different file", func() {
			gzipFile := filepath.Join("testdata", "file1.txt.gz")
			expectedFile := filepath.Join("testdata", "file3.txt")

			// Expect the gzip file not to contain the different file
			Expect(gzipFile).NotTo(ContainThisFileInTheGzip(expectedFile))
		})
	})
})
