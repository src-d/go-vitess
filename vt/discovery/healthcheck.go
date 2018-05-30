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

// Package discovery provides a way to discover all tablets e.g. within a
// specific shard and monitor their current health.
//
// Use the HealthCheck object to query for tablets and their health.
//
// For an example how to use the HealthCheck object, see worker/topo_utils.go.
//
// Tablets have to be manually added to the HealthCheck using AddTablet().
// Alternatively, use a Watcher implementation which will constantly watch
// a source (e.g. the topology) and add and remove tablets as they are
// added or removed from the source.
// For a Watcher example have a look at NewShardReplicationWatcher().
//
// Each HealthCheck has a HealthCheckStatsListener that will receive
// notification of when tablets go up and down.
// TabletStatsCache is one implementation, that caches the known tablets
// and the healthy ones per keyspace/shard/tabletType.
//
// Internally, the HealthCheck module is connected to each tablet and has a
// streaming RPC (StreamHealth) open to receive periodic health infos.
package discovery

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"hash/crc32"
	"html/template"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
	"vitess.io/vitess/go/netutil"
	"vitess.io/vitess/go/stats"
	"vitess.io/vitess/go/vt/grpcclient"
	"vitess.io/vitess/go/vt/log"
	"vitess.io/vitess/go/vt/topo/topoproto"
	"vitess.io/vitess/go/vt/topotools"
	"vitess.io/vitess/go/vt/vttablet/queryservice"
	"vitess.io/vitess/go/vt/vttablet/tabletconn"

	querypb "vitess.io/vitess/go/vt/proto/query"
	topodatapb "vitess.io/vitess/go/vt/proto/topodata"
)

var (
	hcErrorCounters          = stats.NewCountersWithMultiLabels("HealthcheckErrors", "Healthcheck Errors", []string{"Keyspace", "ShardName", "TabletType"})
	hcMasterPromotedCounters = stats.NewCountersWithMultiLabels("HealthcheckMasterPromoted", "Master promoted in keyspace/shard name because of health check errors", []string{"Keyspace", "ShardName"})
	healthcheckOnce          sync.Once
	tabletURLTemplateString  = flag.String("tablet_url_template", "http://{{.GetTabletHostPort}}", "format string describing debug tablet url formatting. See the Go code for getTabletDebugURL() how to customize this.")
	tabletURLTemplate        *template.Template
)

// See the documentation for NewHealthCheck below for an explanation of these parameters.
const (
	DefaultHealthCheckRetryDelay = 5 * time.Second
	DefaultHealthCheckTimeout    = 1 * time.Minute
)

const (
	// DefaultTopoReadConcurrency can be used as default value for the topoReadConcurrency parameter of a TopologyWatcher.
	DefaultTopoReadConcurrency int = 5
	// DefaultTopologyWatcherRefreshInterval can be used as the default value for
	// the refresh interval of a topology watcher.
	DefaultTopologyWatcherRefreshInterval = 1 * time.Minute

	// HealthCheckTemplate is the HTML code to display a TabletsCacheStatusList
	HealthCheckTemplate = `
<style>
  table {
    border-collapse: collapse;
  }
  td, th {
    border: 1px solid #999;
    padding: 0.2rem;
  }
</style>
<table>
  <tr>
    <th colspan="5">HealthCheck Tablet Cache</th>
  </tr>
  <tr>
    <th>Cell</th>
    <th>Keyspace</th>
    <th>Shard</th>
    <th>TabletType</th>
    <th>TabletStats</th>
  </tr>
  {{range $i, $ts := .}}
  <tr>
    <td>{{github_com_vitessio_vitess_vtctld_srv_cell $ts.Cell}}</td>
    <td>{{github_com_vitessio_vitess_vtctld_srv_keyspace $ts.Cell $ts.Target.Keyspace}}</td>
    <td>{{$ts.Target.Shard}}</td>
    <td>{{$ts.Target.TabletType}}</td>
    <td>{{$ts.StatusAsHTML}}</td>
  </tr>
  {{end}}
</table>
`
)

func init() {
	loadTabletURLTemplate()
}

// loadTabletURLTemplate loads or reloads the URL template.
// Should only be used independently for testing.
func loadTabletURLTemplate() {
	tabletURLTemplate = template.New("")
	_, err := tabletURLTemplate.Parse(*tabletURLTemplateString)
	if err != nil {
		log.Exitf("error parsing template: %v", err)
	}
}

