package autodetect_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rsvihladremio/dremio-diagnostic-collector/cmd/local/conf/autodetect"
)

var _ = Describe("DremioPID", func() {
	Context("GetDremioPIDFromText", func() {
		It("should return an error when no matching process name is found", func() {
			jpsOutput := "12345 JavaProcess\n67890 AnotherProcess"
			isAWSE := false
			pid, err := autodetect.GetDremioPIDFromText(jpsOutput, isAWSE)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("found no matching process named DremioDaemon in text 12345 JavaProcess, 67890 AnotherProcess therefore cannot get the pid"))
			Expect(pid).To(Equal(-1))
		})

		It("should return PID when a matching process name is found", func() {
			jpsOutput := "12345 DremioDaemon\n67890 AnotherProcess"
			isAWSE := false
			pid, err := autodetect.GetDremioPIDFromText(jpsOutput, isAWSE)
			Expect(err).NotTo(HaveOccurred())
			Expect(pid).To(Equal(12345))
		})
	})

})
