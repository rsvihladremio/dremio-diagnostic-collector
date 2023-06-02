package masking_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rsvihladremio/dremio-diagnostic-collector/cmd/simplelog"
	"github.com/rsvihladremio/dremio-diagnostic-collector/pkg/masking"
)

var _ = Describe("Config", func() {

	Describe("RemoveSecretsFromDremioConf", func() {
		It("should throw an error if the provided file is not a dremio.conf", func() {
			err := masking.RemoveSecretsFromDremioConf("testdata/myFile.txt")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected file with name 'dremio.conf', got "))
		})

	})

	Describe("RemoveSecretsFromDremioConf", func() {
		It("should mask secrets in the config file", func() {
			// We'll write the dremio.conf contents to a temporary file for this test
			conf := `
			paths: {
				local: ${DREMIO_HOME}"/data"
				dist: "pdfs://"${paths.local}"/pdfs"
			}
	
			services: {
				executor: {
					cache: {
						path.db: "/opt/dremio/cloudcache/c0"
						pctquota.db: 100
	
						path.fs: ["/opt/dremio/cloudcache/c0"]
						pctquota.fs: [100]
						ensurefreespace.fs: [0]
	
					}
				}
	
				javax.net.ssl {
					keyStore: "",
					keyStorePassword: "oh silly man",
					trustStore:"",
					trustStorePassword: "why do you do this"
				}
			}
			`
			tmpDir := os.TempDir()
			tmpfile := filepath.Join(tmpDir, "dremio.conf")
			defer func() {
				if err := os.RemoveAll(tmpDir); err != nil {
					simplelog.Warningf("cannot remove test dir %v due to error %v", tmpDir, err)
				}
			}()

			err := os.WriteFile(tmpfile, []byte(conf), 0700)
			Expect(err).NotTo(HaveOccurred())

			err = masking.RemoveSecretsFromDremioConf(tmpfile)
			Expect(err).NotTo(HaveOccurred())

			cleanedConf, err := os.ReadFile(tmpfile)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(cleanedConf)).To(ContainSubstring(`keyStorePassword: "<REMOVED_POTENTIAL_SECRET>"`))
			Expect(string(cleanedConf)).To(ContainSubstring(`trustStorePassword: "<REMOVED_POTENTIAL_SECRET>"`))
		})
	})

})
