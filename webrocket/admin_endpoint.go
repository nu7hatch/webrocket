// Copyright (C) 2011 by Krzysztof Kowalik <chris@nu7hat.ch>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package webrocket

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
)

// Helper for logging admin handler's statuses.
func adminStatusLog(a *AdminEndpoint, msg string) {
	a.log.Printf("admin: %s", msg)
}

// Helper for logging protocol errors and sending them to
// the client.
func adminError(a *AdminEndpoint, error string) {
	adminStatusLog(a, "Error; "+error)
}

type AdminEndpoint struct {
	*http.Server
	ctx   *Context
	alive bool
	log   *log.Logger
	mtx   sync.Mutex
}

func (ctx *Context) NewAdminEndpoint(addr string) Endpoint {
	e := &AdminEndpoint{
		ctx:    ctx,
		log:    ctx.log,
		Server: &http.Server{Addr: addr},
	}
	e.Server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		e.route(w, r)
	})
	ctx.admin = e
	return e
}

func (a *AdminEndpoint) Addr() string {
	return a.Server.Addr
}

// Returns true if this endpoint is activated.
func (a *AdminEndpoint) IsAlive() bool {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	return a.alive
}

// Kill stops execution of this endpoint.
func (a *AdminEndpoint) Kill() {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	a.alive = false
}

// Extendended http.Server.ListenAndServe function.
func (a *AdminEndpoint) ListenAndServe() error {
	addr := a.Server.Addr
	if addr == "" {
		addr = ":http"
	}
	l, e := net.Listen("tcp", addr)
	if e != nil {
		return e
	}
	a.alive = true
	return a.Server.Serve(l)
}

// Extendended http.Server.ListenAndServeTLS function.
func (a *AdminEndpoint) ListenAndServeTLS(certFile, certKey string) error {
	addr := a.Server.Addr
	if addr == "" {
		addr = ":https"
	}
	config := &tls.Config{
		Rand:       rand.Reader,
		NextProtos: []string{"http/1.1"},
	}
	var err error
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0], err = tls.LoadX509KeyPair(certFile, certKey)
	if err != nil {
		return err
	}
	conn, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	tlsListener := tls.NewListener(conn, config)
	a.alive = true
	return a.Server.Serve(tlsListener)
}

