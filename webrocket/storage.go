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

import (
	"path"
	"os"
	"strings"
	kc "../gocabinet"
)

const (
	storageKeyDelim = "|"  // Storage keys delimiter.
	chanKeyPrefix   = "ch" // Channel entries prefix
	vhostKeyPrefix  = "v"  // Vhost entries prefix
)

type vhostStat struct {
	Name        string
	AccessToken string
}

type channelStat struct {
	Name string
}

type storage struct {
	db  *kc.KCDB
	dir string
}

func newStorage(dir string) (s *storage, err error) {	
	err = os.MkdirAll(dir, 0744)
	if err != nil {
		return
	}
	dbfile := path.Join(dir, "/webrocket.kch")
	db := kc.New()
	err = db.Open(dbfile, kc.KCOREADER | kc.KCOWRITER | kc.KCOCREATE)
	if err != nil {
		return
	}
	s = &storage{dir: dir, db: db}
	return
}

func (s *storage) Vhosts() (vhosts []*vhostStat, err error) {
	// find "v|*"
	res, err := s.db.MatchPrefix(string(s.compKey(vhostKeyPrefix, "")), 1024)
	if err != nil {
		return nil, err
	}
	vhosts = make([]*vhostStat, len(res))
	i := 0
	for _, key := range res {
		// get "v|{matching_name}"
		val, err := s.db.Get(key)
		if err != nil {
			continue
		}
		parts := strings.Split(string(val), ";")
		if len(parts) != 2 {
			continue
		}
		vhosts[i] = &vhostStat{Name: parts[0], AccessToken: parts[1]}
		i += 1
	}
	return vhosts[:i], nil
}

func (s *storage) AddVhost(vhost, token string) error {
	// set "v|vhost_name", "vhost_name;access_token"
	return s.db.Set(s.compKey(vhostKeyPrefix, vhost), []byte(vhost + ";" + token))
}

func (s *storage) DeleteVhost(vhost string) (err error) {
	err = s.db.BeginTran(true)
	if err == nil {
		// del "v|vhost_name"
		s.db.Remove(s.compKey(vhostKeyPrefix, vhost))
		// find "ch|vhost_name|*"
		channels, err := s.db.MatchPrefix(string(s.compKey(chanKeyPrefix, vhost, "")), 1024)
		if err != nil {
			channels = [][]byte{}
		}
		// del "ch|vhost_name|{matching_channel}"
		for _, channel := range channels {
			s.db.Remove(channel)
		}
		err = s.db.EndTran(true)
	}
	return err
}

func (s *storage) Channels(vhost string) (channels []*channelStat, err error) {
	// find "ch|vhost_name|*"
	res, err := s.db.MatchPrefix(string(s.compKey(chanKeyPrefix, vhost, "")), 1024)
	if err != nil {
		return nil, err
	}
	channels = make([]*channelStat, len(res))
	i := 0
	for _, key := range res {
		channel, err := s.db.Get(key)
		if err != nil {
			continue
		}
		channels[i] = &channelStat{Name: string(channel)}
		i += 1
	}
	return channels[:i], nil
}

func (s *storage) AddChannel(vhost, channel string) error {
	// set "ch|vhost_name|channel_name", "channel_name"
	return s.db.Set(s.compKey(chanKeyPrefix, vhost, channel), []byte(channel))
}

func (s *storage) DeleteChannel(vhost, channel string) error {
	// del "ch|vhost_name/channel_name"
	return s.db.Remove(s.compKey(chanKeyPrefix, vhost, channel))
}

func (s *storage) Clear() error {
	return s.db.Clear()
}

func (s *storage) Save() error {
	return s.db.Sync(true)
}

func (s *storage) compKey(parts ...string) []byte {
	return []byte(strings.Join(parts, storageKeyDelim))
}