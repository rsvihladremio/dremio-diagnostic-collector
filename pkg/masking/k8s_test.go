package masking_test

import (
	"bytes"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rsvihladremio/dremio-diagnostic-collector/pkg/masking"
)

var _ = Describe("K8s Masking", func() {

	Context("RemoveSecretsFromK8sJSON", func() {
		It("should mask secrets from k8s JSON", func() {
			input := `{
				"items": [
					{
						"kind": "pod",
						"metadata": {
							"annotations": {
								"kubectl.kubernetes.io/last-applied-configuration": "secret"
							}
						},
						"spec": {
							"containers": [
								{
									"env": [
										{
											"name": "password",
											"value": "secret"
										}
									]
								}
							]
						}
					}
				]
			}`
			expected := `{
				"items": [
					{
						"kind": "pod",
						"metadata": {
							"annotations": {
								"kubectl.kubernetes.io/last-applied-configuration": "REMOVED_POTENTIAL_SECRET"
							}
						},
						"spec": {
							"containers": [
								{
									"env": [
										{
											"name": "password",
											"value": "REMOVED_POTENTIAL_SECRET"
										}
									]
								}
							]
						}
					}
				]
			}`
			output, err := masking.RemoveSecretsFromK8sJSON(input)
			Expect(err).NotTo(HaveOccurred())
			Expect(jsonCompact(output)).To(Equal(jsonCompact(expected)))
		})

		It("should return error for unsupported kind", func() {
			input := `{
				"items": [
					{
						"kind": "unsupported",
						"metadata": {},
						"spec": {}
					}
				]
			}`

			_, err := masking.RemoveSecretsFromK8sJSON(input)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported kind"))
		})

		It("should handle invalid JSON input", func() {
			input := `{
				"items": "invalid"
			}`

			_, err := masking.RemoveSecretsFromK8sJSON(input)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("items must be an array but was 'string'"))
		})
	})
})

func jsonCompact(s string) string {
	buf := new(bytes.Buffer)
	json.Compact(buf, []byte(s))
	return buf.String()
}
