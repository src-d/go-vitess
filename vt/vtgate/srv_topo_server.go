// Copyright 2013, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vtgate

import (
	"flag"
	"sync"
	"time"

	log "github.com/golang/glog"

	"github.com/youtube/vitess/go/stats"
	"github.com/youtube/vitess/go/vt/health"
	"github.com/youtube/vitess/go/vt/topo"
)

var (
	srvTopoCacheTTL = flag.Duration("srv_topo_cache_ttl", 1*time.Second, "how long to use cached entries for topology")
)

const (
	queryCategory  = "query"
	cachedCategory = "cached"
	errorCategory  = "error"
)

// SrvTopoServer is a subset of topo.Server that only contains the serving
// graph read-only calls used by clients to resolve serving addresses.
type SrvTopoServer interface {
	GetSrvKeyspaceNames(cell string) ([]string, error)

	GetSrvKeyspace(cell, keyspace string) (*topo.SrvKeyspace, error)

	GetEndPoints(cell, keyspace, shard string, tabletType topo.TabletType) (*topo.EndPoints, error)
}

// ResilientSrvTopoServer is an implementation of SrvTopoServer based
// on another SrvTopoServer that uses a cache for two purposes:
// - limit the QPS to the underlying SrvTopoServer
// - return the last known value of the data if there is an error
type ResilientSrvTopoServer struct {
	topoServer SrvTopoServer
	cacheTTL   time.Duration
	counts     *stats.Counters

	// mutex protects the cache map itself, not the individual
	// values in the cache.
	mutex                 sync.Mutex
	srvKeyspaceNamesCache map[string]*srvKeyspaceNamesEntry
	srvKeyspaceCache      map[string]*srvKeyspaceEntry
	endPointsCache        map[string]*endPointsEntry
}

type srvKeyspaceNamesEntry struct {
	// the mutex protects any access to this structure (read or write)
	mutex sync.Mutex

	insertionTime time.Time
	value         []string
	lastError     error
}

type srvKeyspaceEntry struct {
	// the mutex protects any access to this structure (read or write)
	mutex sync.Mutex

	insertionTime time.Time
	value         *topo.SrvKeyspace
	lastError     error
}

type endPointsEntry struct {
	// the mutex protects any access to this structure (read or write)
	mutex sync.Mutex

	insertionTime time.Time

	// Value is the end points that were returned to the client.
	Value *topo.EndPoints
	// OriginalValue is the end points that were returned from
	// the topology server.
	OriginalValue *topo.EndPoints
	lastError     error
}

// filterUnhealthyServers removes the unhealthy servers from the list,
// unless all servers are unhealthy, then it keeps them all.
func filterUnhealthyServers(endPoints *topo.EndPoints) *topo.EndPoints {

	// no endpoints, return right away
	if endPoints == nil || len(endPoints.Entries) == 0 {
		return endPoints
	}

	healthyEndPoints := make([]topo.EndPoint, 0, len(endPoints.Entries))
	for _, ep := range endPoints.Entries {
		// if we are behind on replication, we're not 100% healthy
		if ep.Health != nil && ep.Health[health.ReplicationLag] == health.ReplicationLagHigh {
			continue
		}

		healthyEndPoints = append(healthyEndPoints, ep)
	}

	// we have healthy guys, we return them
	if len(healthyEndPoints) > 0 {
		return &topo.EndPoints{Entries: healthyEndPoints}
	}

	// we only have unhealthy guys, return them
	return endPoints
}

// NewResilientSrvTopoServer creates a new ResilientSrvTopoServer
// based on the provided SrvTopoServer.
func NewResilientSrvTopoServer(base SrvTopoServer, counterName string) *ResilientSrvTopoServer {
	return &ResilientSrvTopoServer{
		topoServer: base,
		cacheTTL:   *srvTopoCacheTTL,
		counts:     stats.NewCounters(counterName),

		srvKeyspaceNamesCache: make(map[string]*srvKeyspaceNamesEntry),
		srvKeyspaceCache:      make(map[string]*srvKeyspaceEntry),
		endPointsCache:        make(map[string]*endPointsEntry),
	}
}

func (server *ResilientSrvTopoServer) GetSrvKeyspaceNames(cell string) ([]string, error) {
	server.counts.Add(queryCategory, 1)

	// find the entry in the cache, add it if not there
	key := cell
	server.mutex.Lock()
	entry, ok := server.srvKeyspaceNamesCache[key]
	if !ok {
		entry = &srvKeyspaceNamesEntry{}
		server.srvKeyspaceNamesCache[key] = entry
	}
	server.mutex.Unlock()

	// Lock the entry, and do everything holding the lock.  This
	// means two concurrent requests will only issue one
	// underlying query.
	entry.mutex.Lock()
	defer entry.mutex.Unlock()

	// If the entry is fresh enough, return it
	if time.Now().Sub(entry.insertionTime) < server.cacheTTL {
		return entry.value, entry.lastError
	}

	// not in cache or too old, get the real value
	result, err := server.topoServer.GetSrvKeyspaceNames(cell)
	if err != nil {
		if entry.insertionTime.IsZero() {
			server.counts.Add(errorCategory, 1)
			log.Errorf("GetSrvKeyspaceNames(%v) failed: %v (no cached value, caching and returning error)", cell, err)
		} else {
			server.counts.Add(cachedCategory, 1)
			log.Warningf("GetSrvKeyspaceNames(%v) failed: %v (returning cached value)", cell, err)
			return entry.value, nil
		}
	}

	// save the value we got and the current time in the cache
	entry.insertionTime = time.Now()
	entry.value = result
	entry.lastError = err
	return result, err
}

