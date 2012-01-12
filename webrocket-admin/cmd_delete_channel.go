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
	"net/url"
)

func cmdDeleteChannel(vhostPath, chanName string) {
	if vhostPath == "" || chanName == "" {
		usage()
		return
	}
	s.Start("Deleting a channel")
	cli := &http.Client{}
	form := make(url.Values)
	form.Set("vhost", vhostPath)
	form.Set("name", chanName)
	req, _ := http.NewRequest("DELETE", urlFor("/channel?%s", form.Encode()), nil)
	req.Header.Set("X-WebRocket-Cookie", Cookie)
	res, err := cli.Do(req)
	if err != nil {
		s.Fail("couldn't delete a channel (is server running?)", true)
	}
	_, err = decodeResponse(res, "")
	if err != nil {
		s.Fail(err.Error(), true)
	}
	s.Ok()
}