// HealthCheckStatsListener is the listener to receive health check stats update.
type HealthCheckStatsListener interface {
	// StatsUpdate is called when:
	// - a new tablet is known to the HealthCheck, and its first
	//   streaming healthcheck is returned. (then ts.Up is true).
	// - a tablet is removed from the list of tablets we watch
	//   (then ts.Up is false).
	// - a tablet dynamically changes its type. When registering the
	//   listener, if sendDownEvents is true, two events are generated
	//   (ts.Up false on the old type, ts.Up true on the new type).
	//   If it is false, only one event is sent (ts.Up true on the new
	//   type).
	StatsUpdate(*TabletStats)
}

// TabletStats is returned when getting the set of tablets.
type TabletStats struct {
	// Key uniquely identifies that serving tablet. It is computed
	// from the Tablet's record Hostname and PortMap. If a tablet
	// is restarted on different ports, its Key will be different.
	// Key is computed using the TabletToMapKey method below.
	// key can be used in GetConnection().
	Key string
	// Tablet is the tablet object that was sent to HealthCheck.AddTablet.
	Tablet *topodatapb.Tablet
	// Name is an optional tag (e.g. alternative address) for the
	// tablet.  It is supposed to represent the tablet as a task,
	// not as a process.  For instance, it can be a
	// cell+keyspace+shard+tabletType+taskIndex value.
	Name string
	// Target is the current target as returned by the streaming
	// StreamHealth RPC.
	Target *querypb.Target
	// Up describes whether the tablet is added or removed.
	Up bool
	// Serving describes if the tablet can be serving traffic.
	Serving bool
	// TabletExternallyReparentedTimestamp is the last timestamp
	// that this tablet was either elected the master, or received
	// a TabletExternallyReparented event. It is set to 0 if the
	// tablet doesn't think it's a master.
	TabletExternallyReparentedTimestamp int64
	// Stats is the current health status, as received by the
	// StreamHealth RPC (replication lag, ...).
	Stats *querypb.RealtimeStats
	// LastError is the error we last saw when trying to get the
	// tablet's healthcheck.
	LastError error
}

// String is defined because we want to print a []*TabletStats array nicely.
func (e *TabletStats) String() string {
	return fmt.Sprint(*e)
}

// DeepEqual compares two TabletStats. Since we include protos, we
// need to use proto.Equal on these.
func (e *TabletStats) DeepEqual(f *TabletStats) bool {
	return e.Key == f.Key &&
		proto.Equal(e.Tablet, f.Tablet) &&
		e.Name == f.Name &&
		proto.Equal(e.Target, f.Target) &&
		e.Up == f.Up &&
		e.Serving == f.Serving &&
		e.TabletExternallyReparentedTimestamp == f.TabletExternallyReparentedTimestamp &&
		proto.Equal(e.Stats, f.Stats) &&
		((e.LastError == nil && f.LastError == nil) ||
			(e.LastError != nil && f.LastError != nil && e.LastError.Error() == f.LastError.Error()))
}

// GetTabletHostPort formats a tablet host port address.
func (e TabletStats) GetTabletHostPort() string {
	vtPort := e.Tablet.PortMap["vt"]
	return netutil.JoinHostPort(e.Tablet.Hostname, vtPort)
}

// GetHostNameLevel returns the specified hostname level. If the level does not exist it will pick the closest level.
// This seems unused but can be utilized by certain url formatting templates. See getTabletDebugURL for more details.
func (e TabletStats) GetHostNameLevel(level int) string {
	chunkedHostname := strings.Split(e.Tablet.Hostname, ".")

	if level < 0 {
		return chunkedHostname[0]
	} else if level >= len(chunkedHostname) {
		return chunkedHostname[len(chunkedHostname)-1]
	} else {
		return chunkedHostname[level]
	}
}

// getTabletDebugURL formats a debug url to the tablet.
// It uses a format string that can be passed into the app to format
// the debug URL to accommodate different network setups. It applies
// the html/template string defined to a TabletStats object. The
// format string can refer to members and functions of TabletStats
// like a regular html/template string.
//
// For instance given a tablet with hostname:port of host.dc.domain:22
// could be configured as follows:
// http://{{.GetTabletHostPort}} -> http://host.dc.domain:22
// https://{{.Tablet.Hostname}} -> https://host.dc.domain
// https://{{.GetHostNameLevel 0}}.bastion.corp -> https://host.bastion.corp
func (e TabletStats) getTabletDebugURL() string {
	var buffer bytes.Buffer
	tabletURLTemplate.Execute(&buffer, e)
	return buffer.String()
}

