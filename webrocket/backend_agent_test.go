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
	uuid "../uuid"
	"testing"
	"time"
)

func newTestBackendAgent() *BackendAgent {
	ctx := NewContext()
	b := ctx.NewBackendEndpoint(":9772")
	v, _ := newVhost(ctx, "/foo")
	uuid, _ := uuid.NewV4()
	a := newBackendAgent(b.(*BackendEndpoint), v, uuid[:])
	return a
}

func TestNewBackendAgent(t *testing.T) {
	a := newTestBackendAgent()
	if !a.IsAlive() {
		t.Errorf("Expected new agent to be alive")
	}
	if string(a.id) == "" {
		t.Errorf("Expected new agent to have proper id")
	}
}

func TestNewBackendAgentIsAlive(t *testing.T) {
	a := newTestBackendAgent()
	a.expiry = time.Now()
	if a.IsAlive() {
		t.Errorf("Expected agent to be not alive")
	}
	a.updateExpiration()
	if !a.IsAlive() {
		t.Errorf("Expected agent to be alive")
	}
	a.Kill()
	if a.IsAlive() {
		t.Errorf("Expected agent to be killed")
	}
}