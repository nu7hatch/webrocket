// Copyright (C) 2011 by Krzysztof Kowalik <chris@nu7hat.ch>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

func urlFor(path string, params ...interface{}) string {
	return "http://" + Addr + fmt.Sprintf(path, params...)
}

func decodeResponse(r *http.Response, key string) (data interface{}, err error) {
	var dec *json.Decoder
	var res map[string]interface{}
	dec = json.NewDecoder(r.Body)
	err = dec.Decode(&res)
	if err != nil {
		if err != io.EOF {
			return
		} else {
			data = make(map[string]interface{})
		}
	}
	if data, ok := res[key]; ok && r.StatusCode < 400 {
		return data, nil
	} else if data, ok := res["error"]; ok && r.StatusCode > 400 {
		if msg, ok := data.(string); ok {
			err := errors.New(msg)
			return nil, err
		}
		return data, nil
	} else if key == "" && r.StatusCode < 400 {
		return nil, nil
	}
	err = errors.New("couldn't read the data")
	return
}

func followRedirects(r *http.Response) (*http.Response, error) {
	if location, err := r.Location(); err == nil && location != nil {
		cli := &http.Client{}
		req, _ := http.NewRequest("GET", location.String(), nil)
		req.Header.Set("X-WebRocket-Cookie", Cookie)
		return cli.Do(req)
	}
	return r, nil
}
