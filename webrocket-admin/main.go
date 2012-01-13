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
	"flag"
	"fmt"
	"os"
	"io"
	"../webrocket"
	stepper "../gostepper"
)

var (
	Addr       string
	CookiePath string
	Cookie     string
	Cmd        string
	Params     []string
	s          stepper.Stepper
)

func init() {
	flag.StringVar(&Addr, "admin-addr", "127.0.0.1:8082", "Address of the server's admin interface")
	flag.StringVar(&CookiePath, "cookie", "/var/lib/webrocket/cookie", "Path to the server's cookie file")
	flag.Parse()
	
	Cmd = flag.Arg(0)
	if Cmd == "" {
		usage()
		os.Exit(1)
	}

	readCookie()
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [options] command [args ...]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\nAvailable commands\n")
	fmt.Fprintf(os.Stderr, "  list_vhosts: Shows list of the registered vhosts\n")
	fmt.Fprintf(os.Stderr, "  add_vhost [path]: Registers new vhost\n")
	fmt.Fprintf(os.Stderr, "  delete_vhost [path]: Removes specified vhost\n")
	fmt.Fprintf(os.Stderr, "  show_vhost [path]: Shows information about the specified vhost\n")
	fmt.Fprintf(os.Stderr, "  clear_vhosts: Removes all vhosts\n")
	fmt.Fprintf(os.Stderr, "  regenerate_vhost_token [path]: Generates new access token for the specified vhost\n")
	fmt.Fprintf(os.Stderr, "  list_channels [vhost]: Shows list of vhosts opened under given channel\n")
	fmt.Fprintf(os.Stderr, "  add_channel [vhost] [name]: Shows list of vhosts opened under given channel\n")
	fmt.Fprintf(os.Stderr, "  delete_channel [vhost] [name]: Removes channel from the specified vhost\n")
	fmt.Fprintf(os.Stderr, "  clear_channels [vhost]: Removes all channels from the specified vhost\n")
	fmt.Fprintf(os.Stderr, "  list_workers [vhost]: Shows list of the backend workers connected to the specified vhost\n")
	fmt.Fprintf(os.Stderr, "\nAvailable options\n")
	flag.PrintDefaults()
}

func readCookie() {
	s.Start("Reading cookie")
	cookieFile, err := os.Open(CookiePath)
	if err != nil {
		s.Fail(err.Error(), true)
	}
	var buf [webrocket.CookieSize]byte
	n, err := io.ReadFull(cookieFile, buf[:])
	if n != webrocket.CookieSize || err != nil {
		s.Fail("invalid cookie format", true)
	}
	Cookie = string(buf[:])
	s.Ok()
}

func main() {
	switch Cmd {
	case "list_vhosts":
		cmdListVhosts()
	case "add_vhost":
		cmdAddVhost(flag.Arg(1))
	case "delete_vhost":
		cmdDeleteVhost(flag.Arg(1))
	case "show_vhost":
		cmdShowVhost(flag.Arg(1))
	case "clear_vhosts":
		cmdClearVhosts()
	case "regenerate_vhost_token":
		cmdRegenerateVhostToken(flag.Arg(1))
	case "list_channels":
		cmdListChannels(flag.Arg(1))
	case "add_channel":
		cmdAddChannel(flag.Arg(1), flag.Arg(2))
	case "delete_channel":
		cmdDeleteChannel(flag.Arg(1), flag.Arg(2))
	case "clear_channels":
		cmdClearChannels(flag.Arg(1))
	case "list_workers":
		cmdListWorkers(flag.Arg(1))
	default:
		usage()
	}
}