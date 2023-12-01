package consoleprint_test

import (
	"strings"
	"testing"

	"github.com/dremio/dremio-diagnostic-collector/pkg/consoleprint"
	"github.com/dremio/dremio-diagnostic-collector/pkg/output"
)

func TestClearsScreen(t *testing.T) {
	out, err := output.CaptureOutput(func() {
		consoleprint.PrintState()
	})
	if err != nil {
		t.Fatal(err)
	}

	// relying on test mode detection
	if !strings.Contains(out, "CLEAR SCREEN") {
		t.Errorf("output %v did not contain 'CLEAR SCREEN'", out)
	}
}
