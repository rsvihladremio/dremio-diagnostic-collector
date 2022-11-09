/*
   Copyright 2022 Ryan SVIHLA

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

// diagnostics contains all the commands that run server diagnostics to find problems on the host
package diagnostics

import (
	"reflect"
	"testing"
)

func TestJFRPid(t *testing.T) {
	result := JfrPid()
	expected := []string{"ps", "ax"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %#v but was %#v", expected, result)
	}
}

func TestJFREnable(t *testing.T) {
	result := JfrEnable("1")
	expected := []string{"jcmd", "1", "VM.unlock_commercial_features"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %#v but was %#v", expected, result)
	}
}

func TestJFREnableSudo(t *testing.T) {
	result := JfrEnableSudo("dremio", "1")
	expected := []string{"sudo", "-u", "dremio", "jcmd", "1", "VM.unlock_commercial_features"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %#v but was %#v", expected, result)
	}
}

func TestJFRRun(t *testing.T) {
	result := JfrRun("1", 600, "dremio", "/opt/dremio/data")
	expected := []string{"jcmd", "1", "JFR.start", "name=dremio", "settings=profile", "maxage=600s", "filename=/opt/dremio/data", "dumponexit=true"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %#v but was %#v", expected, result)
	}
}

func TestJFRRunSudo(t *testing.T) {
	result := JfrRunSudo("dremio", "1", 600, "dremio", "/opt/dremio/data")
	expected := []string{"sudo", "-u", "dremio", "jcmd", "1", "JFR.start", "name=dremio", "settings=profile", "maxage=600s", "filename=/opt/dremio/data", "dumponexit=true"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %#v but was %#v", expected, result)
	}
}

func TestJFRCheck(t *testing.T) {
	result := JfrCheck("600")
	expected := []string{"jcmd", "600", "JFR.check"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %#v but was %#v", expected, result)
	}
}

func TestJFRCheckSudo(t *testing.T) {
	result := JfrCheckSudo("dremio", "600")
	expected := []string{"sudo", "-u", "dremio", "jcmd", "600", "JFR.check"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %#v but was %#v", expected, result)
	}
}