func (server *ResilientSrvTopoServer) GetSrvKeyspace(cell, keyspace string) (*topo.SrvKeyspace, error) {
	server.counts.Add(queryCategory, 1)

	// find the entry in the cache, add it if not there
	key := cell + ":" + keyspace
	server.mutex.Lock()
	entry, ok := server.srvKeyspaceCache[key]
	if !ok {
		entry = &srvKeyspaceEntry{}
		server.srvKeyspaceCache[key] = entry
	}
	server.mutex.Unlock()

	// Lock the entry, and do everything holding the lock.  This
	// means two concurrent requests will only issue one
	// underlying query.
	entry.mutex.Lock()
	defer entry.mutex.Unlock()

	// If the entry is fresh enough, return it
	if time.Now().Sub(entry.insertionTime) < server.cacheTTL {
		return entry.value, entry.lastError
	}

	// not in cache or too old, get the real value
	result, err := server.topoServer.GetSrvKeyspace(cell, keyspace)
	if err != nil {
		if entry.insertionTime.IsZero() {
			server.counts.Add(errorCategory, 1)
			log.Errorf("GetSrvKeyspace(%v, %v) failed: %v (no cached value, caching and returning error)", cell, keyspace, err)
		} else {
			server.counts.Add(cachedCategory, 1)
			log.Warningf("GetSrvKeyspace(%v, %v) failed: %v (returning cached value)", cell, keyspace, err)
			return entry.value, nil
		}
	}

	// save the value we got and the current time in the cache
	entry.insertionTime = time.Now()
	entry.value = result
	entry.lastError = err
	return result, err
}

func (server *ResilientSrvTopoServer) GetEndPoints(cell, keyspace, shard string, tabletType topo.TabletType) (*topo.EndPoints, error) {
	server.counts.Add(queryCategory, 1)

	// find the entry in the cache, add it if not there
	key := cell + ":" + keyspace + ":" + shard + ":" + string(tabletType)
	server.mutex.Lock()
	entry, ok := server.endPointsCache[key]
	if !ok {
		entry = &endPointsEntry{}
		server.endPointsCache[key] = entry
	}
	server.mutex.Unlock()

	// Lock the entry, and do everything holding the lock.  This
	// means two concurrent requests will only issue one
	// underlying query.
	entry.mutex.Lock()
	defer entry.mutex.Unlock()

	// If the entry is fresh enough, return it
	if time.Now().Sub(entry.insertionTime) < server.cacheTTL {
		return entry.Value, entry.lastError
	}

	// not in cache or too old, get the real value
	result, err := server.topoServer.GetEndPoints(cell, keyspace, shard, tabletType)
	if err != nil {
		if entry.insertionTime.IsZero() {
			server.counts.Add(errorCategory, 1)
			log.Errorf("GetEndPoints(%v, %v, %v, %v) failed: %v (no cached value, caching and returning error)", cell, keyspace, shard, tabletType, err)
		}
		server.counts.Add(cachedCategory, 1)
		log.Warningf("GetEndPoints(%v, %v, %v, %v) failed: %v (returning cached value)", cell, keyspace, shard, tabletType, err)
		return entry.Value, nil
	}

	// save the value we got and the current time in the cache
	entry.insertionTime = time.Now()
	entry.OriginalValue = result
	entry.Value = filterUnhealthyServers(result)
	entry.lastError = err

	return entry.Value, err
}

func (server *ResilientSrvTopoServer) HealthyEndpointCount() map[string]int64 {
	result := make(map[string]int64)
	server.mutex.Lock()
	defer server.mutex.Unlock()
	for k, entry := range server.endPointsCache {
		entry.mutex.Lock()
		result[k] = int64(len(entry.Value.Entries))
		entry.mutex.Unlock()
	}
	return result
}

func (server *ResilientSrvTopoServer) DegradedEndpointCount() map[string]int64 {
	result := make(map[string]int64)
	server.mutex.Lock()
	defer server.mutex.Unlock()
	for k, entry := range server.endPointsCache {
		entry.mutex.Lock()
		result[k] = int64(len(entry.OriginalValue.Entries) - len(entry.Value.Entries))
		entry.mutex.Unlock()
	}
	return result
}
