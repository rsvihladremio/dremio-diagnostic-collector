package threading_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestThreading(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Threading Suite")
}