// HealthCheck defines the interface of health checking module.
// The goal of this object is to maintain a StreamHealth RPC
// to a lot of tablets. Tablets are added / removed by calling the
// AddTablet / RemoveTablet methods (other discovery module objects
// can for instance watch the topology and call these).
//
// Updates to the health of all registered tablet can be watched by
// registering a listener. To get the underlying "TabletConn" object
// which is used for each tablet, use the "GetConnection()" method
// below and pass in the Key string which is also sent to the
// listener in each update (as it is part of TabletStats).
type HealthCheck interface {
	// TabletRecorder interface adds AddTablet and RemoveTablet methods.
	// AddTablet adds the tablet, and starts health check on it.
	// RemoveTablet removes the tablet, and stops its StreamHealth RPC.
	TabletRecorder

	// RegisterStats registers the connection counts and checksum stats.
	// It can only be called on one Healthcheck object per process.
	RegisterStats()
	// SetListener sets the listener for healthcheck
	// updates. sendDownEvents is used when a tablet changes type
	// (from replica to master for instance). If the listener
	// wants two events (Up=false on old type, Up=True on new
	// type), sendDownEvents should be set. Otherwise, the
	// healthcheck will only send one event (Up=true on new type).
	//
	// Note that the default implementation requires to set the
	// listener before any tablets are added to the healthcheck.
	SetListener(listener HealthCheckStatsListener, sendDownEvents bool)
	// WaitForInitialStatsUpdates waits until all tablets added via
	// AddTablet() call were propagated to the listener via corresponding
	// StatsUpdate() calls. Note that code path from AddTablet() to
	// corresponding StatsUpdate() is asynchronous but not cancelable, thus
	// this function is also non-cancelable and can't return error. Also
	// note that all AddTablet() calls should happen before calling this
	// method. WaitForInitialStatsUpdates won't wait for StatsUpdate() calls
	// corresponding to AddTablet() calls made during its execution.
	WaitForInitialStatsUpdates()
	// GetConnection returns the TabletConn of the given tablet.
	GetConnection(key string) queryservice.QueryService
	// CacheStatus returns a displayable version of the cache.
	CacheStatus() TabletsCacheStatusList
	// Close stops the healthcheck.
	Close() error
}

// healthCheckConn contains details about a tablet.
// It is used internally by HealthCheckImpl to keep all the info
// about a tablet.
type healthCheckConn struct {
	// set at construction time
	ctx        context.Context
	cancelFunc context.CancelFunc

	// mu protects all the following fields.
	// When locking both mutex from HealthCheck and healthCheckConn,
	// HealthCheck.mu goes first.
	// Note tabletStats.Tablet and tabletStats.Name are immutable.
	mu                    sync.RWMutex
	conn                  queryservice.QueryService
	streamCancelFunc      context.CancelFunc
	tabletStats           TabletStats
	loggedServingState    bool
	lastResponseTimestamp time.Time // timestamp of the last healthcheck response
}

// HealthCheckImpl performs health checking and notifies downstream components about any changes.
type HealthCheckImpl struct {
	// Immutable fields set at construction time.
	listener           HealthCheckStatsListener
	sendDownEvents     bool
	retryDelay         time.Duration
	healthCheckTimeout time.Duration
	closeChan          chan struct{} // signals the process gorouting to terminate
	// wg keeps track of all launched Go routines.
	wg sync.WaitGroup

	// mu protects all the following fields.
	// When locking both mutex from HealthCheck and healthCheckConn,
	// HealthCheck.mu goes first.
	mu sync.RWMutex

	// addrToConns maps from address to the healthCheckConn object.
	addrToConns map[string]*healthCheckConn

	// Wait group that's used to wait until all initial StatsUpdate() calls are made after the AddTablet() calls.
	initialUpdatesWG sync.WaitGroup
}

// NewDefaultHealthCheck creates a new HealthCheck object with a default configuration.
func NewDefaultHealthCheck() HealthCheck {
	return NewHealthCheck(DefaultHealthCheckRetryDelay, DefaultHealthCheckTimeout)
}

