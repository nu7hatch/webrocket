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

func TestNewWebsocketEndpoint(t *testing.T) {
	ctx := NewContext()
	e := newWebsocketEndpoint(ctx, "127.0.0.1:9000")
	if e.Addr() != "127.0.0.1:9000" {
		t.Errorf("Expected to bing websockets endpoint to 127.0.0.1:9000")
	}
}
