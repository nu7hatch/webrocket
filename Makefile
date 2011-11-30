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