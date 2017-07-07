/*
Copyright 2017 Google Inc.

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

package mysqlctl

import (
	"reflect"
	"testing"

	"github.com/youtube/vitess/go/mysql"
)

func TestMariadbMakeBinlogEvent(t *testing.T) {
	input := []byte{1, 2, 3}
	want := mysql.NewMariadbBinlogEvent([]byte{1, 2, 3})
	if got := (&mariaDB10{}).MakeBinlogEvent(input); !reflect.DeepEqual(got, want) {
		t.Errorf("(&mariaDB10{}).MakeBinlogEvent(%#v) = %#v, want %#v", input, got, want)
	}
}

func TestMariadbParseGTID(t *testing.T) {
	input := "12-34-5678"
	want := mysql.MariadbGTID{Domain: 12, Server: 34, Sequence: 5678}

	got, err := (&mariaDB10{}).ParseGTID(input)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("(&mariaDB10{}).ParseGTID(%#v) = %#v, want %#v", input, got, want)
	}
}

func TestMariadbVersionMatch(t *testing.T) {
	table := map[string]bool{
		"10.0.13-MariaDB-1~precise-log": true,
		"5.1.63-google-log":             false,
	}
	for input, want := range table {
		if got := (&mariaDB10{}).VersionMatch(input); got != want {
			t.Errorf("(&mariaDB10{}).VersionMatch(%#v) = %v, want %v", input, got, want)
		}
	}
}
