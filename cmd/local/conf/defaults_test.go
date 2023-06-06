package conf_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rsvihladremio/dremio-diagnostic-collector/cmd/local/conf"
	"github.com/spf13/viper"
)

var _ = Describe("SetViperDefaults", func() {
	var (
		defaultThreads        int
		hostName              string
		defaultCaptureSeconds int
		outputDir             string
	)

	BeforeEach(func() {
		// Set up some default values for the inputs.
		defaultThreads = 10
		hostName = "test-host"
		defaultCaptureSeconds = 30
		outputDir = "/tmp"

		viper.Reset()
		// Run the function.
		conf.SetViperDefaults(defaultThreads, hostName, defaultCaptureSeconds, outputDir)
	})

	It("should set the correct default values", func() {
		Expect(viper.Get(conf.KeyCollectAccelerationLog)).To(BeFalse())
		Expect(viper.Get(conf.KeyCollectAccessLog)).To(BeFalse())
		Expect(viper.Get(conf.KeyDremioLogDir)).To(Equal("/var/log/dremio"))
		Expect(viper.Get(conf.KeyNumberThreads)).To(Equal(defaultThreads))
		Expect(viper.Get(conf.KeyDremioUsername)).To(Equal("dremio"))
		Expect(viper.Get(conf.KeyDremioPatToken)).To(Equal(""))
		Expect(viper.Get(conf.KeyDremioConfDir)).To(Equal("/opt/dremio/conf"))
		Expect(viper.Get(conf.KeyDremioRocksdbDir)).To(Equal("/opt/dremio/data/db"))
		Expect(viper.Get(conf.KeyCollectDremioConfiguration)).To(BeTrue())
		Expect(viper.Get(conf.KeyCaptureHeapDump)).To(BeFalse())
		Expect(viper.Get(conf.KeyNumberJobProfiles)).To(Equal(25000))
		Expect(viper.Get(conf.KeyDremioEndpoint)).To(Equal("http://localhost:9047"))
		Expect(viper.Get(conf.KeyTmpOutputDir)).To(Equal(outputDir))
		Expect(viper.Get(conf.KeyCollectMetrics)).To(BeTrue())
		Expect(viper.Get(conf.KeyCollectDiskUsage)).To(BeTrue())
		Expect(viper.Get(conf.KeyDremioLogsNumDays)).To(Equal(7))
		Expect(viper.Get(conf.KeyDremioQueriesJSONNumDays)).To(Equal(28))
		Expect(viper.Get(conf.KeyDremioGCFilePattern)).To(Equal("gc*.log*"))
		Expect(viper.Get(conf.KeyCollectQueriesJSON)).To(BeTrue())
		Expect(viper.Get(conf.KeyCollectServerLogs)).To(BeTrue())
		Expect(viper.Get(conf.KeyCollectMetaRefreshLog)).To(BeTrue())
		Expect(viper.Get(conf.KeyCollectReflectionLog)).To(BeTrue())
		Expect(viper.Get(conf.KeyCollectGCLogs)).To(BeTrue())
		Expect(viper.Get(conf.KeyCollectJFR)).To(BeTrue())
		Expect(viper.Get(conf.KeyCollectJStack)).To(BeTrue())
		Expect(viper.Get(conf.KeyCollectSystemTablesExport)).To(BeTrue())
		Expect(viper.Get(conf.KeyCollectWLM)).To(BeTrue())
		Expect(viper.Get(conf.KeyCollectKVStoreReport)).To(BeTrue())
		Expect(viper.Get(conf.KeyDremioJStackTimeSeconds)).To(Equal(defaultCaptureSeconds))
		Expect(viper.Get(conf.KeyDremioJFRTimeSeconds)).To(Equal(defaultCaptureSeconds))
		Expect(viper.Get(conf.KeyNodeMetricsCollectDurationSeconds)).To(Equal(defaultCaptureSeconds))
		Expect(viper.Get(conf.KeyDremioJStackFreqSeconds)).To(Equal(1))
		Expect(viper.Get(conf.KeyDremioGCLogsDir)).To(Equal(""))
		Expect(viper.Get(conf.KeyNodeName)).To(Equal(hostName))
		Expect(viper.Get(conf.KeyAcceptCollectionConsent)).To(BeTrue())
		Expect(viper.Get(conf.KeyAllowInsecureSSL)).To(BeFalse())
	})
})
