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

Once you install the Go compiler, building the WebRocket is very easy:

    $ git clone git://github.com/nu7hatch/webrocket.git
    $ cd webrocket
    $ make install
	
## Server

To start the server node with defauld configuration simply run:

    $ webrocket-server
	
Obviously you can tweak up the configuration whatever you want:

    $ webrocket-server -wsaddr "myhost.com:9772" -ctladdr "localhost:9773"

To get more information check `man webrocket-server` or just run
the server with `-help` switch.

## Management (UNDER DEVELOPMENT)

Architecture of WebRocket is oriented on distributed deployment and
horizontal scalability, that's why to manage your node you have to use
another tool, `rocket-ctl`. To quick start with the server you just
have to create a **vhost** and add at least one user for it. 

    $ webrocket-ctl add_vhost /hello
	$ webrocket-ctl add_user /hello joe READ|WRITE
	
Again, for more details check `man rocket-ctl` or simply run it with
help command.

## Note on Patches/Pull Requests
 
* Fork the project.
* Make your feature addition or bug fix.
* Add tests for it. This is important so I don't break it in a
  future version unintentionally.
* Commit, do not mess with rakefile, version, or history.
  (if you want to have your own version, that is fine but bump version in a commit by itself I can ignore when I pull)
* Send me a pull request. Bonus points for topic branches.

## Copyright

Copyright (C) 2011 by Krzysztof Kowalik <chris@nu7hat.ch>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