// NewHealthCheck creates a new HealthCheck object.
// Parameters:
// retryDelay.
//   The duration to wait before retrying to connect (e.g. after a failed connection
//   attempt).
// healthCheckTimeout.
//   The duration for which we consider a health check response to be 'fresh'. If we don't get
//   a health check response from a tablet for more than this duration, we consider the tablet
//   not healthy.
func NewHealthCheck(retryDelay, healthCheckTimeout time.Duration) HealthCheck {
	hc := &HealthCheckImpl{
		addrToConns:        make(map[string]*healthCheckConn),
		retryDelay:         retryDelay,
		healthCheckTimeout: healthCheckTimeout,
		closeChan:          make(chan struct{}),
	}

	hc.wg.Add(1)
	go func() {
		defer hc.wg.Done()
		// Start another go routine to check timeout.
		// Currently vttablet sends healthcheck response every 20 seconds.
		// We set the default timeout to 1 minute (20s * 3),
		// and also perform the timeout check in sync with vttablet frequency.
		// When we change the healthcheck frequency on vttablet,
		// we should also adjust here.
		t := time.NewTicker(healthCheckTimeout / 3)
		defer t.Stop()
		for {
			select {
			case <-hc.closeChan:
				return
			case _, ok := <-t.C:
				if !ok {
					// the ticker stoped
					return
				}
				hc.checkHealthCheckTimeout()
			}
		}
	}()

	healthcheckOnce.Do(func() {
		http.Handle("/debug/gateway", hc)
	})

	return hc
}

// RegisterStats registers the connection counts stats
func (hc *HealthCheckImpl) RegisterStats() {
	stats.NewGaugesFuncWithMultiLabels(
		"HealthcheckConnections",
		"the number of healthcheck connections registered",
		[]string{"Keyspace", "ShardName", "TabletType"},
		hc.servingConnStats)

	stats.NewGaugeFunc(
		"HealthcheckChecksum",
		"crc32 checksum of the current healthcheck state",
		hc.stateChecksum)
}

// ServeHTTP is part of the http.Handler interface. It renders the current state of the discovery gateway tablet cache into json.
func (hc *HealthCheckImpl) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	status := hc.cacheStatusMap()
	b, err := json.MarshalIndent(status, "", " ")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	buf := bytes.NewBuffer(nil)
	json.HTMLEscape(buf, b)
	w.Write(buf.Bytes())
}

// servingConnStats returns the number of serving tablets per keyspace/shard/tablet type.
func (hc *HealthCheckImpl) servingConnStats() map[string]int64 {
	res := make(map[string]int64)
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	for _, hcc := range hc.addrToConns {
		hcc.mu.RLock()
		if !hcc.tabletStats.Up || !hcc.tabletStats.Serving || hcc.tabletStats.LastError != nil {
			hcc.mu.RUnlock()
			continue
		}
		key := fmt.Sprintf("%s.%s.%s", hcc.tabletStats.Target.Keyspace, hcc.tabletStats.Target.Shard, topoproto.TabletTypeLString(hcc.tabletStats.Target.TabletType))
		hcc.mu.RUnlock()
		res[key]++
	}
	return res
}

// stateChecksum returns a crc32 checksum of the healthcheck state
func (hc *HealthCheckImpl) stateChecksum() int64 {
	// CacheStatus is sorted so this should be stable across vtgates
	cacheStatus := hc.CacheStatus()
	var buf bytes.Buffer
	for _, st := range cacheStatus {
		fmt.Fprintf(&buf,
			"%v%v%v%v\n",
			st.Cell,
			st.Target.Keyspace,
			st.Target.Shard,
			st.Target.TabletType.String(),
		)
		sort.Sort(st.TabletsStats)
		for _, ts := range st.TabletsStats {
			fmt.Fprintf(&buf, "%v%v%v\n", ts.Up, ts.Serving, ts.TabletExternallyReparentedTimestamp)
		}
	}

	return int64(crc32.ChecksumIEEE(buf.Bytes()))
}

// finalizeConn closes the health checking connection and sends the final
// notification about the tablet to downstream. To be called only on exit from
// checkConn().
func (hc *HealthCheckImpl) finalizeConn(hcc *healthCheckConn) {
	hcc.mu.Lock()
	hccConn := hcc.conn
	hccCtx := hcc.ctx
	hcc.conn = nil
	hcc.tabletStats.Up = false
	hcc.setServingState(false, "finalizeConn closing connection")
	// Note: checkConn() exits only when hcc.ctx.Done() is closed. Thus it's
	// safe to simply get Err() value here and assign to LastError.
	hcc.tabletStats.LastError = hcc.ctx.Err()
	ts := hcc.tabletStats
	hcc.mu.Unlock()

	if hccConn != nil {
		hccConn.Close(hccCtx)
	}

	if hc.listener != nil {
		hc.listener.StatsUpdate(&ts)
	}
}

