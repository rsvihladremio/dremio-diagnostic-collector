package autodetect_test

import (
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rsvihladremio/dremio-diagnostic-collector/cmd/local/conf/autodetect"
)

var _ = Describe("AWSE", func() {
	Context("IsAWSEFromText", func() {
		It("should return false when AwsDremioDaemon is not found in the text", func() {
			jpsText := "12345 JavaProcess\n67890 AnotherProcess"
			isAWSE, err := autodetect.IsAWSEFromJPSOutput(jpsText)
			Expect(err).NotTo(HaveOccurred())
			Expect(isAWSE).To(BeFalse())
		})

		It("should return true when AwsDremioDaemon is found in the text", func() {
			jpsText := "12345 AwsDremioDaemon\n67890 AnotherProcess"
			isAWSE, err := autodetect.IsAWSEFromJPSOutput(jpsText)
			Expect(err).NotTo(HaveOccurred())
			Expect(isAWSE).To(BeTrue())
		})
	})

	Context("IsAWSEExecutorUsingDir", func() {
		var (
			testDir  string
			nodeName string
			err      error
		)

		BeforeEach(func() {
			testDir, err = ioutil.TempDir("", "example")
			Expect(err).NotTo(HaveOccurred())
			nodeName = "TestNode"

			subDir := testDir + "/SubDirectory"
			err = os.Mkdir(subDir, 0755)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			os.RemoveAll(testDir)
		})

		It("should return true when node name is found", func() {
			nodeDir := testDir + "/" + nodeName
			err := os.Mkdir(nodeDir, 0755)
			Expect(err).NotTo(HaveOccurred())

			isAWSE, err := autodetect.IsAWSEExecutorUsingDir(testDir, nodeName)
			Expect(err).NotTo(HaveOccurred())
			Expect(isAWSE).To(BeTrue())
		})

		It("should return false when node name is not found", func() {
			isAWSE, err := autodetect.IsAWSEExecutorUsingDir(testDir, nodeName)
			Expect(err).NotTo(HaveOccurred())
			Expect(isAWSE).To(BeFalse())
		})
	})

	Context("IsAWSEExecutorUsingDir", func() {
		var (
			testDir  string
			nodeName string
			err      error
		)

		BeforeEach(func() {
			testDir, err = ioutil.TempDir("", "example")
			Expect(err).NotTo(HaveOccurred())
			nodeName = "TestNode"

			subDir := testDir + "/SubDirectory"
			err = os.Mkdir(subDir, 0755)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			os.RemoveAll(testDir)
		})

		It("should return true when node name is found", func() {
			nodeDir := testDir + "/" + nodeName
			err := os.Mkdir(nodeDir, 0755)
			Expect(err).NotTo(HaveOccurred())

			isAWSE, err := autodetect.IsAWSEExecutorUsingDir(testDir, nodeName)
			Expect(err).NotTo(HaveOccurred())
			Expect(isAWSE).To(BeTrue())
		})

		It("should return false when node name is not found", func() {
			isAWSE, err := autodetect.IsAWSEExecutorUsingDir(testDir, nodeName)
			Expect(err).NotTo(HaveOccurred())
			Expect(isAWSE).To(BeFalse())
		})
	})
})
