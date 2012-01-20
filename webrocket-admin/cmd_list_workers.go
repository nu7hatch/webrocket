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

func cmdListWorkers(vhostPath string) {
	var data interface{}
	if vhostPath == "" {
		usage()
		return
	}
	s.Start("Loading workers")
	cli := &http.Client{}
	form := make(url.Values)
	form.Set("vhost", vhostPath)
	req, _ := http.NewRequest("GET", urlFor("/workers?%s", form.Encode()), nil)
	req.Header.Set("X-WebRocket-Cookie", Cookie)
	res, err := cli.Do(req)
	if err != nil {
		s.Fail("couldn't list workers (is server running?)", true)
	}
	data, err = decodeResponse(res, "workers")
	if err != nil {
		s.Fail(err.Error(), true)
	}
	if channels, ok := data.([]interface{}); ok {
		s.Ok()
		fmt.Printf("---\n")
		for _, entry := range channels {
			channel, ok := entry.(map[string]interface{})
			if !ok {
				continue
			}
			name, _ := channel["id"].(string)
			if name == "" {
				continue
			}
			fmt.Printf("%s\n", name)
		}
		return
	}
}
