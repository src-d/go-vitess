// Copyright 2017, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tabletserver

import (
	"reflect"
	"testing"
)

func TestMessagerCacheOrder(t *testing.T) {
	mc := NewMessagerCache(10)
	if !mc.Add(&MessageRow{
		TimeNext: 1,
		Epoch:    0,
		id:       "row01",
	}) {
		t.Fatal("Add returned false")
	}
	if !mc.Add(&MessageRow{
		TimeNext: 2,
		Epoch:    0,
		id:       "row02",
	}) {
		t.Fatal("Add returned false")
	}
	if !mc.Add(&MessageRow{
		TimeNext: 2,
		Epoch:    1,
		id:       "row12",
	}) {
		t.Fatal("Add returned false")
	}
	if !mc.Add(&MessageRow{
		TimeNext: 1,
		Epoch:    1,
		id:       "row11",
	}) {
		t.Fatal("Add returned false")
	}
	if !mc.Add(&MessageRow{
		TimeNext: 3,
		Epoch:    0,
		id:       "row03",
	}) {
		t.Fatal("Add returned false")
	}
	var rows []string
	for i := 0; i < 5; i++ {
		rows = append(rows, mc.Pop().id)
	}
	want := []string{
		"row03",
		"row02",
		"row01",
		"row12",
		"row11",
	}
	if !reflect.DeepEqual(rows, want) {
		t.Errorf("Pop order: %+v, want %+v", rows, want)
	}
}

func TestMessagerCacheDupKey(t *testing.T) {
	mc := NewMessagerCache(10)
	if !mc.Add(&MessageRow{
		TimeNext: 1,
		Epoch:    0,
		id:       "row01",
	}) {
		t.Fatal("Add returned false")
	}
	if !mc.Add(&MessageRow{
		TimeNext: 1,
		Epoch:    0,
		id:       "row01",
	}) {
		t.Error("Add(dup): returned false, want true")
	}
	_ = mc.Pop()
	if !mc.Add(&MessageRow{
		TimeNext: 1,
		Epoch:    0,
		id:       "row01",
	}) {
		t.Error("Add(dup): returned false, want true")
	}
	mc.Discard("row01")
	if !mc.Add(&MessageRow{
		TimeNext: 1,
		Epoch:    0,
		id:       "row01",
	}) {
		t.Fatal("Add returned false")
	}
}

func TestMessagerCacheDiscard(t *testing.T) {
	mc := NewMessagerCache(10)
	if !mc.Add(&MessageRow{
		TimeNext: 1,
		Epoch:    0,
		id:       "row01",
	}) {
		t.Fatal("Add returned false")
	}
	mc.Discard("row01")
	if row := mc.Pop(); row != nil {
		t.Errorf("Pop: want nil, got %s", row.id)
	}
	if !mc.Add(&MessageRow{
		TimeNext: 1,
		Epoch:    0,
		id:       "row01",
	}) {
		t.Fatal("Add returned false")
	}
	if row := mc.Pop(); row == nil || row.id != "row01" {
		t.Errorf("Pop: want row01, got %v", row)
	}

	// Add will be a no-op.
	if !mc.Add(&MessageRow{
		TimeNext: 1,
		Epoch:    0,
		id:       "row01",
	}) {
		t.Fatal("Add returned false")
	}
	if row := mc.Pop(); row != nil {
		t.Errorf("Pop: want nil, got %s", row.id)
	}
	mc.Discard("row01")

	// Now we can add.
	if !mc.Add(&MessageRow{
		TimeNext: 1,
		Epoch:    0,
		id:       "row01",
	}) {
		t.Fatal("Add returned false")
	}
	if row := mc.Pop(); row == nil || row.id != "row01" {
		t.Errorf("Pop: want row01, got %v", row)
	}
}

func TestMessagerCacheFull(t *testing.T) {
	mc := NewMessagerCache(2)
	if !mc.Add(&MessageRow{
		TimeNext: 1,
		Epoch:    0,
		id:       "row01",
	}) {
		t.Fatal("Add returned false")
	}
	if !mc.Add(&MessageRow{
		TimeNext: 2,
		Epoch:    0,
		id:       "row02",
	}) {
		t.Fatal("Add returned false")
	}
	if mc.Add(&MessageRow{
		TimeNext: 2,
		Epoch:    1,
		id:       "row12",
	}) {
		t.Error("Add(full): returned true, want false")
	}
}

func TestMessagerCacheEmpty(t *testing.T) {
	mc := NewMessagerCache(2)
	if !mc.Add(&MessageRow{
		TimeNext: 1,
		Epoch:    0,
		id:       "row01",
	}) {
		t.Fatal("Add returned false")
	}
	mc.Clear()
	if row := mc.Pop(); row != nil {
		t.Errorf("Pop(empty): %v, want nil", row)
	}
	if !mc.Add(&MessageRow{
		TimeNext: 1,
		Epoch:    0,
		id:       "row01",
	}) {
		t.Fatal("Add returned false")
	}
	if row := mc.Pop(); row == nil {
		t.Errorf("Pop(non-empty): nil, want %v", row)
	}
}
