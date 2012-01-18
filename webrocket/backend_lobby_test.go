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
)

func newTestBackendWorker() *BackendWorker {
	id, _ := uuid.NewV4()
	return &BackendWorker{id: id.String()}
}

func TestNewBackendLobby(t *testing.T) {
	bl := newBackendLobby()
	if !bl.IsAlive() {
		t.Errorf("Expected lobby to be alive")
	}
}

func TestBackendLobbyAddWorker(t *testing.T) {
	bl := newBackendLobby()
	bl.addWorker(newTestBackendWorker())
	if len(bl.workers) != 1 {
		t.Errorf("Expected to add worker")
	}
	if bl.robin.Len() != 1 {
		t.Errorf("Expected to register worker in load ballancer")
	}
}

func TestBackendLobbyDeleteWorker(t *testing.T) {
	bl := newBackendLobby()
	a := newTestBackendWorker()
	bl.addWorker(a)
	bl.deleteWorker(a)
	if len(bl.workers) != 0 {
		t.Errorf("Expected to delete worker")
	}
	if bl.robin.Len() != 1 {
		t.Errorf("Expected keep deleted worker in load ballancer")
	}
}

func TestBackendLobbyGetWorkerById(t *testing.T) {
	bl := newBackendLobby()
	a := newTestBackendWorker()
	bl.addWorker(a)
	al, ok := bl.getWorkerById(string(a.id))
	if !ok || al == nil || al.Id() != a.Id() {
		t.Errorf("Expected to get worker by id")
	}
	al, ok = bl.getWorkerById("invalid")
	if ok {
		t.Errorf("Expected to not find worker with invalid id")
	}
}

func TestBackendLobbyLoadBallancer(t *testing.T) {
	bl := newBackendLobby()
	bl.addWorker(newTestBackendWorker())
	bl.addWorker(newTestBackendWorker())
	var lastWorker, currentWorker *BackendWorker
	for i := 0; i < 5; i += 1 {
		currentWorker = bl.getAvailableWorker()
		if currentWorker == nil {
			t.Errorf("Expected to pick valid worker")
			continue
		}
		if lastWorker != nil {
			if currentWorker.Id() == lastWorker.Id() {
				t.Errorf("Expected to pick different worker then previous one")
			}
		}
		lastWorker = currentWorker
	}
	bl.deleteWorker(lastWorker)
	lastWorker, currentWorker = nil, nil
	for i := 0; i < 3; i += 1 {
		currentWorker = bl.getAvailableWorker()
		if currentWorker == nil {
			t.Errorf("Expected to pick valid worker")
			continue
		}
		if lastWorker != nil {
			if currentWorker.Id() != lastWorker.Id() {
				t.Errorf("Expected to pick the same worker while there's only one")
			}
		}
		lastWorker = currentWorker
	}
	if bl.robin.Len() != 1 {
		t.Errorf("Expected to remove deleted worker from rign while load ballancing")
	}
}

func TestBackendLobbyKill(t *testing.T) {
	bl := newBackendLobby()
	bl.Kill()
	if bl.IsAlive() {
		t.Errorf("Expected lobby to be killed")
	}
}
