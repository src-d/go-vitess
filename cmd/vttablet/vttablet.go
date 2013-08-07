// Copyright 2012, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// vt tablet server: Serves queries and performs housekeeping jobs.
package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/youtube/vitess/go/jscfg"
	"github.com/youtube/vitess/go/proc"
	"github.com/youtube/vitess/go/relog"
	rpc "github.com/youtube/vitess/go/rpcplus"
	"github.com/youtube/vitess/go/rpcwrap/auth"
	"github.com/youtube/vitess/go/rpcwrap/bsonrpc"
	"github.com/youtube/vitess/go/rpcwrap/jsonrpc"
	_ "github.com/youtube/vitess/go/snitch"
	"github.com/youtube/vitess/go/vt/dbconfigs"
	"github.com/youtube/vitess/go/vt/mysqlctl"
	"github.com/youtube/vitess/go/vt/servenv"
	ts "github.com/youtube/vitess/go/vt/tabletserver"
	"github.com/youtube/vitess/go/vt/topo"
	"github.com/youtube/vitess/go/vt/vttablet"
)

var (
	port          = flag.Int("port", 6509, "port for the server")
	tabletPath    = flag.String("tablet-path", "", "tablet alias or path to zk node representing the tablet")
	qsConfigFile  = flag.String("queryserver-config-file", "", "config file name for the query service")
	mycnfFile     = flag.String("mycnf-file", "", "my.cnf file")
	authConfig    = flag.String("auth-credentials", "", "name of file containing auth credentials")
	queryLog      = flag.String("debug-querylog-file", "", "for testing: log all queries to this file")
	customrules   = flag.String("customrules", "", "custom query rules file")
	overridesFile = flag.String("schema-override", "", "schema overrides file")

	securePort = flag.Int("secure-port", 0, "port for the secure server")
	cert       = flag.String("cert", "", "cert file")
	key        = flag.String("key", "", "key file")
	caCert     = flag.String("ca-cert", "", "ca-cert file")
)

// Default values for the config
//
// The value for StreamBufferSize was chosen after trying out a few of
// them. Too small buffers force too many packets to be sent. Too big
// buffers force the clients to read them in multiple chunks and make
// memory copies.  so with the encoding overhead, this seems to work
// great.  (the overhead makes the final packets on the wire about
// twice bigger than this).
var qsConfig = ts.Config{
	PoolSize:           16,
	StreamPoolSize:     750,
	TransactionCap:     20,
	TransactionTimeout: 30,
	MaxResultSize:      10000,
	QueryCacheSize:     5000,
	SchemaReloadTime:   30 * 60,
	QueryTimeout:       0,
	IdleTimeout:        30 * 60,
	StreamBufferSize:   32 * 1024,
	RowCache:           nil,
}

// tabletParamToTabletAlias takes either an old style ZK tablet path or a
// new style tablet alias as a string, and returns a TabletAlias.
func tabletParamToTabletAlias(param string) topo.TabletAlias {
	if param[0] == '/' {
		// old zookeeper path, convert to new-style string tablet alias
		zkPathParts := strings.Split(param, "/")
		if len(zkPathParts) != 6 || zkPathParts[0] != "" || zkPathParts[1] != "zk" || zkPathParts[3] != "vt" || zkPathParts[4] != "tablets" {
			relog.Fatal("Invalid tablet path: %v", param)
		}
		param = zkPathParts[2] + "-" + zkPathParts[5]
	}
	result, err := topo.ParseTabletAliasString(param)
	if err != nil {
		relog.Fatal("Invalid tablet alias %v: %v", param, err)
	}
	return result
}

func main() {
	dbConfigsFile, dbCredentialsFile := dbconfigs.RegisterCommonFlags()
	flag.Parse()

	if err := servenv.Init("vttablet"); err != nil {
		relog.Fatal("Error in servenv.Init: %s", err)
	}

	tabletAlias := tabletParamToTabletAlias(*tabletPath)

	mycnf := readMycnf(tabletAlias.Uid)
	dbcfgs, err := dbconfigs.Init(mycnf.SocketFile, *dbConfigsFile, *dbCredentialsFile)
	if err != nil {
		relog.Warning("%s", err)
	}

	initQueryService(dbcfgs)
	mysqlctl.RegisterUpdateStreamService(mycnf)
	ts.RegisterCacheInvalidator()                                                                                                                          // depends on both query and updateStream
	err = vttablet.InitAgent(tabletAlias, dbcfgs, mycnf, *dbConfigsFile, *dbCredentialsFile, *port, *securePort, *mycnfFile, *customrules, *overridesFile) // depends on both query and updateStream
	if err != nil {
		relog.Fatal("%s", err)
	}

	rpc.HandleHTTP()

	// NOTE(szopa): Changing credentials requires a server
	// restart.
	if *authConfig != "" {
		if err := auth.LoadCredentials(*authConfig); err != nil {
			relog.Error("could not load authentication credentials, not starting rpc servers: %v", err)
		}
		serveAuthRPC()
	}

	serveRPC()

	vttablet.HttpHandleSnapshots(mycnf, tabletAlias.Uid)

	l, err := proc.Listen(fmt.Sprintf("%v", *port))
	if err != nil {
		relog.Fatal("%s", err)
	}
	go http.Serve(l, nil)

	var secureListener net.Listener
	if *securePort != 0 {
		relog.Info("listening on secure port %v", *securePort)
		vttablet.SecureServe(fmt.Sprintf(":%d", *securePort), *cert, *key, *caCert)
	}

	relog.Info("started vttablet %v", *port)
	s := proc.Wait()
	if secureListener != nil {
		secureListener.Close()
	}

	// A SIGUSR1 means that we're restarting
	if s == syscall.SIGUSR1 {
		// Give some time for the other process
		// to pick up the listeners
		relog.Info("Exiting on SIGUSR1")
		time.Sleep(5 * time.Millisecond)
		ts.DisallowQueries(true)
	} else {
		relog.Info("Exiting on SIGTERM")
		ts.DisallowQueries(false)
	}
	mysqlctl.DisableUpdateStreamService()
	topo.CloseServers()
	vttablet.CloseAgent()
	relog.Info("done")
}

func serveAuthRPC() {
	bsonrpc.ServeAuthRPC()
	jsonrpc.ServeAuthRPC()
}

func serveRPC() {
	jsonrpc.ServeHTTP()
	jsonrpc.ServeRPC()
	bsonrpc.ServeHTTP()
	bsonrpc.ServeRPC()
}

func readMycnf(tabletId uint32) *mysqlctl.Mycnf {
	if *mycnfFile == "" {
		*mycnfFile = mysqlctl.MycnfFile(tabletId)
	}
	mycnf, mycnfErr := mysqlctl.ReadMycnf(*mycnfFile)
	if mycnfErr != nil {
		relog.Fatal("mycnf read failed: %v", mycnfErr)
	}
	return mycnf
}

func initQueryService(dbcfgs dbconfigs.DBConfigs) {
	ts.SqlQueryLogger.ServeLogs("/debug/querylog")
	ts.TxLogger.ServeLogs("/debug/txlog")

	if err := jscfg.ReadJson(*qsConfigFile, &qsConfig); err != nil {
		relog.Warning("%s", err)
	}
	ts.RegisterQueryService(qsConfig)
	if *queryLog != "" {
		if f, err := os.OpenFile(*queryLog, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644); err == nil {
			ts.QueryLogger = relog.New(f, "", relog.DEBUG)
		} else {
			relog.Fatal("Error opening file %v: %v", *queryLog, err)
		}
	}
}
