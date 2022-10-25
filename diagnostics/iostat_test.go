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

func TestIoStatArgs(t *testing.T) {
	result := IOStatArgs(10)
	expected := []string{"iostat", "-y", "-x", "-d", "-c", "-t", "1", "10"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %#v but was %#v", expected, result)
	}
}
