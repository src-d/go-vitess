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

package endtoend

import (
	"io"
	"reflect"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/youtube/vitess/go/sqltypes"
	"github.com/youtube/vitess/go/vt/vttablet/endtoend/framework"

	querypb "github.com/youtube/vitess/go/vt/proto/query"
)

var createMessage = `create table vitess_message(
	time_scheduled bigint,
	id bigint,
	time_next bigint,
	epoch bigint,
	time_created bigint,
	time_acked bigint,
	message varchar(128),
	primary key(time_scheduled, id),
	unique index id_idx(id),
	index next_idx(time_next, epoch))
comment 'vitess_message,vt_ack_wait=1,vt_purge_after=3,vt_batch_size=2,vt_cache_size=10,vt_poller_interval=1'`

func TestMessage(t *testing.T) {
	ch := make(chan *sqltypes.Result)
	done := make(chan struct{})
	client := framework.NewClient()
	if _, err := client.Execute(createMessage, nil); err != nil {
		t.Fatal(err)
	}
	defer client.Execute("drop table vitess_message", nil)

	// Start goroutine to consume message stream.
	go func() {
		if err := client.MessageStream("vitess_message", func(qr *sqltypes.Result) error {
			select {
			case <-done:
				return io.EOF
			default:
			}
			ch <- qr
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		close(ch)
	}()
	// Once the test is done, consume any left-over pending
	// messages. Some could make it into the pipeline and get
	// stuck forever causing vttablet shutdown to hang.
	defer func() {
		go func() {
			for range ch {
			}
		}()
	}()
	got := <-ch
	want := &sqltypes.Result{
		Fields: []*querypb.Field{{
			Name: "id",
			Type: sqltypes.Int64,
		}, {
			Name: "time_scheduled",
			Type: sqltypes.Int64,
		}, {
			Name: "message",
			Type: sqltypes.VarChar,
		}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("message(field) received:\n%v, want\n%v", got, want)
	}
	runtime.Gosched()
	defer func() { close(done) }()

	// Create message.
	err := client.Begin(false)
	if err != nil {
		t.Error(err)
		return
	}
	_, err = client.Execute("insert into vitess_message(id, message) values(1, 'hello world')", nil)
	if err != nil {
		t.Error(err)
		return
	}
	err = client.Commit()
	if err != nil {
		t.Error(err)
		return
	}

	// Consume first message.
	start := time.Now().UnixNano()
	got = <-ch
	// Check time_scheduled separately.
	scheduled, err := sqltypes.ToInt64(got.Rows[0][1])
	if err != nil {
		t.Error(err)
	}
	if now := time.Now().UnixNano(); now-scheduled >= int64(10*time.Second) {
		t.Errorf("scheduled: %v, must be close to %v", scheduled, now)
	}
	want = &sqltypes.Result{
		Rows: [][]sqltypes.Value{{
			sqltypes.NewInt64(1),
			got.Rows[0][1],
			sqltypes.NewVarChar("hello world"),
		}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("message received:\n%v, want\n%v", got, want)
	}
	qr, err := client.Execute("select time_next, epoch from vitess_message where id = 1", nil)
	if err != nil {
		t.Fatal(err)
	}
	next, epoch := getTimeEpoch(qr)
	// epoch could be 0 or 1, depending on how fast the row is updated
	switch epoch {
	case 0:
		if !(start-1e9 < next && next < start) {
			t.Errorf("next: %d. must be within 1s of start: %d", next/1e9, start/1e9)
		}
	case 1:
		if !(start < next && next < start+3e9) {
			t.Errorf("next: %d. must be about 1s after start: %d", next/1e9, start/1e9)
		}
	default:
		t.Errorf("epoch: %d, must be 0 or 1", epoch)
	}

	// Consume the resend.
	<-ch
	qr, err = client.Execute("select time_next, epoch from vitess_message where id = 1", nil)
	if err != nil {
		t.Fatal(err)
	}
	next, epoch = getTimeEpoch(qr)
	// epoch could be 1 or 2, depending on how fast the row is updated
	switch epoch {
	case 1:
		if !(start < next && next < start+3e9) {
			t.Errorf("next: %d. must be about 1s after start: %d", next/1e9, start/1e9)
		}
	case 2:
		if !(start+2e9 < next && next < start+6e9) {
			t.Errorf("next: %d. must be about 3s after start: %d", next/1e9, start/1e9)
		}
	default:
		t.Errorf("epoch: %d, must be 1 or 2", epoch)
	}

	// Ack the message.
	count, err := client.MessageAck("vitess_message", []string{"1"})
	if err != nil {
		t.Error(err)
	}
	if count != 1 {
		t.Errorf("count: %d, want 1", count)
	}
	qr, err = client.Execute("select time_acked, epoch from vitess_message where id = 1", nil)
	if err != nil {
		t.Fatal(err)
	}
	end := time.Now().UnixNano()
	ack, _ := getTimeEpoch(qr)
	if !(end-1e9 < ack && ack < end) {
		t.Errorf("ack: %d. must be within 1s of end: %d", ack/1e9, end/1e9)
	}

	// Within 3+1 seconds, the row should be deleted.
	time.Sleep(4 * time.Second)
	qr, err = client.Execute("select time_acked, epoch from vitess_message where id = 1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if qr.RowsAffected != 0 {
		t.Error("The row has not been purged yet")
	}
}

var createThreeColMessage = `create table vitess_message3(
	time_scheduled bigint,
	id bigint,
	time_next bigint,
	epoch bigint,
	time_created bigint,
	time_acked bigint,
	msg1 varchar(128),
	msg2 bigint,
	primary key(time_scheduled, id),
	unique index id_idx(id),
	index next_idx(time_next, epoch))
comment 'vitess_message,vt_ack_wait=1,vt_purge_after=3,vt_batch_size=2,vt_cache_size=10,vt_poller_interval=1'`

func TestThreeColMessage(t *testing.T) {
	ch := make(chan *sqltypes.Result)
	done := make(chan struct{})
	client := framework.NewClient()
	if _, err := client.Execute(createThreeColMessage, nil); err != nil {
		t.Fatal(err)
	}
	defer client.Execute("drop table vitess_message3", nil)

	go func() {
		if err := client.MessageStream("vitess_message3", func(qr *sqltypes.Result) error {
			select {
			case <-done:
				return io.EOF
			default:
			}
			ch <- qr
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		close(ch)
	}()
	// Once the test is done, consume any left-over pending
	// messages. Some could make it into the pipeline and get
	// stuck forever causing vttablet shutdown to hang.
	defer func() {
		go func() {
			for range ch {
			}
		}()
	}()

	// Verify fields.
	got := <-ch
	want := &sqltypes.Result{
		Fields: []*querypb.Field{{
			Name: "id",
			Type: sqltypes.Int64,
		}, {
			Name: "time_scheduled",
			Type: sqltypes.Int64,
		}, {
			Name: "msg1",
			Type: sqltypes.VarChar,
		}, {
			Name: "msg2",
			Type: sqltypes.Int64,
		}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("message(field) received:\n%v, want\n%v", got, want)
	}
	runtime.Gosched()
	defer func() { close(done) }()
	err := client.Begin(false)
	if err != nil {
		t.Error(err)
		return
	}
	_, err = client.Execute("insert into vitess_message3(id, msg1, msg2) values(1, 'hello world', 3)", nil)
	if err != nil {
		t.Error(err)
		return
	}
	err = client.Commit()
	if err != nil {
		t.Error(err)
		return
	}

	// Verify row.
	got = <-ch
	want = &sqltypes.Result{
		Rows: [][]sqltypes.Value{{
			sqltypes.NewInt64(1),
			got.Rows[0][1],
			sqltypes.NewVarChar("hello world"),
			sqltypes.NewInt64(3),
		}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("message received:\n%v, want\n%v", got, want)
	}

	// Verify Ack.
	count, err := client.MessageAck("vitess_message3", []string{"1"})
	if err != nil {
		t.Error(err)
	}
	if count != 1 {
		t.Errorf("count: %d, want 1", count)
	}
}

func getTimeEpoch(qr *sqltypes.Result) (int64, int64) {
	if len(qr.Rows) != 1 {
		return 0, 0
	}
	t, _ := strconv.Atoi(qr.Rows[0][0].String())
	e, _ := strconv.Atoi(qr.Rows[0][1].String())
	return int64(t), int64(e)
}
