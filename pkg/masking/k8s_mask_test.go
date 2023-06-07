//	Copyright 2023 Dremio Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package masking_test

import (
	"bytes"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/dremio/dremio-diagnostic-collector/cmd/simplelog"
	"github.com/dremio/dremio-diagnostic-collector/pkg/masking"
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

		It("should no op for unsupported kind", func() {
			input := `{
				"items": [
					{
						"kind": "unsupported",
						"metadata": {},
						"spec": {}
					}
				]
			}`
			expected := `{
				"items": [
					{
						"kind": "unsupported",
						"metadata": {},
						"spec": {}
					}
				]
			}`
			output, err := masking.RemoveSecretsFromK8sJSON(input)
			Expect(err).ToNot(HaveOccurred())
			Expect(jsonCompact(output)).To(Equal(jsonCompact(expected)))

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
	if err := json.Compact(buf, []byte(s)); err != nil {
		simplelog.Errorf("json compact failed due to error %v", err)
	}
	return buf.String()
}
