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
)

func cmdListVhosts() {
	var data interface{}
	s.Start("Loading vhosts")
	cli := &http.Client{}
	req, _ := http.NewRequest("GET", urlFor("/vhosts"), nil)
	req.Header.Set("X-WebRocket-Cookie", Cookie)
	res, err := cli.Do(req)
	if err != nil {
		s.Fail("couldn't list vhosts (is server running?)", true)
	}
	data, err = decodeResponse(res, "vhosts")
	if err != nil {
		s.Fail(err.Error(), true)
	}
	if vhosts, ok := data.([]interface{}); ok {
		s.Ok()
		fmt.Printf("---\n")
		for _, entry := range vhosts {
			vhost, ok := entry.(map[string]interface{})
			if !ok {
				continue
			}
			path, _ := vhost["path"].(string)
			if path == "" {
				continue
			}
			fmt.Printf("%s\n", path)
		}
		return
	}
}