func (a *AdminEndpoint) route(w http.ResponseWriter, r *http.Request) {
	cookie := r.Header.Get("X-WebRocket-Cookie")
	if cookie != a.ctx.Cookie() {
		a.error(w, http.StatusForbidden, errors.New("access denied"))
		return
	}
	w.Header().Set("X-WebRocket-Cookie", cookie)
	w.Header().Set("Content-Type", "application/json")
	r.ParseForm()

	switch r.URL.Path {
	case "/vhost":
		switch r.Method {
		case "GET":
			a.getVhost(w, r)
		case "DELETE":
			a.deleteVhost(w, r)
		}
	case "/vhost/token":
		if r.Method == "PUT" {
			a.regenerateVhostTokenHandler(w, r)
		}
	case "/vhosts":
		switch r.Method {
		case "GET":
			a.allVhosts(w, r)
		case "POST":
			a.addVhost(w, r)
		case "DELETE":
			a.clearVhosts(w, r)
		}
	case "/channel":
		switch r.Method {
		case "GET":
			a.getChannel(w, r)
		case "DELETE":
			a.deleteChannel(w, r)
		}
	case "/channels":
		switch r.Method {
		case "GET":
			a.allChannels(w, r)
		case "POST":
			a.addChannel(w, r)
		case "DELETE":
			a.clearChannels(w, r)
		}
	case "/workers":
		switch r.Method {
		case "GET":
			a.allWorkers(w, r)
		}
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (a *AdminEndpoint) error(w http.ResponseWriter, status int, err error) {
	if err != nil {
		adminError(a, err.Error())
	}
	data, err := json.Marshal(map[string]string{"error": err.Error()})
	if err != nil {
		adminError(a, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(status)
	w.Write(data)
}

func (a *AdminEndpoint) addVhost(w http.ResponseWriter, r *http.Request) {
	path := r.Form.Get("path")
	vhost, err := a.ctx.AddVhost(path)
	if err != nil {
		a.error(w, 418, /*UnprocessibleEntity*/ err)
		return
	}
	adminStatusLog(a, fmt.Sprintf("Created vhost `%s`", vhost.path))
	w.Header().Set("Location", fmt.Sprintf("/vhost?path=%s", vhost.path))
	w.WriteHeader(http.StatusFound)
}

func (a *AdminEndpoint) deleteVhost(w http.ResponseWriter, r *http.Request) {
	path := r.Form.Get("path")
	err := a.ctx.DeleteVhost(path)
	if err != nil {
		a.error(w, http.StatusNotFound, err)
		return
	}
	adminStatusLog(a, fmt.Sprintf("Deleted vhost `%s`", path))
	w.WriteHeader(http.StatusAccepted)
}

func (a *AdminEndpoint) getVhost(w http.ResponseWriter, r *http.Request) {
	path := r.Form.Get("path")
	vhost, err := a.ctx.Vhost(path)
	if err != nil {
		a.error(w, http.StatusNotFound, err)
		return
	}
	vhostChannelsData := []interface{}{}
	for _, channel := range vhost.Channels() {
		vhostChannelsData = append(vhostChannelsData, map[string]interface{}{
			"self":  fmt.Sprintf("/channel?vhost=%s&name=%s", vhost.path, channel.name),
			"vhost": fmt.Sprintf("/vhost?path=%s", vhost.path),
			"name":  channel.name,
			"subscribers": map[string]interface{}{
				"self": fmt.Sprintf("/subscribers?vhost=%s&channel=%s", vhost.path, channel.name),
				"size": int32(len(channel.subscribers)),
			},
		})
	}
	vhostData := map[string]interface{}{
		"vhost": map[string]interface{}{
			"self":        fmt.Sprintf("/vhost?path=%s", vhost.path),
			"path":        vhost.path,
			"accessToken": vhost.accessToken,
			"channels":    vhostChannelsData,
		},
	}
	data, err := json.Marshal(vhostData)
	if err != nil {
		a.error(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (a *AdminEndpoint) allVhosts(w http.ResponseWriter, r *http.Request) {
	vhosts := []interface{}{}
	for _, vhost := range a.ctx.Vhosts() {
		vhosts = append(vhosts, map[string]interface{}{
			"self":        fmt.Sprintf("/vhost?path=%s", vhost.path),
			"path":        vhost.path,
			"accessToken": vhost.accessToken,
			"channels": map[string]interface{}{
				"self": fmt.Sprintf("/channels?vhost=%s", vhost.path),
				"size": len(vhost.channels),
			},
		})
	}
	vhostsData := map[string]interface{}{"vhosts": vhosts}
	data, err := json.Marshal(vhostsData)
	if err != nil {
		a.error(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (a *AdminEndpoint) clearVhosts(w http.ResponseWriter, r *http.Request) {
	for path := range a.ctx.vhosts {
		a.ctx.DeleteVhost(path)
	}
	adminStatusLog(a, "All vhosts deleted")
	w.WriteHeader(http.StatusAccepted)
}

func (a *AdminEndpoint) regenerateVhostTokenHandler(w http.ResponseWriter, r *http.Request) {
	path := r.Form.Get("path")
	vhost, err := a.ctx.Vhost(path)
	if err != nil {
		a.error(w, http.StatusNotFound, err)
		return
	}
	vhost.GenerateAccessToken()
	adminStatusLog(a, fmt.Sprintf("Regenerated token for `%s` vhost", vhost.path))
	w.Header().Set("Location", fmt.Sprintf("/vhost?path=%s", vhost.path))
	w.WriteHeader(http.StatusFound)
}

func (a *AdminEndpoint) addChannel(w http.ResponseWriter, r *http.Request) {
	vhostPath := r.Form.Get("vhost")
	vhost, err := a.ctx.Vhost(vhostPath)
	if err != nil {
		a.error(w, http.StatusNotFound, err)
		return
	}
	chanName := r.Form.Get("name")
	chanType := channelTypeFromName(chanName)
	channel, err := vhost.OpenChannel(chanName, chanType)
	if err != nil {
		a.error(w, 418, /*UnprocessibleEntity*/ err)
		return
	}
	adminStatusLog(a, fmt.Sprintf("Created channel `%s` under vhost `%s`", channel.name, vhost.path))
	w.Header().Set("Location", fmt.Sprintf("/channel?vhost=%s&name=%s", vhost.path, channel.name))
	w.WriteHeader(http.StatusFound)
}

func (a *AdminEndpoint) deleteChannel(w http.ResponseWriter, r *http.Request) {
	vhostPath := r.Form.Get("vhost")
	vhost, err := a.ctx.Vhost(vhostPath)
	if err != nil {
		a.error(w, http.StatusNotFound, err)
		return
	}
	chanName := r.Form.Get("name")
	ok := vhost.DeleteChannel(chanName) // FIXME: why DeleteVhost returns error and DeleteChannel bool? o_O
	if !ok {
		a.error(w, http.StatusNotFound, errors.New("channel doesn exist"))
		return
	}
	adminStatusLog(a, fmt.Sprintf("Closed channel `%s` under vhost `%s`", chanName, vhost.path))
	w.WriteHeader(http.StatusAccepted)
}

func (a *AdminEndpoint) getChannel(w http.ResponseWriter, r *http.Request) {
	vhostPath := r.Form.Get("vhost")
	vhost, err := a.ctx.Vhost(vhostPath)
	if err != nil {
		a.error(w, http.StatusNotFound, err)
		return
	}
	chanName := r.Form.Get("name")
	channel, err := vhost.Channel(chanName)
	if err != nil {
		a.error(w, http.StatusNotFound, err)
		return
	}
	channelSubscribersData := []interface{}{}
	for _, subsber := range channel.Subscribers() {
		channelSubscribersData = append(channelSubscribersData, map[string]interface{}{
			"self": fmt.Sprintf("/subscriber?vhost=%s&channel=%s&sid=%s", vhost.path, channel.name, subsber.Id()),
			"sid":  subsber.Id(),
		})
	}
	channelData := map[string]interface{}{
		"channel": map[string]interface{}{
			"self":        fmt.Sprintf("/channel?vhost=%s&name=%s", vhost.path, channel.name),
			"vhost":       fmt.Sprintf("/vhost?path=%s", vhost.path),
			"name":        channel.name,
			"subscribers": channelSubscribersData,
		},
	}
	data, err := json.Marshal(channelData)
	if err != nil {
		a.error(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (a *AdminEndpoint) allChannels(w http.ResponseWriter, r *http.Request) {
	vhostPath := r.Form.Get("vhost")
	vhost, err := a.ctx.Vhost(vhostPath)
	if err != nil {
		a.error(w, http.StatusNotFound, err)
		return
	}
	channels := []interface{}{}
	for _, channel := range vhost.Channels() {
		channels = append(channels, map[string]interface{}{
			"self":  fmt.Sprintf("/channel?vhost=%s&name=%s", vhost.path, channel.name),
			"vhost": fmt.Sprintf("/vhost?path=%s", vhost.path),
			"name":  channel.name,
			"subscribers": map[string]interface{}{
				"self": fmt.Sprintf("/subscribers?vhost=%s&channel=%s", vhost.path, channel.name),
				"size": int32(len(channel.subscribers)),
			},
		})
	}
	channelsData := map[string]interface{}{"channels": channels}
	data, err := json.Marshal(channelsData)
	if err != nil {
		a.error(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (a *AdminEndpoint) clearChannels(w http.ResponseWriter, r *http.Request) {
	vhostPath := r.Form.Get("vhost")
	vhost, err := a.ctx.Vhost(vhostPath)
	if err != nil {
		a.error(w, http.StatusNotFound, err)
		return
	}
	for name := range vhost.channels {
		vhost.DeleteChannel(name)
	}
	adminStatusLog(a, fmt.Sprintf("All channels deleted from `%s` vhost", vhost.path))
	w.WriteHeader(http.StatusAccepted)
}

func (a *AdminEndpoint) allWorkers(w http.ResponseWriter, r *http.Request) {
	vhostPath := r.Form.Get("vhost")
	vhost, err := a.ctx.Vhost(vhostPath)
	if err != nil {
		a.error(w, http.StatusNotFound, err)
		return
	}
	lobby := a.ctx.backend.lobbys.Match(vhost.path)
	if lobby == nil {
		a.error(w, http.StatusInternalServerError, errors.New("something went wrong"))
		return
	}
	workers := []interface{}{}
	for _, worker := range lobby.workers {
		if worker.IsAlive() {
			workers = append(workers, map[string]interface{}{
				"self":  fmt.Sprintf("/worker?vhost=%s&id=%s", vhost.path, worker.id),
				"vhost": fmt.Sprintf("/vhost?path=%s", vhost.path),
				"id":    string(worker.id),
			})
		}
	}
	workersData := map[string]interface{}{"workers": workers}
	data, err := json.Marshal(workersData)
	if err != nil {
		a.error(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
