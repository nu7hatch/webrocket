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

Webrocket implements simple and fast JSON-based protocol, offering support for authentication and
basic access controll, channels subscribing, and messages broadcasting. 

Each message handled by the server may cause possible errors. All error which are not affecting
connection with the client are forwarded to it using the following payload format:

    {"err": "ERROR_NAME"}
	
Possible errors are listed for each action.

### Authenticate

Payload:

    {
	    "authenticate": {
		    "user": "user-name", 
			"secret": "secret-key"
		}
	}

* `user` - name of the configured user you want to authenticate
* `secret` - authentication secret for specified user

Errors:

* `INVALID_CREDENTIALS` - returned when given secret is invalid
* `INVALID_USER` - returned when given user does not exist
* `INVALID_PAYLOAD` - returned when payload format is invalid

Success response:

    {"authenticated": "user-name"}

### Subscribe

Payload:

    {"subscribe": {"channel": "channel-name"}}

* `channel` - name of channel you want to subscribe, not existing channels are created automatically
    
Errors:

* `ACCESS_DENIED` - returned when current session is not authenticated for reading
* `INVALID_PAYLOAD` - when payload format is invalid

Success response:

    {"subscribed": "channel-name"}

### Unsubscribe

Payload:

    {"unsubscribe": {"channel": "channel-name"}}

* `channel` - name of channel you want to unsubscribe

Errors:

* `INVALID_PAYLOAD` - when payload format is invalid

Success response:

    {"unsubscribed": "channel-name"}

### Broadcast

Payload:

    {"broadcast": {"event": "event-name", "channel": "channel-name", "data": {"foo": "bar"}}}

* `event` - communication is event oriented, so each message needs to specify which event triggers it
* `channel` - channel have to exist
* `data` - published data

Errors:

* `ACCESS_DENIED` - returned when current session is not authenticated for writing
* `INVALID_CHANNEL` - returned when destination channel doesn't exist
* `INVALID_PAYLOAD` - when payload format is invalid

Success response:

    {"broadcasted": "channel-name"}

### Direct message (NOT IMPLEMENTED)

Payload:

    {"direct": {"event": "event-name", "client": "client-token", "data": {"foo": "bar"}}}

* `event` - communication is event oriented, so each message needs to specify which event triggers it
* `channel` - channel have to exist
* `data` - published data

Errors:

* `INVALID_CLIENT` - returned when destination client doesn't exist
* `INVALID_PAYLOAD` - when payload format is invalid

Success response:

    {"directSent": "client-token"}

### Session logout

Payload:

    {"logout": true}
    
Errors:

* `INVALID_PAYLOAD` - when payload format is invalid

Success response:

    {"loggedOut": true}
	
### Safe disconnect

Payload:

    {"disconnect": true}

Errors:

* `INVALID_PAYLOAD` - when payload format is invalid

No success response, connection is closed immediately after this message.

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
