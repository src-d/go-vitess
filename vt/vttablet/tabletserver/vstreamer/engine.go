/*
Copyright 2018 The Vitess Authors.

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

package vstreamer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"

	"gopkg.in/src-d/go-vitess.v1/acl"
	"gopkg.in/src-d/go-vitess.v1/mysql"
	"gopkg.in/src-d/go-vitess.v1/sqltypes"
	"gopkg.in/src-d/go-vitess.v1/stats"
	"gopkg.in/src-d/go-vitess.v1/vt/dbconfigs"
	"gopkg.in/src-d/go-vitess.v1/vt/log"
	"gopkg.in/src-d/go-vitess.v1/vt/srvtopo"
	"gopkg.in/src-d/go-vitess.v1/vt/topo"
	"gopkg.in/src-d/go-vitess.v1/vt/vtgate/vindexes"
	"gopkg.in/src-d/go-vitess.v1/vt/vttablet/tabletserver/schema"

	binlogdatapb "gopkg.in/src-d/go-vitess.v1/vt/proto/binlogdata"
	vschemapb "gopkg.in/src-d/go-vitess.v1/vt/proto/vschema"
)

var (
	once           sync.Once
	vschemaErrors  *stats.Counter
	vschemaUpdates *stats.Counter
)

// Engine is the engine for handling vreplication streaming requests.
type Engine struct {
	// cp is initialized by InitDBConfig
	cp *mysql.ConnParams

	// mu protects isOpen, streamers, streamIdx and kschema.
	mu sync.Mutex

	isOpen bool
	// wg is incremented for every Stream, and decremented on end.
	// Close waits for all current streams to end by waiting on wg.
	wg           sync.WaitGroup
	streamers    map[int]*vstreamer
	rowStreamers map[int]*rowStreamer
	streamIdx    int

	// watcherOnce is used for initializing kschema
	// and setting up the vschema watch. It's guaranteed that
	// no stream will start until kschema is initialized by
	// the first call through watcherOnce.
	watcherOnce sync.Once
	kschema     *vindexes.KeyspaceSchema

	// The following members are initialized once at the beginning.
	ts       srvtopo.Server
	se       *schema.Engine
	keyspace string
	cell     string
}

// NewEngine creates a new Engine.
// Initialization sequence is: NewEngine->InitDBConfig->Open.
// Open and Close can be called multiple times and are idempotent.
func NewEngine(ts srvtopo.Server, se *schema.Engine) *Engine {
	vse := &Engine{
		streamers:    make(map[int]*vstreamer),
		rowStreamers: make(map[int]*rowStreamer),
		kschema:      &vindexes.KeyspaceSchema{},
		ts:           ts,
		se:           se,
	}
	once.Do(func() {
		vschemaErrors = stats.NewCounter("VSchemaErrors", "Count of VSchema errors")
		vschemaUpdates = stats.NewCounter("VSchemaUpdates", "Count of VSchema updates. Does not include errors")
		http.Handle("/debug/tablet_vschema", vse)
	})
	return vse
}

// InitDBConfig performs saves the required info from dbconfigs for future use.
func (vse *Engine) InitDBConfig(dbcfgs *dbconfigs.DBConfigs) {
	vse.cp = dbcfgs.DbaWithDB()
}

// Open starts the Engine service.
func (vse *Engine) Open(keyspace, cell string) error {
	vse.mu.Lock()
	defer vse.mu.Unlock()
	if vse.isOpen {
		return nil
	}
	vse.isOpen = true
	vse.keyspace = keyspace
	vse.cell = cell
	return nil
}

// Close closes the Engine service.
func (vse *Engine) Close() {
	func() {
		vse.mu.Lock()
		defer vse.mu.Unlock()
		if !vse.isOpen {
			return
		}
		for _, s := range vse.streamers {
			// cancel is non-blocking.
			s.Cancel()
		}
		for _, s := range vse.rowStreamers {
			// cancel is non-blocking.
			s.Cancel()
		}
		vse.isOpen = false
	}()

	// Wait only after releasing the lock because the end of every
	// stream will use the lock to remove the entry from streamers.
	vse.wg.Wait()
}

func (vse *Engine) vschema() *vindexes.KeyspaceSchema {
	vse.mu.Lock()
	defer vse.mu.Unlock()
	return vse.kschema
}

// Stream starts a new stream.
func (vse *Engine) Stream(ctx context.Context, startPos string, filter *binlogdatapb.Filter, send func([]*binlogdatapb.VEvent) error) error {
	// Ensure kschema is initialized and the watcher is started.
	// Starting of the watcher has to be delayed till the first call to Stream
	// because this overhead should be incurred only if someone uses this feature.
	vse.watcherOnce.Do(vse.setWatch)

	// Create stream and add it to the map.
	streamer, idx, err := func() (*vstreamer, int, error) {
		vse.mu.Lock()
		defer vse.mu.Unlock()
		if !vse.isOpen {
			return nil, 0, errors.New("VStreamer is not open")
		}
		streamer := newVStreamer(ctx, vse.cp, vse.se, startPos, filter, vse.kschema, send)
		idx := vse.streamIdx
		vse.streamers[idx] = streamer
		vse.streamIdx++
		// Now that we've added the stream, increment wg.
		// This must be done before releasing the lock.
		vse.wg.Add(1)
		return streamer, idx, nil
	}()
	if err != nil {
		return err
	}

	// Remove stream from map and decrement wg when it ends.
	defer func() {
		vse.mu.Lock()
		defer vse.mu.Unlock()
		delete(vse.streamers, idx)
		vse.wg.Done()
	}()

	// No lock is held while streaming, but wg is incremented.
	return streamer.Stream()
}

// StreamRows streams rows.
func (vse *Engine) StreamRows(ctx context.Context, query string, lastpk []sqltypes.Value, send func(*binlogdatapb.VStreamRowsResponse) error) error {
	// Ensure kschema is initialized and the watcher is started.
	// Starting of the watcher has to be delayed till the first call to Stream
	// because this overhead should be incurred only if someone uses this feature.
	vse.watcherOnce.Do(vse.setWatch)
	log.Infof("Streaming rows for query %s, lastpk: %s", query, lastpk)

	// Create stream and add it to the map.
	rowStreamer, idx, err := func() (*rowStreamer, int, error) {
		vse.mu.Lock()
		defer vse.mu.Unlock()
		if !vse.isOpen {
			return nil, 0, errors.New("VStreamer is not open")
		}
		rowStreamer := newRowStreamer(ctx, vse.cp, vse.se, query, lastpk, vse.kschema, send)
		idx := vse.streamIdx
		vse.rowStreamers[idx] = rowStreamer
		vse.streamIdx++
		// Now that we've added the stream, increment wg.
		// This must be done before releasing the lock.
		vse.wg.Add(1)
		return rowStreamer, idx, nil
	}()
	if err != nil {
		return err
	}

	// Remove stream from map and decrement wg when it ends.
	defer func() {
		vse.mu.Lock()
		defer vse.mu.Unlock()
		delete(vse.rowStreamers, idx)
		vse.wg.Done()
	}()

	// No lock is held while streaming, but wg is incremented.
	return rowStreamer.Stream()
}

// ServeHTTP shows the current VSchema.
func (vse *Engine) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if err := acl.CheckAccessHTTP(request, acl.DEBUGGING); err != nil {
		acl.SendError(response, err)
		return
	}
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	vs := vse.vschema()
	if vs == nil || vs.Keyspace == nil {
		response.Write([]byte("{}"))
	}
	b, err := json.MarshalIndent(vs, "", "  ")
	if err != nil {
		response.Write([]byte(err.Error()))
		return
	}
	buf := bytes.NewBuffer(nil)
	json.HTMLEscape(buf, b)
	response.Write(buf.Bytes())
}

func (vse *Engine) setWatch() {
	// WatchSrvVSchema does not return until the inner func has been called at least once.
	vse.ts.WatchSrvVSchema(context.TODO(), vse.cell, func(v *vschemapb.SrvVSchema, err error) {
		var kschema *vindexes.KeyspaceSchema
		switch {
		case err == nil:
			kschema, err = vindexes.BuildKeyspaceSchema(v.Keyspaces[vse.keyspace], vse.keyspace)
			if err != nil {
				log.Errorf("Error building vschema %s: %v", vse.keyspace, err)
				vschemaErrors.Add(1)
				return
			}
		case topo.IsErrType(err, topo.NoNode):
			// No-op.
		default:
			log.Errorf("Error fetching vschema %s: %v", vse.keyspace, err)
			vschemaErrors.Add(1)
			return
		}

		if kschema == nil {
			kschema = &vindexes.KeyspaceSchema{
				Keyspace: &vindexes.Keyspace{
					Name: vse.keyspace,
				},
			}
		}

		// Broadcast the change to all streamers.
		vse.mu.Lock()
		defer vse.mu.Unlock()
		vse.kschema = kschema
		b, _ := json.MarshalIndent(kschema, "", "  ")
		log.Infof("Updated KSchema: %s", b)
		for _, s := range vse.streamers {
			s.SetKSchema(kschema)
		}
		vschemaUpdates.Add(1)
	})
}
