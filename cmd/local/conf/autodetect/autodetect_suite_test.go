package autodetect_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAutodetect(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Autodetect Suite")
}
