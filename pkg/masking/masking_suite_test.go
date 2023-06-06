package masking_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMasking(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Masking Suite")
}
