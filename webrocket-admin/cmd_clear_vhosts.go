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

package main

import (
	"net/http"
)

func cmdClearVhosts() {
	s.Start("Deleting all vhosts")
	cli := &http.Client{}
	req, err := http.NewRequest("DELETE", urlFor("/vhosts"), nil)
	req.Header.Set("X-WebRocket-Cookie", Cookie)
	res, err := cli.Do(req)
	if err != nil {
		s.Fail("couldn't delete all vhosts (is server running?)", true)
	}
	_, err = decodeResponse(res, "")
	if err != nil {
		s.Fail(err.Error(), true)
	}
	s.Ok()
}
