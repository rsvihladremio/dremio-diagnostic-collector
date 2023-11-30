package jps

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"

	"github.com/dremio/dremio-diagnostic-collector/cmd/local/ddcio"
)

func CaptureFlagsFromPID(pid int) (string, error) {
	var buf bytes.Buffer
	if err := ddcio.Shell(&buf, "jps -v"); err != nil {
		return "", fmt.Errorf("failed getting flags: '%w', output was: '%v'", err, buf.String())
	}
	scanner := bufio.NewScanner(&buf)
	//adjust the max line size capacity as the jpv output can be large
	const maxCapacity = 512 * 1024
	lineBuffer := make([]byte, maxCapacity)
	scanner.Buffer(lineBuffer, maxCapacity)
	jvmFlagsForPid := ""
	for scanner.Scan() {
		line := scanner.Text()
		pidPrefix := fmt.Sprintf("%v ", pid)
		if strings.HasPrefix(line, pidPrefix) {
			//matched now let's eliminate the pid part
			flagText := strings.TrimPrefix(line, pidPrefix)
			jvmFlagsForPid = strings.TrimSpace(flagText)
		}
	}
	if strings.TrimSpace(jvmFlagsForPid) == "" {
		return "", fmt.Errorf("pid %v not found in jps output", pid)
	}
	return jvmFlagsForPid, nil
}