// checkConn performs health checking on the given tablet.
func (hc *HealthCheckImpl) checkConn(hcc *healthCheckConn, name string) {
	defer hc.wg.Done()
	defer hc.finalizeConn(hcc)

	// Initial notification for downstream about the tablet existence.
	hcc.mu.Lock()
	ts := hcc.tabletStats
	hcc.mu.Unlock()
	if hc.listener != nil {
		hc.listener.StatsUpdate(&ts)
	}
	hc.initialUpdatesWG.Done()

	retryDelay := hc.retryDelay
	for {
		ctx, cancel := context.WithCancel(hcc.ctx)
		hcc.mu.Lock()
		hcc.streamCancelFunc = cancel
		hcc.mu.Unlock()

		// Read stream health responses.
		hcc.stream(ctx, hc, func(shr *querypb.StreamHealthResponse) error {
			// We received a message. Reset the back-off.
			retryDelay = hc.retryDelay
			return hcc.processResponse(hc, shr)
		})

		// Streaming RPC failed e.g. because vttablet was restarted or took too long.
		// Sleep until the next retry is up or the context is done/canceled.
		select {
		case <-hcc.ctx.Done():
			return
		case <-time.After(retryDelay):
			// Exponentially back-off to prevent tight-loop.
			retryDelay *= 2
		}
	}
}

// setServingState sets the tablet state to the given value.
//
// If the state changes, it logs the change so that failures
// from the health check connection are logged the first time,
// but don't continue to log if the connection stays down.
//
// hcc.mu must be locked before calling this function
func (hcc *healthCheckConn) setServingState(serving bool, reason string) {
	if !hcc.loggedServingState || (serving != hcc.tabletStats.Serving) {
		// Emit the log from a separate goroutine to avoid holding
		// the hcc lock while logging is happening
		go log.Infof("HealthCheckUpdate(Serving State): %v, tablet: %v serving => %v for %v/%v (%v) reason: %s",
			hcc.tabletStats.Name,
			topotools.TabletIdent(hcc.tabletStats.Tablet),
			serving,
			hcc.tabletStats.Tablet.GetKeyspace(),
			hcc.tabletStats.Tablet.GetShard(),
			hcc.tabletStats.Target.GetTabletType(),
			reason,
		)
		hcc.loggedServingState = true
	}

	hcc.tabletStats.Serving = serving
}

// stream streams healthcheck responses to callback.
func (hcc *healthCheckConn) stream(ctx context.Context, hc *HealthCheckImpl, callback func(*querypb.StreamHealthResponse) error) {
	hcc.mu.Lock()
	conn := hcc.conn
	hcc.mu.Unlock()

	if conn == nil {
		var err error
		conn, err = tabletconn.GetDialer()(hcc.tabletStats.Tablet, grpcclient.FailFast(true))
		if err != nil {
			hcc.mu.Lock()
			hcc.tabletStats.LastError = err
			hcc.mu.Unlock()
			return
		}

		hcc.mu.Lock()
		hcc.conn = conn
		hcc.tabletStats.LastError = nil
		hcc.mu.Unlock()
	}

	if err := conn.StreamHealth(ctx, callback); err != nil {
		hcc.mu.Lock()
		hcc.conn = nil
		hcc.setServingState(false, err.Error())
		hcc.tabletStats.LastError = err
		ts := hcc.tabletStats
		hcc.mu.Unlock()
		// notify downstream for serving status change
		if hc.listener != nil {
			hc.listener.StatsUpdate(&ts)
		}
		conn.Close(ctx)
		return
	}
	return
}

