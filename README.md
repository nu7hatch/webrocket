# Rocket - Advanced WebSocket server

Rocket is very fast and reliable WebSockets server written in [Go language](http://golang.org)!
Package contains also the `webrocket` library, which provides highly extensible
backend, eg. for defining your own protocols (see *Hacking* section for details). 

## Installation

First of all, I predict you don't have Go installed... Follow this 
[installation guide](http://golang.org/doc/install.html) and get **the newest version**
of the compiler. Rocket uses some of unreleased websocket's stuff, so remember to clone
**the head version** or the latest **weekly release**:

    $ hg clone https://go.googlecode.com/hg/ go

Go is very actively developed, so it's good idea is to use head version and update 
it regullary. 

Once you install the Go compiler, building the webrocket is very easy.
First, clone the repo:

    $ git clone git://github.com/nu7hatch/webrocket.git
    $ cd webrocket
	
Build and install the `webrocket` library:
 	
    $ make && make install
	
Finally, build `rocket` command line tool:

    $ cd server
    $ make
	
If everything will go fine, then you will find the `./rocket` binary in current 
directory.

## Usage

Server is quite easy to configure and run. The only thing you have to do
is to create your own configuration based on the versioned `example.json` file. 

    $ cp example.json my.json
    $ # edit configuration file...
    $ rocket my.json

By default rocket listens on port `9772` on localhost. You can change it
in your configuration or by setting proper flags. Use `rocket --help` to 
check available flags and options.

## Protocol

By default, Rocket implements simple JSON protocol for pub-sub channels. All following actions, 
when successfully performed, should return:

    {"ok": true}
    
Otherwise, when error encountered, then response message has following format:

    {"err": "error-code"}

### Authentication

    {"auth": {"access": "access-type", "secret": "secret-key"}}

* `secret` - key for specified access type
* `access` - can be either `read-only` or `read-write`

Error responses:

* `invalid_credentials` - obviously, returned when given secret is invalid

### Subscribing

    {"subscribe": {"channel": "channel-name"}}

* `channel` - name of channel you want to (un)subscribe, not existing channels are created automatically
    
Error responses:

* `access_denied` - returned when current session is not authenticated for reading
* `invalid_channel_name` - returned when given channel name is invalid

### Unsubscribing

    {"unsubscribe": {"channel": "channel-name"}}

### Publishing

    {"publish": {"event": "event-name", "channel": "channel-name", "data": {"foo": "bar"}}}

* `event` - communication is event oriented, so each message needs to specify which event triggers it
* `channel` - channel have to exist
* `data` - published data

Error responses:

* `access_denied` - returned when current session is not authenticated for writing
* `invalid_data` - returned when published message has invalid format
* `invalid_channel` - returned when destination channel doesn't exist

### Closing session

    {"logout": true}
    
### Safe disconnecting

    {"disconnect": true}

## Hacking

TODO...

## Note on Patches/Pull Requests
 
* Fork the project.
* Make your feature addition or bug fix.
* Add tests for it. This is important so I don't break it in a
  future version unintentionally.
* Commit, do not mess with rakefile, version, or history.
  (if you want to have your own version, that is fine but bump version in a commit by itself I can ignore when I pull)
* Send me a pull request. Bonus points for topic branches.

## Copyright

Copyright (c) 2011 Chris Kowalik (nu7hatch). See LICENSE for details.
