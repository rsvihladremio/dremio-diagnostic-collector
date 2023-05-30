package ddcio_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rsvihladremio/dremio-diagnostic-collector/pkg/ddcio"
)

func TestCompareFiles(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CompareFiles Suite")
}

var _ = Describe("CompareFiles", func() {
	Context("when comparing files with the same content", func() {
		It("should return true", func() {
			file1 := "testdata/file1.txt"
			file2 := "testdata/file1_copy.txt"

			areSame, err := ddcio.CompareFiles(file1, file2)
			Expect(err).To(BeNil())
			Expect(areSame).To(BeTrue())
		})
	})

	Context("when comparing files with different content", func() {
		It("should return false", func() {
			file1 := "testdata/file1.txt"
			file2 := "testdata/file2.txt"

			areSame, err := ddcio.CompareFiles(file1, file2)
			Expect(err).To(BeNil())
			Expect(areSame).To(BeFalse())
		})
	})

	Context("when comparing non-existent files", func() {
		It("should return an error", func() {
			file1 := "testdata/nonexistent1.txt"
			file2 := "testdata/nonexistent2.txt"

			areSame, err := ddcio.CompareFiles(file1, file2)
			Expect(err).NotTo(BeNil())
			Expect(areSame).To(BeFalse())
		})
	})
})
