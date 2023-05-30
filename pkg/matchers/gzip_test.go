package matchers_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/rsvihladremio/dremio-diagnostic-collector/pkg/matchers"
)

var _ = Describe("Gzip Matchers", func() {
	Context("ContainFileInGzip", func() {
		It("should contain the expected file", func() {
			gzipFile := "testdata/archive.tar.gz"
			expectedFile := "file1.txt"

			// Expect the gzip file to contain the expected file
			Expect(gzipFile).To(ContainFileInGzip(expectedFile))
		})

		It("should not contain a different file", func() {
			gzipFile := "testdata/archive.tar.gz"
			expectedFile := "file3.txt"

			// Expect the gzip file not to contain the different file
			Expect(gzipFile).NotTo(ContainFileInGzip(expectedFile))
		})
	})
})
