include $(GOROOT)/src/Make.inc

TARG = webrocket

GOFILES = \
	permission.go \
	message.go \
	connection.go \
	channel.go \
	context.go \
	endpoint.go \
	vhost.go \
	statuses.go \
	websocket_client.go \
	websocket_endpoint.go \
	websocket_handler.go \
	websocket_protocol.go \
	backend_agent.go \
	backend_identity.go \
	backend_endpoint.go \
	backend_lobby.go \
	backend_protocol.go

include $(GOROOT)/src/Make.pkg

format:
	gofmt -s=true -w .