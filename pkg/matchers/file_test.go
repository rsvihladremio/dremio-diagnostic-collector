package matchers_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/rsvihladremio/dremio-diagnostic-collector/pkg/matchers"
)

var _ = Describe("File Matchers", func() {
	Context("MatchFile", func() {
		It("should match files with the same content", func() {
			file1 := "testdata/file1.txt"
			file2 := "testdata/file2.txt"
			expectedFile := "testdata/expected.txt"

			// Expect file1 to match the expected file
			Expect(file1).To(MatchFile(expectedFile))

			// Expect file2 to match the expected file
			Expect(file2).To(MatchFile(expectedFile))
		})

		It("should not match files with different content", func() {
			file1 := "testdata/file1.txt"
			file3 := "testdata/file3.txt"
			expectedFile := "testdata/expected.txt"

			// Expect file1 to match the expected file
			Expect(file1).To(MatchFile(expectedFile))

			// Expect file3 not to match the expected file
			Expect(file3).NotTo(MatchFile(expectedFile))
		})
	})
})
