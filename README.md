# Rocket - Advanced WebSocket server

Rocket is very fast and reliable WebSockets server written in [Go language](http://golang.org)!
Package contains also the `webrocket` library, which provides highly extensible
backend, eg. for defining your own protocols (see *Hacking* section for details). 

## History and motivation

Some time ago i wrote [first version of Rocket](https://github.com/araneo/rocket) 
server in Ruby at top of EventMachine and EM-Websocket libs. This proof of concept
version worked fine and done his job properly so i din't touched anything... until 
i found perfect platform to reimplement it! Go language with its concurrency model 
powered by goroutines and channels, and with amazing standard library impressed me 
and gave opportunity to write much faster, nicer and more stable Rocket implementation. 

Btw. Last time i was so impressed by language and it's standard library when i wrote
my first Ruby application (6 years ago?)!

## Installation

First of all, i predict you don't have Go installed... Follow this 
[installation guide](http://golang.org/doc/install.html) and get the 
newest version of compiler. Go is active actively developed, so the
best idea is to use head version and update it regullary. 

Once you install the Go compiler, building rocket is very easy.
First, clone the repo:

    $ git clone git://github.com/nu7hatch/webrocket.git
	$ cd webrocket
	
Build and install the `webrocket` library:
 	
	$ make && make install
	
Finally, build `rocket` command line tool:

    $ cd server
	$ make
	
If everything will go fine, then you will find the `./rocket` binary in 
current directory.

## Usage

Server is quite easy to configure and run. The only thing you have to do
is to create your own configuration based on the versioned `example.conf` file. 

    $ cp example.conf my.conf
	$ # edit configuration file...
    $ rocket my.conf

By default rocket listens on port `9772` on localhost. You can change it
in your configuration or by setting proper flags. Use `rocket --help` to 
check available flags and options.

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
