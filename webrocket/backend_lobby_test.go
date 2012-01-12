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

import "testing"

func TestNewBackendLobby(t *testing.T) {
	bl := newBackendLobby()
	if !bl.IsAlive() {
		t.Errorf("Expected lobby to be alive")
	}
}

func TestBackendLobbyAddAgent(t *testing.T) {
	bl := newBackendLobby()
	bl.addAgent(newTestBackendAgent())
	if len(bl.agents) != 1 {
		t.Errorf("Expected to add agent")
	}
	if bl.robin.Len() != 1 {
		t.Errorf("Expected to register agent in load ballancer")
	}
}

func TestBackendLobbyDeleteAgent(t *testing.T) {
	bl := newBackendLobby()
	a := newTestBackendAgent()
	bl.addAgent(a)
	bl.deleteAgent(a)
	if len(bl.agents) != 0 {
		t.Errorf("Expected to delete agent")
	}
	if bl.robin.Len() != 1 {
		t.Errorf("Expected keep deleted agent in load ballancer")
	}
}

func TestBackendLobbyGetAgentById(t *testing.T) {
	bl := newBackendLobby()
	a := newTestBackendAgent()
	bl.addAgent(a)
	al, ok := bl.getAgentById(string(a.id))
	if !ok || al == nil || al.Id() != a.Id() {
		t.Errorf("Expected to get agent by id")
	}
	al, ok = bl.getAgentById("invalid")
	if ok {
		t.Errorf("Expected to not find agent with invalid id")
	}
}

func TestBackendLobbyLoadBallance(t *testing.T) {
	bl := newBackendLobby()
	bl.addAgent(newTestBackendAgent())
	bl.addAgent(newTestBackendAgent())
	var lastAgent, currentAgent *BackendAgent
	for i := 0; i < 5; i += 1 {
		currentAgent = bl.loadBallance()
		if currentAgent == nil {
			t.Errorf("Expected to pick valid agent")
			continue
		}
		if lastAgent != nil {
			if currentAgent.Id() == lastAgent.Id() {
				t.Errorf("Expected to pick different agent then previous one")
			}
		}
		lastAgent = currentAgent
	}
	bl.deleteAgent(lastAgent)
	lastAgent, currentAgent = nil, nil
	for i := 0; i < 3; i += 1 {
		currentAgent = bl.loadBallance()
		if currentAgent == nil {
			t.Errorf("Expected to pick valid agent")
			continue
		}
		if lastAgent != nil {
			if currentAgent.Id() != lastAgent.Id() {
				t.Errorf("Expected to pick the same agent while there's only one")
			}
		}
		lastAgent = currentAgent
	}
	if bl.robin.Len() != 1 {
		t.Errorf("Expected to remove deleted agent from rign while load ballancing")
	}
}

func TestBackendLobbyKill(t *testing.T) {
	bl := newBackendLobby()
	bl.Kill()
	if bl.IsAlive() {
		t.Errorf("Expected lobby to be killed")
	}
}
