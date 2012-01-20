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
	"fmt"
	"net/http"
	"net/url"
)

func cmdAddVhost(vhostPath string) {
	var data interface{}
	if vhostPath == "" {
		usage()
		return
	}
	s.Start("Adding a vhost")
	cli := &http.Client{}
	form := make(url.Values)
	form.Set("path", vhostPath)
	req, _ := http.NewRequest("POST", urlFor("/vhosts?%s", form.Encode()), nil)
	req.Header.Set("X-WebRocket-Cookie", Cookie)
	res, err := cli.Do(req)
	if err != nil {
		goto error
	}
	res, err = followRedirects(res)
	if err != nil {
		goto error
	}
	data, err = decodeResponse(res, "vhost")
	if err != nil {
		s.Fail(err.Error(), true)
	}
	if vhost, ok := data.(map[string]interface{}); ok {
		token, ok := vhost["accessToken"].(string)
		if !ok || token != "" {
			s.Ok()
			fmt.Printf("---\n")
			fmt.Printf("Access token for this vhost: %s\n", token)
			return
		}
	}
error:
	s.Fail("couldn't create a vhost (is server running?)", true)
}