// processResponse reads one health check response, and notifies HealthCheckStatsListener.
func (hcc *healthCheckConn) processResponse(hc *HealthCheckImpl, shr *querypb.StreamHealthResponse) error {
	select {
	case <-hcc.ctx.Done():
		return hcc.ctx.Err()
	default:
	}

	// Check for invalid data, better than panicking.
	if shr.Target == nil || shr.RealtimeStats == nil {
		return fmt.Errorf("health stats is not valid: %v", shr)
	}

	hcc.mu.RLock()
	oldTs := hcc.tabletStats
	hcc.mu.RUnlock()

	// an app-level error from tablet, force serving state.
	var healthErr error
	serving := shr.Serving
	if shr.RealtimeStats.HealthError != "" {
		healthErr = fmt.Errorf("vttablet error: %v", shr.RealtimeStats.HealthError)
		serving = false
	}

	// oldTs.Tablet.Alias.Uid may be 0 because the youtube internal mechanism uses a different
	// code path to initialize this value. If so, we should skip this check.
	if shr.TabletAlias != nil && oldTs.Tablet.Alias.Uid != 0 && !proto.Equal(shr.TabletAlias, oldTs.Tablet.Alias) {
		return fmt.Errorf("health stats mismatch, tablet %+v alias does not match response alias %v", oldTs.Tablet, shr.TabletAlias)
	}

	// In the case where a tablet changes type (but not for the
	// initial message), we want to log it, and maybe advertise it too.
	if hcc.tabletStats.Target.TabletType != topodatapb.TabletType_UNKNOWN && hcc.tabletStats.Target.TabletType != shr.Target.TabletType {
		// Log and maybe notify
		log.Infof("HealthCheckUpdate(Type Change): %v, tablet: %s, target %+v => %+v, reparent time: %v",
			oldTs.Name, topotools.TabletIdent(oldTs.Tablet), topotools.TargetIdent(oldTs.Target), topotools.TargetIdent(shr.Target), shr.TabletExternallyReparentedTimestamp)
		if hc.listener != nil && hc.sendDownEvents {
			oldTs.Up = false
			hc.listener.StatsUpdate(&oldTs)
		}

		// Track how often a tablet gets promoted to master. It is used for
		// comparing against the variables in go/vtgate/buffer/variables.go.
		if oldTs.Target.TabletType != topodatapb.TabletType_MASTER && shr.Target.TabletType == topodatapb.TabletType_MASTER {
			hcMasterPromotedCounters.Add([]string{shr.Target.Keyspace, shr.Target.Shard}, 1)
		}
	}

	// In this case where a new tablet is initialized or a tablet type changes, we want to
	// initialize the counter so the rate can be calculated correctly.
	if hcc.tabletStats.Target.TabletType != shr.Target.TabletType {
		hcErrorCounters.Add([]string{shr.Target.Keyspace, shr.Target.Shard, topoproto.TabletTypeLString(shr.Target.TabletType)}, 0)
	}

	// Update our record, and notify downstream for tabletType and
	// realtimeStats change.
	ts := hcc.update(shr, serving, healthErr)
	if hc.listener != nil {
		hc.listener.StatsUpdate(&ts)
	}
	return nil
}

// update updates the stats of a healthCheckConn, and returns a copy
// of its tabletStats.
func (hcc *healthCheckConn) update(shr *querypb.StreamHealthResponse, serving bool, healthErr error) TabletStats {
	hcc.mu.Lock()
	defer hcc.mu.Unlock()
	hcc.lastResponseTimestamp = time.Now()
	hcc.tabletStats.Target = shr.Target
	hcc.tabletStats.TabletExternallyReparentedTimestamp = shr.TabletExternallyReparentedTimestamp
	hcc.tabletStats.Stats = shr.RealtimeStats
	hcc.tabletStats.LastError = healthErr
	reason := "healthCheck update"
	if healthErr != nil {
		reason = "healthCheck update error: " + healthErr.Error()
	}
	hcc.setServingState(serving, reason)
	return hcc.tabletStats
}

func (hc *HealthCheckImpl) checkHealthCheckTimeout() {
	hc.mu.RLock()
	list := make([]*healthCheckConn, 0, len(hc.addrToConns))
	for _, hcc := range hc.addrToConns {
		list = append(list, hcc)
	}
	hc.mu.RUnlock()
	for _, hcc := range list {
		hcc.mu.RLock()
		if !hcc.tabletStats.Serving {
			// ignore non-serving tablet
			hcc.mu.RUnlock()
			continue
		}
		if time.Now().Sub(hcc.lastResponseTimestamp) < hc.healthCheckTimeout {
			// received a healthcheck response recently
			hcc.mu.RUnlock()
			continue
		}
		hcc.mu.RUnlock()
		// mark the tablet non-serving as we have not seen a health check response for a long time
		hcc.mu.Lock()
		// check again to avoid race condition
		if !hcc.tabletStats.Serving {
			// ignore non-serving tablet
			hcc.mu.Unlock()
			continue
		}
		if time.Now().Sub(hcc.lastResponseTimestamp) < hc.healthCheckTimeout {
			// received a healthcheck response recently
			hcc.mu.Unlock()
			continue
		}

		//Timeout detected. Cancel the current streaming RPC and let checkConn() restart it.
		hcc.streamCancelFunc()
		hcc.tabletStats.LastError = fmt.Errorf("healthcheck timed out (latest %v)", hcc.lastResponseTimestamp)
		hcc.setServingState(false, hcc.tabletStats.LastError.Error())
		ts := hcc.tabletStats
		hcc.mu.Unlock()
		// notify downstream for serving status change
		if hc.listener != nil {
			hc.listener.StatsUpdate(&ts)
		}
		hcErrorCounters.Add([]string{ts.Target.Keyspace, ts.Target.Shard, topoproto.TabletTypeLString(ts.Target.TabletType)}, 1)
	}
}

