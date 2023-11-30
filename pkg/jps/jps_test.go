package jps_test

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/dremio/dremio-diagnostic-collector/pkg/jps"
)

func TestJvmFlagCapture(t *testing.T) {
	jarLoc := filepath.Join("testdata", "demo.jar")
	cmd := exec.Command("java", "-jar", "-Dmyflag=1", "-Xmx512M", jarLoc)
	if err := cmd.Start(); err != nil {
		t.Fatalf("cmd.Start() failed with %s\n", err)
	}

	defer func() {
		if err := cmd.Process.Kill(); err != nil {
			t.Fatalf("failed to kill process: %s", err)
		} else {
			t.Log("Process killed successfully.")
		}
	}()
	flags, err := jps.CaptureFlagsFromPID(cmd.Process.Pid)
	if err != nil {
		t.Fatalf("expected no error but got %v", err)
	}
	expected := "demo.jar -Dmyflag=1 -Xmx512M"
	if expected != flags {
		t.Errorf("expected %v to %v", flags, expected)
	}
}
