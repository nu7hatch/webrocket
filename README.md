# WEBROCKET - Distributed WebSockets/MQ server

WebRocket is a hybrid of MQ and WebSockets server with great support for 
horizontal scalability. WebRocket is very fast and easy to use from both
sides: backend (via MQ conneciton) and frontend (via WebSockets). 
This combination will lead you to new quality of web development and
will finally make bidirectional Web easy for everyone. 
 
Package contains also the `webrocket` library, which provides highly extensible
backend, eg. for defining your own protocols (see *Hacking* section for details). 

## Installation

Project is writter in awesome [Go language](http://golang.org)!
If you don't have Go installed yet... Follow this [installation guide](http://golang.org/doc/install.html) 
and get **the newest version** of the compiler. 

WebRocket is using bunch of unreleased features, and want to catch up with 
Go's development, so remember to clone **the head version** or the latest 
**weekly release**:

    $ hg clone https://go.googlecode.com/hg/ go

Once you install the Go compiler, building the WebRocket is very easy.
First, clone the repo:

    $ git clone git://github.com/nu7hatch/webrocket.git
    $ cd webrocket
	
... and install the WebRocket library with all tools:
 	
    $ ./all.bash
	
## Server

To start the server node with defauld configuration simply run:

    $ rocket-server
	
Obviously you can tweak up the configuration whatever you want:

    $ rocket-server -wsaddr "myhost.com:9772" -ctladdr "localhost:9773"

To get more information about all settings run server with help switch:

    $ rocket-server -help

## Management (UNDER DEVELOPMENT)

Architecture of WebRocket is oriented on distributed deployment and
horizontal scalability, that's why to manage your node you have to use
another tool, `rocket-ctl`. To quick start with the server you just
have to create a **vhost** and add at least one user for it. 

    $ rocket-ctl add_vhost /hello
	$ rocket-ctl add_user /hello joe READ|WRITE
	
Again, for more details simply run help command:

    $ rocket-ctl -help

## Monitoring (UNDER DEVELOPMENT)

To monitor your server or cluster activity use the `rocket-monitor` tool:

    $ rocket-monitor

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
