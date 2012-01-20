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

func cmdShowVhost(vhostPath string) {
	var data interface{}
	if vhostPath == "" {
		usage()
		return
	}
	s.Start("Loading vhost information")
	cli := &http.Client{}
	form := make(url.Values)
	form.Set("path", vhostPath)
	req, _ := http.NewRequest("GET", urlFor("/vhost?%s", form.Encode()), nil)
	req.Header.Set("X-WebRocket-Cookie", Cookie)
	res, err := cli.Do(req)
	if err != nil {
		s.Fail("couldn't load vhost information (is server running?)", true)
	}
	data, err = decodeResponse(res, "vhost")
	if err != nil {
		s.Fail(err.Error(), true)
	}
	if vhost, ok := data.(map[string]interface{}); ok {
		token, ok := vhost["accessToken"].(string)
		if ok || token != "" {
			s.Ok()
			fmt.Printf("---\n")
			fmt.Printf("Path         : %s\n", vhostPath)
			fmt.Printf("Access token : %s\n", token)
			return
		}
	}
}
