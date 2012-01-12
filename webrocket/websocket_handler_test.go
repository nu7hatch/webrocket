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

func TestWebsocketHandlerStartingAndStopping(t *testing.T) {
	ctx := NewContext()
	v, _ := newVhost(ctx, "/foo")
	h := newWebsocketHandler(v)
	if !h.IsAlive() {
		t.Errorf("Expected websocket handler state set to running by default")
	}
	h.Kill()
	if h.IsAlive() {
		t.Errorf("Expected websocket handler state to change to not running")
	}
}
