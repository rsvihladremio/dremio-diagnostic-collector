package threading_test

import (
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rsvihladremio/dremio-diagnostic-collector/pkg/threading"
)

var _ = Describe("Threading", func() {
	var (
		tp *threading.ThreadPool
	)

	BeforeEach(func() {
		tp = threading.NewThreadPool(10)
	})

	When("forget to call start", func() {
		var executed bool
		var waitErr error
		BeforeEach(func() {
			executed = false
			jobFunc := func() error {
				executed = true
				return nil
			}

			tp.AddJob(jobFunc)
			waitErr = tp.Wait()
		})

		It("should forget to execute", func() {
			Expect(executed).ToNot(BeTrue())
		})

		It("wait should error out", func() {
			Expect(waitErr).ToNot(BeNil())
		})
	})

	When("Wait with one job", func() {
		var waitErr error
		var executed bool
		BeforeEach(func() {
			executed = false
			jobFunc := func() error {
				executed = true
				return nil
			}

			tp.Start()
			tp.AddJob(jobFunc)
			waitErr = tp.Wait()
		})

		It("should execute all jobs", func() {
			Expect(executed).To(BeTrue())
		})

		It("should wait successfully", func() {
			Expect(waitErr).To(BeNil())
		})
	})

	When("Wait with no jobs", func() {
		It("should fail", func() {
			tp.Start()
			err := tp.Wait()
			Expect(err).ToNot(BeNil())
		})
	})

	When("there are a lot more jobs to add than there are threads", func() {
		var executed []bool
		var mut sync.RWMutex
		var waitErr error
		BeforeEach(func() {
			jobFunc := func() error {
				mut.Lock()
				defer mut.Unlock()
				executed = append(executed, true)
				return nil
			}
			for i := 0; i < 100; i++ {
				tp.AddJob(jobFunc)
			}
			tp.Start()
			waitErr = tp.Wait()
		})

		It("should execute all jobs", func() {
			Expect(executed).To(HaveLen(100))
		})

		It("should wait successfully", func() {
			Expect(waitErr).To(BeNil())
		})

	})

	When("Wait", func() {
		var executed []bool
		var mut sync.RWMutex
		var waitErr error
		BeforeEach(func() {
			jobFunc := func() error {
				mut.Lock()
				defer mut.Unlock()
				executed = append(executed, true)
				return nil
			}

			for i := 0; i < 10; i++ {
				tp.AddJob(jobFunc)
			}
			tp.Start()
			waitErr = tp.Wait()
		})

		It("should execute all jobs", func() {
			Expect(executed).To(HaveLen(10))
		})

		It("should wait successfully", func() {
			Expect(waitErr).To(BeNil())
		})
	})
})
