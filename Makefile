include ${GOROOT}/src/Make.inc

TARG = webrocket
GOFILES = \
	webrocket_user.go \
	webrocket_server.go \
	webrocket_channel.go \
	webrocket_connection.go \
	webrocket_websocket_api.go \
	webrocket_vhost.go

include ${GOROOT}/src/Make.pkg

serv:
	@cd server && make
serv_install:
	@cd server && make install
serv_clean:
	@cd server && make clean