func (hc *HealthCheckImpl) deleteConn(tablet *topodatapb.Tablet) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	key := TabletToMapKey(tablet)
	hcc, ok := hc.addrToConns[key]
	if !ok {
		log.Warningf("deleting unknown tablet: %+v", tablet)
		return
	}
	hcc.mu.Lock()
	hcc.tabletStats.Up = false
	hcc.mu.Unlock()
	hcc.cancelFunc()
	delete(hc.addrToConns, key)
}

// SetListener sets the listener for healthcheck updates.
// It must be called after NewHealthCheck and before any tablets are added
// (either through AddTablet or through a Watcher).
func (hc *HealthCheckImpl) SetListener(listener HealthCheckStatsListener, sendDownEvents bool) {
	if hc.listener != nil {
		panic("must not call SetListener twice")
	}

	hc.mu.Lock()
	defer hc.mu.Unlock()
	if len(hc.addrToConns) > 0 {
		panic("must not call SetListener after tablets were added")
	}

	hc.listener = listener
	hc.sendDownEvents = sendDownEvents
}

// AddTablet adds the tablet, and starts health check.
// It does not block on making connection.
// name is an optional tag for the tablet, e.g. an alternative address.
func (hc *HealthCheckImpl) AddTablet(tablet *topodatapb.Tablet, name string) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	key := TabletToMapKey(tablet)
	hcc := &healthCheckConn{
		ctx:        ctx,
		cancelFunc: cancelFunc,
		tabletStats: TabletStats{
			Key:    key,
			Tablet: tablet,
			Name:   name,
			Target: &querypb.Target{},
			Up:     true,
		},
	}
	hc.mu.Lock()
	if _, ok := hc.addrToConns[key]; ok {
		hc.mu.Unlock()
		log.Warningf("adding duplicate tablet %v for %v: %+v", name, tablet.Alias.Cell, tablet)
		return
	}
	hc.addrToConns[key] = hcc
	hc.initialUpdatesWG.Add(1)
	hc.mu.Unlock()

	hc.wg.Add(1)
	go hc.checkConn(hcc, name)
}

// RemoveTablet removes the tablet, and stops the health check.
// It does not block.
func (hc *HealthCheckImpl) RemoveTablet(tablet *topodatapb.Tablet) {
	go hc.deleteConn(tablet)
}

// ReplaceTablet removes the old tablet and adds the new tablet.
func (hc *HealthCheckImpl) ReplaceTablet(old, new *topodatapb.Tablet, name string) {
	go func() {
		hc.deleteConn(old)
		hc.AddTablet(new, name)
	}()
}

// WaitForInitialStatsUpdates waits until all tablets added via AddTablet() call
// were propagated to downstream via corresponding StatsUpdate() calls.
func (hc *HealthCheckImpl) WaitForInitialStatsUpdates() {
	hc.initialUpdatesWG.Wait()
}

// GetConnection returns the TabletConn of the given tablet.
func (hc *HealthCheckImpl) GetConnection(key string) queryservice.QueryService {
	hc.mu.RLock()
	hcc := hc.addrToConns[key]
	if hcc == nil {
		hc.mu.RUnlock()
		return nil
	}
	hc.mu.RUnlock()
	hcc.mu.RLock()
	defer hcc.mu.RUnlock()
	return hcc.conn
}

// TabletsCacheStatus is the current tablets for a cell/target.
type TabletsCacheStatus struct {
	Cell         string
	Target       *querypb.Target
	TabletsStats TabletStatsList
}

// TabletStatsList is used for sorting.
type TabletStatsList []*TabletStats

// Len is part of sort.Interface.
func (tsl TabletStatsList) Len() int {
	return len(tsl)
}

// Less is part of sort.Interface
func (tsl TabletStatsList) Less(i, j int) bool {
	name1 := tsl[i].Name
	if name1 == "" {
		name1 = tsl[i].Key
	}
	name2 := tsl[j].Name
	if name2 == "" {
		name2 = tsl[j].Key
	}
	return name1 < name2
}

// Swap is part of sort.Interface
func (tsl TabletStatsList) Swap(i, j int) {
	tsl[i], tsl[j] = tsl[j], tsl[i]
}

