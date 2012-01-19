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
	kc "../gocabinet"
	"os"
	"path"
	"strconv"
	"strings"
)

const (
	// Storage key parts delimiter.
	storageKeyDelim = "|"
	// Channel entries key prefix
	storageChanKeyPrefix = "ch"
	// Vhost entries key prefix
	storageVhostKeyPrefix = "v"
)

// vhostStat is an internal struct to represent stored information about
// the vhost.
type vhostStat struct {
	// The vhost's name.
	Name string
	// The vhost's access token.
	AccessToken string
}

// channelStat is an internal struct to represent stored information about
// the channel.
type channelStat struct {
	// The channel's name.
	Name string
	// The channel's type.
	Kind ChannelType
}

// storage implements an adapter for the persistence layer. At the moment
// a Kyoto Cabinet database is used to provide data persistency. All storage's
// functions are threadsafe.
type storage struct {
	// Kyoto Cabinet database object.
	db *kc.KCDB
	// Path to storage directory.
	dir string
}

// Internal constructor
// -----------------------------------------------------------------------------

// newStorage creates new persistence under the specified directory.
// At the moment Kyoto Cabinet is used, so it creates a 'webrocket.kch'
// database file there.
//
// dir - A path to the storage location.
//
// Returns configured storage or error if something went wrong
func newStorage(dir string) (s *storage, err error) {
	if err = os.MkdirAll(dir, 0744); err != nil {
		return
	}
	dbfile := path.Join(dir, "/webrocket.kch")
	db := kc.New()
	err = db.Open(dbfile, kc.KCOREADER|kc.KCOWRITER|kc.KCOCREATE)
	if err != nil {
		return
	}
	s = &storage{dir: dir, db: db}
	return
}

// Internal
// -----------------------------------------------------------------------------

// compKey combines given parts into the storage entry key.
//
// parts - A list of the parts to combine.
//
// Examples
//
//     key = s.compKey("foo", "bar")
//     println(string(key))
//     // => "foo|bar"
//
// Returns a key combined from given parts.
func (s *storage) compKey(parts ...string) []byte {
	return []byte(strings.Join(parts, storageKeyDelim))
}

// Exported
// -----------------------------------------------------------------------------

// Vhosts loads databse entries for all the registered vhosts.
//
// Returns list of the vhostStat entries or an error if something went wrong.
func (s *storage) Vhosts() (vhosts []*vhostStat, err error) {
	// find "v|*"
	res, err := s.db.MatchPrefix(string(s.compKey(storageVhostKeyPrefix, "")), 1024)
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

// AddVhost creates a databse entry for the specified vhost.
//
// vhost - The name of the vhost to be created.
// token - The vhost's access token value.
//
// Returns an error if something went wrong.
func (s *storage) AddVhost(vhost, token string) error {
	// set "v|vhost_name", "vhost_name;access_token"
	return s.db.Set(s.compKey(storageVhostKeyPrefix, vhost), []byte(vhost+";"+token))
}

// DeleteVhost removes a database entry for the specified vhost and removes
// all its channels' entries as well.
//
// vhost - The name of the vhost to be deleted.
//
// Returns an error if something went wrong.
func (s *storage) DeleteVhost(vhost string) (err error) {
	err = s.db.BeginTran(true)
	if err == nil {
		// del "v|vhost_name"
		s.db.Remove(s.compKey(storageVhostKeyPrefix, vhost))
		// find "ch|vhost_name|*"
		channels, err := s.db.MatchPrefix(string(s.compKey(storageChanKeyPrefix, vhost, "")), 1024)
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

// Channels loads entries for all the channels registered under the specified vhost.
//
// vhost - The name of the channels' parent vhost.
//
// Returns list of the channelStat entries oran error if something went wrong.
func (s *storage) Channels(vhost string) (channels []*channelStat, err error) {
	// find "ch|vhost_name|*"
	res, err := s.db.MatchPrefix(string(s.compKey(storageChanKeyPrefix, vhost, "")), 1024)
	if err != nil {
		return nil, err
	}
	channels = make([]*channelStat, len(res))
	i := 0
	for _, key := range res {
		val, err := s.db.Get(key)
		if err != nil {
			continue
		}
		parts := strings.Split(string(val), ";")
		if len(parts) != 2 {
			continue
		}
		channelType, _ := strconv.Atoi(string(parts[1]))
		channels[i] = &channelStat{Name: parts[0], Kind: ChannelType(channelType)}
		i += 1
	}
	return channels[:i], nil
}

// AddChannel create a databse entry for the specified channel.
//
// vhost   - The name of the channel's parent vhost.
// channel - The name of the channel to be added.
// kind    - A type of the channel.
//
// Returns an error if something went wrong.
func (s *storage) AddChannel(vhost, channel string, kind ChannelType) error {
	// set "ch|vhost_name|channel_name", "channel_name"
	kindVal := strconv.Itoa(int(kind))
	return s.db.Set(s.compKey(storageChanKeyPrefix, vhost, channel), []byte(channel+";"+kindVal))
}

// DeleteChannel removes a given channel's entry from the database.
//
// vhost   - The name of the channel's parent vhost.
// channel - The name of the channel to be deleted.
//
// Returns an error if something went wrong.
func (s *storage) DeleteChannel(vhost, channel string) error {
	// del "ch|vhost_name/channel_name"
	return s.db.Remove(s.compKey(storageChanKeyPrefix, vhost, channel))
}

// Clear truncates all the data in the storage.
//
// Returns an error if something went wrong.
func (s *storage) Clear() error {
	return s.db.Clear()
}

// Save writes down and synchronizes all the data.
//
// Returns an error if something went wrong.
func (s *storage) Save() error {
	return s.db.Sync(true)
}

// Kill closes the storage.
func (s *storage) Kill() {
	s.db.Close()
}
