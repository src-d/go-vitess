package tabletserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"text/template"

	"github.com/youtube/vitess/go/acl"
)

var (
	streamqueryzHeader = []byte(`<thead>
		<tr>
			<th>Query</th>
			<th>RemoteAddr</th>
			<th>Username</th>
			<th>Duration</th>
			<th>Start</th>
			<th>SessionID</th>
			<th>TransactionID</th>
			<th>ConnectionID</th>
			<th>Current State</th>
			<th>Terminate</th>
		</tr>
        </thead>
	`)
	streamqueryzTmpl = template.Must(template.New("example").Parse(`
		<tr> 
			<td>{{.Query}}</td>
			<td>{{.RemoteAddr}}</td>
			<td>{{.Username}}</td>
			<td>{{.Duration}}</td>
			<td>{{.Start}}</td>
			<td>{{.SessionID}}</td>
			<td>{{.TransactionID}}</td>
			<td>{{.ConnID}}</td>
			<td>{{.State}}</td>
			<td>{{if .ShowTerminateLink }}<a href='/streamqueryz/terminate?connID={{.ConnID}}'>Terminate</a>{{end}}</td>
		</tr>
	`))
)

func init() {
	http.HandleFunc("/streamqueryz", streamqueryzHandler)
	http.HandleFunc("/streamqueryz/terminate", streamqueryzTerminateHandler)
}

func streamqueryzHandler(w http.ResponseWriter, r *http.Request) {
	if err := acl.CheckAccessHTTP(r, acl.DEBUGGING); err != nil {
		acl.SendError(w, err)
		return
	}
	rows := SqlQueryRpcService.qe.streamQList.GetQueryzRows()
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("cannot parse form: %s", err), http.StatusInternalServerError)
		return
	}
	format := r.FormValue("format")
	if format == "json" {
		js, err := json.Marshal(rows)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
		return
	}
	startHTMLTable(w)
	defer endHTMLTable(w)
	w.Write(streamqueryzHeader)
	for i := range rows {
		streamqueryzTmpl.Execute(w, rows[i])
	}
}

func streamqueryzTerminateHandler(w http.ResponseWriter, r *http.Request) {
	if err := acl.CheckAccessHTTP(r, acl.ADMIN); err != nil {
		acl.SendError(w, err)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("cannot parse form: %s", err), http.StatusInternalServerError)
		return
	}
	connID := r.FormValue("connID")
	c, err := strconv.Atoi(connID)
	if err != nil {
		http.Error(w, "invalid connID", http.StatusInternalServerError)
		return
	}
	qd := SqlQueryRpcService.qe.streamQList.Get(int64(c))
	if qd == nil {
		http.Error(w, fmt.Sprintf("connID %v not found in query list", connID), http.StatusInternalServerError)
		return
	}
	if !qd.Terminate() {
		http.Error(w, fmt.Sprintf("connID %v is not in running state", connID), http.StatusInternalServerError)
		return
	}
	streamqueryzHandler(w, r)
}