// StatusAsHTML returns an HTML version of the status.
func (tcs *TabletsCacheStatus) StatusAsHTML() template.HTML {
	tLinks := make([]string, 0, 1)
	if tcs.TabletsStats != nil {
		sort.Sort(tcs.TabletsStats)
	}
	for _, ts := range tcs.TabletsStats {
		color := "green"
		extra := ""
		if ts.LastError != nil {
			color = "red"
			extra = fmt.Sprintf(" (%v)", ts.LastError)
		} else if !ts.Serving {
			color = "red"
			extra = " (Not Serving)"
		} else if !ts.Up {
			color = "red"
			extra = " (Down)"
		} else if ts.Target.TabletType == topodatapb.TabletType_MASTER {
			extra = fmt.Sprintf(" (MasterTS: %v)", ts.TabletExternallyReparentedTimestamp)
		} else {
			extra = fmt.Sprintf(" (RepLag: %v)", ts.Stats.SecondsBehindMaster)
		}
		name := ts.Name
		if name == "" {
			name = ts.GetTabletHostPort()
		}
		tLinks = append(tLinks, fmt.Sprintf(`<a href="%s" style="color:%v">%v</a>%v`, ts.getTabletDebugURL(), color, name, extra))
	}
	return template.HTML(strings.Join(tLinks, "<br>"))
}

// TabletsCacheStatusList is used for sorting.
type TabletsCacheStatusList []*TabletsCacheStatus

// Len is part of sort.Interface.
func (tcsl TabletsCacheStatusList) Len() int {
	return len(tcsl)
}

// Less is part of sort.Interface
func (tcsl TabletsCacheStatusList) Less(i, j int) bool {
	return tcsl[i].Cell+"."+tcsl[i].Target.Keyspace+"."+tcsl[i].Target.Shard+"."+string(tcsl[i].Target.TabletType) <
		tcsl[j].Cell+"."+tcsl[j].Target.Keyspace+"."+tcsl[j].Target.Shard+"."+string(tcsl[j].Target.TabletType)
}

// Swap is part of sort.Interface
func (tcsl TabletsCacheStatusList) Swap(i, j int) {
	tcsl[i], tcsl[j] = tcsl[j], tcsl[i]
}

// CacheStatus returns a displayable version of the cache.
func (hc *HealthCheckImpl) CacheStatus() TabletsCacheStatusList {
	tcsMap := hc.cacheStatusMap()
	tcsl := make(TabletsCacheStatusList, 0, len(tcsMap))
	for _, tcs := range tcsMap {
		tcsl = append(tcsl, tcs)
	}
	sort.Sort(tcsl)
	return tcsl
}

func (hc *HealthCheckImpl) cacheStatusMap() map[string]*TabletsCacheStatus {
	tcsMap := make(map[string]*TabletsCacheStatus)
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	for _, hcc := range hc.addrToConns {
		hcc.mu.RLock()
		key := fmt.Sprintf("%v.%v.%v.%v", hcc.tabletStats.Tablet.Alias.Cell, hcc.tabletStats.Target.Keyspace, hcc.tabletStats.Target.Shard, hcc.tabletStats.Target.TabletType.String())
		var tcs *TabletsCacheStatus
		var ok bool
		if tcs, ok = tcsMap[key]; !ok {
			tcs = &TabletsCacheStatus{
				Cell:   hcc.tabletStats.Tablet.Alias.Cell,
				Target: hcc.tabletStats.Target,
			}
			tcsMap[key] = tcs
		}
		stats := hcc.tabletStats
		hcc.mu.RUnlock()
		tcs.TabletsStats = append(tcs.TabletsStats, &stats)
	}
	return tcsMap
}

// Close stops the healthcheck.
// After Close() returned, it's guaranteed that the listener isn't
// currently executing and won't be called again.
func (hc *HealthCheckImpl) Close() error {
	hc.mu.Lock()
	close(hc.closeChan)
	for _, hcc := range hc.addrToConns {
		hcc.cancelFunc()
	}
	hc.addrToConns = make(map[string]*healthCheckConn)
	// Release the lock early or a pending checkHealthCheckTimeout
	// cannot get a read lock on it.
	hc.mu.Unlock()

	// Wait for the checkHealthCheckTimeout Go routine and each Go
	// routine per tablet.
	hc.wg.Wait()

	return nil
}

// TabletToMapKey creates a key to the map from tablet's host and ports.
// It should only be used in discovery and related module.
func TabletToMapKey(tablet *topodatapb.Tablet) string {
	parts := make([]string, 0, 1)
	for name, port := range tablet.PortMap {
		parts = append(parts, netutil.JoinHostPort(name, port))
	}
	sort.Strings(parts)
	parts = append([]string{tablet.Hostname}, parts...)
	return strings.Join(parts, ",")
}
