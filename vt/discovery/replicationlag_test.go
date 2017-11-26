/*
Copyright 2017 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreedto in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package discovery

import (
	"testing"

	querypb "github.com/youtube/vitess/go/vt/proto/query"
	"github.com/youtube/vitess/go/vt/topo"
)

// testSetMinNumTablets is a test helper function, if this is used by a production code path, something is wrong.
func testSetMinNumTablets(newMin int) {
	*minNumTablets = newMin
}

func TestFilterByReplicationLag(t *testing.T) {
	// 0 tablet
	got := FilterByReplicationLag([]*TabletStats{})
	if len(got) != 0 {
		t.Errorf("FilterByReplicationLag([]) = %+v, want []", got)
	}
	// 1 serving tablet
	ts1 := &TabletStats{
		Tablet:  topo.NewTablet(1, "cell", "host1"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{},
	}
	ts2 := &TabletStats{
		Tablet:  topo.NewTablet(2, "cell", "host2"),
		Serving: false,
		Stats:   &querypb.RealtimeStats{},
	}
	got = FilterByReplicationLag([]*TabletStats{ts1, ts2})
	if len(got) != 1 {
		t.Errorf("len(FilterByReplicationLag([{Tablet: {Uid: 1}, Serving: true}, {Tablet: {Uid: 2}, Serving: false}])) = %v, want 1", len(got))
	}
	if len(got) > 0 && !got[0].DeepEqual(ts1) {
		t.Errorf("FilterByReplicationLag([{Tablet: {Uid: 1}, Serving: true}, {Tablet: {Uid: 2}, Serving: false}]) = %+v, want %+v", got[0], ts1)
	}
	// lags of (1s, 1s, 1s, 30s)
	ts1 = &TabletStats{
		Tablet:  topo.NewTablet(1, "cell", "host1"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 1},
	}
	ts2 = &TabletStats{
		Tablet:  topo.NewTablet(2, "cell", "host2"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 1},
	}
	ts3 := &TabletStats{
		Tablet:  topo.NewTablet(3, "cell", "host3"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 1},
	}
	ts4 := &TabletStats{
		Tablet:  topo.NewTablet(4, "cell", "host4"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 30},
	}
	got = FilterByReplicationLag([]*TabletStats{ts1, ts2, ts3, ts4})
	if len(got) != 4 || !got[0].DeepEqual(ts1) || !got[1].DeepEqual(ts2) || !got[2].DeepEqual(ts3) || !got[3].DeepEqual(ts4) {
		t.Errorf("FilterByReplicationLag([1s, 1s, 1s, 30s]) = %+v, want all", got)
	}
	// lags of (5s, 10s, 15s, 120s)
	ts1 = &TabletStats{
		Tablet:  topo.NewTablet(1, "cell", "host1"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 5},
	}
	ts2 = &TabletStats{
		Tablet:  topo.NewTablet(2, "cell", "host2"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 10},
	}
	ts3 = &TabletStats{
		Tablet:  topo.NewTablet(3, "cell", "host3"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 15},
	}
	ts4 = &TabletStats{
		Tablet:  topo.NewTablet(4, "cell", "host4"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 120},
	}
	got = FilterByReplicationLag([]*TabletStats{ts1, ts2, ts3, ts4})
	if len(got) != 3 || !got[0].DeepEqual(got[0]) || !got[1].DeepEqual(ts2) || !got[2].DeepEqual(ts3) {
		t.Errorf("FilterByReplicationLag([5s, 10s, 15s, 120s]) = %+v, want [5s, 10s, 15s]", got)
	}
	// lags of (30m, 35m, 40m, 45m)
	ts1 = &TabletStats{
		Tablet:  topo.NewTablet(1, "cell", "host1"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 30 * 60},
	}
	ts2 = &TabletStats{
		Tablet:  topo.NewTablet(2, "cell", "host2"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 35 * 60},
	}
	ts3 = &TabletStats{
		Tablet:  topo.NewTablet(3, "cell", "host3"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 40 * 60},
	}
	ts4 = &TabletStats{
		Tablet:  topo.NewTablet(4, "cell", "host4"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 45 * 60},
	}
	got = FilterByReplicationLag([]*TabletStats{ts1, ts2, ts3, ts4})
	if len(got) != 4 || !got[0].DeepEqual(ts1) || !got[1].DeepEqual(ts2) || !got[2].DeepEqual(ts3) || !got[3].DeepEqual(ts4) {
		t.Errorf("FilterByReplicationLag([30m, 35m, 40m, 45m]) = %+v, want all", got)
	}
	// lags of (1s, 1s, 1m, 40m, 40m) - not run filter the second time as first run removed two items.
	ts1 = &TabletStats{
		Tablet:  topo.NewTablet(1, "cell", "host1"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 1},
	}
	ts2 = &TabletStats{
		Tablet:  topo.NewTablet(2, "cell", "host2"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 1},
	}
	ts3 = &TabletStats{
		Tablet:  topo.NewTablet(3, "cell", "host3"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 1 * 60},
	}
	ts4 = &TabletStats{
		Tablet:  topo.NewTablet(4, "cell", "host4"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 40 * 60},
	}
	ts5 := &TabletStats{
		Tablet:  topo.NewTablet(5, "cell", "host5"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 40 * 60},
	}
	got = FilterByReplicationLag([]*TabletStats{ts1, ts2, ts3, ts4, ts5})
	if len(got) != 3 || !got[0].DeepEqual(ts1) || !got[1].DeepEqual(ts2) || !got[2].DeepEqual(ts3) {
		t.Errorf("FilterByReplicationLag([1s, 1s, 1m, 40m, 40m]) = %+v, want [1s, 1s, 1m]", got)
	}
	// lags of (1s, 1s, 10m, 40m) - run filter twice to remove two items
	ts1 = &TabletStats{
		Tablet:  topo.NewTablet(1, "cell", "host1"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 1},
	}
	ts2 = &TabletStats{
		Tablet:  topo.NewTablet(2, "cell", "host2"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 1},
	}
	ts3 = &TabletStats{
		Tablet:  topo.NewTablet(3, "cell", "host3"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 10 * 60},
	}
	ts4 = &TabletStats{
		Tablet:  topo.NewTablet(4, "cell", "host4"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 40 * 60},
	}
	got = FilterByReplicationLag([]*TabletStats{ts1, ts2, ts3, ts4})
	if len(got) != 2 || !got[0].DeepEqual(ts1) || !got[1].DeepEqual(ts2) {
		t.Errorf("FilterByReplicationLag([1s, 1s, 10m, 40m]) = %+v, want [1s, 1s]", got)
	}
	// lags of (1m, 100m) - return at least 2 items to avoid overloading if the 2nd one is not delayed too much.
	ts1 = &TabletStats{
		Tablet:  topo.NewTablet(1, "cell", "host1"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 1 * 60},
	}
	ts2 = &TabletStats{
		Tablet:  topo.NewTablet(2, "cell", "host2"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 100 * 60},
	}
	got = FilterByReplicationLag([]*TabletStats{ts1, ts2})
	if len(got) != 2 || !got[0].DeepEqual(ts1) || !got[1].DeepEqual(ts2) {
		t.Errorf("FilterByReplicationLag([1m, 100m]) = %+v, want all", got)
	}
	// lags of (1m, 3h) - return 1 if the 2nd one is delayed too much.
	ts1 = &TabletStats{
		Tablet:  topo.NewTablet(1, "cell", "host1"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 1 * 60},
	}
	ts2 = &TabletStats{
		Tablet:  topo.NewTablet(2, "cell", "host2"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 3 * 60 * 60},
	}
	got = FilterByReplicationLag([]*TabletStats{ts1, ts2})
	if len(got) != 1 || !got[0].DeepEqual(ts1) {
		t.Errorf("FilterByReplicationLag([1m, 3h]) = %+v, want [1m]", got)
	}
}

func TestFilterByReplicationLagThreeTabletMin(t *testing.T) {
	// Use at least 3 tablets if possible
	testSetMinNumTablets(3)
	// lags of (1s, 1s, 10m, 11m) - returns at least32 items where the slightly delayed ones that are returned are the 10m and 11m ones.
	ts1 := &TabletStats{
		Tablet:  topo.NewTablet(1, "cell", "host1"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 1},
	}
	ts2 := &TabletStats{
		Tablet:  topo.NewTablet(2, "cell", "host2"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 1},
	}
	ts3 := &TabletStats{
		Tablet:  topo.NewTablet(3, "cell", "host3"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 10 * 60},
	}
	ts4 := &TabletStats{
		Tablet:  topo.NewTablet(4, "cell", "host4"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 11 * 60},
	}
	got := FilterByReplicationLag([]*TabletStats{ts1, ts2, ts3, ts4})
	if len(got) != 3 || !got[0].DeepEqual(ts1) || !got[1].DeepEqual(ts2) || !got[2].DeepEqual(ts3) {
		t.Errorf("FilterByReplicationLag([1s, 1s, 10m, 11m]) = %+v, want [1s, 1s, 10m]", got)
	}
	// lags of (11m, 10m, 1s, 1s) - reordered tablets returns the same 3 items where the slightly delayed one that is returned is the 10m and 11m ones.
	ts1 = &TabletStats{
		Tablet:  topo.NewTablet(1, "cell", "host1"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 11 * 60},
	}
	ts2 = &TabletStats{
		Tablet:  topo.NewTablet(2, "cell", "host2"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 10 * 60},
	}
	ts3 = &TabletStats{
		Tablet:  topo.NewTablet(3, "cell", "host3"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 1},
	}
	ts4 = &TabletStats{
		Tablet:  topo.NewTablet(4, "cell", "host4"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 1},
	}
	got = FilterByReplicationLag([]*TabletStats{ts1, ts2, ts3, ts4})
	if len(got) != 3 || !got[0].DeepEqual(ts3) || !got[1].DeepEqual(ts4) || !got[2].DeepEqual(ts2) {
		t.Errorf("FilterByReplicationLag([1s, 1s, 10m, 11m]) = %+v, want [1s, 1s, 10m]", got)
	}
	// Reset to the default
	testSetMinNumTablets(2)
}

func TestFilterByReplicationLagOneTabletMin(t *testing.T) {
	// Use at least 1 tablets if possible
	testSetMinNumTablets(1)
	// lags of (1s, 100m) - return only healthy tablet if that is all that is available.
	ts1 := &TabletStats{
		Tablet:  topo.NewTablet(1, "cell", "host1"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 1},
	}
	ts2 := &TabletStats{
		Tablet:  topo.NewTablet(2, "cell", "host2"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 100 * 60},
	}
	got := FilterByReplicationLag([]*TabletStats{ts1, ts2})
	if len(got) != 1 || !got[0].DeepEqual(ts1) {
		t.Errorf("FilterByReplicationLag([1s, 100m]) = %+v, want [1s]", got)
	}
	// lags of (1m, 100m) - return only healthy tablet if that is all that is healthy enough.
	ts1 = &TabletStats{
		Tablet:  topo.NewTablet(1, "cell", "host1"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 1 * 60},
	}
	ts2 = &TabletStats{
		Tablet:  topo.NewTablet(2, "cell", "host2"),
		Serving: true,
		Stats:   &querypb.RealtimeStats{SecondsBehindMaster: 100 * 60},
	}
	got = FilterByReplicationLag([]*TabletStats{ts1, ts2})
	if len(got) != 1 || !got[0].DeepEqual(ts1) {
		t.Errorf("FilterByReplicationLag([1m, 100m]) = %+v, want [1m]", got)
	}
	// Reset to the default
	testSetMinNumTablets(2)
}

func TestTrivialStatsUpdate(t *testing.T) {
	// Note the healthy threshold is set to 30s.
	cases := []struct {
		o        uint32
		n        uint32
		expected bool
	}{
		// both are under 30s
		{o: 0, n: 1, expected: true},
		{o: 15, n: 20, expected: true},

		// one is under 30s, the other isn't
		{o: 2, n: 40, expected: false},
		{o: 40, n: 10, expected: false},

		// both are over 30s, but close enough
		{o: 100, n: 100, expected: true},
		{o: 100, n: 105, expected: true},
		{o: 105, n: 100, expected: true},

		// both are over 30s, but too far
		{o: 100, n: 120, expected: false},
		{o: 120, n: 100, expected: false},
	}

	for _, c := range cases {
		o := &TabletStats{
			Stats: &querypb.RealtimeStats{
				SecondsBehindMaster: c.o,
			},
		}
		n := &TabletStats{
			Stats: &querypb.RealtimeStats{
				SecondsBehindMaster: c.n,
			},
		}
		got := TrivialStatsUpdate(o, n)
		if got != c.expected {
			t.Errorf("TrivialStatsUpdate(%v, %v) = %v, expected %v", c.o, c.n, got, c.expected)
		}
	}
}
