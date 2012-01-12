include $(GOROOT)/src/Make.inc

TARG = webrocket

GOFILES = \
	webrocket.go \
	permission.go \
	message.go \
	connection.go \
	channel.go \
	context.go \
	endpoint.go \
	vhost.go \
	statuses.go \
	storage.go \
	websocket_client.go \
	websocket_endpoint.go \
	websocket_handler.go \
	websocket_protocol.go \
	backend_agent.go \
	backend_identity.go \
	backend_endpoint.go \
	backend_lobby.go \
	backend_request.go \
	backend_protocol.go \
	admin_endpoint.go

include $(GOROOT)/src/Make.pkg

format:
	gofmt -s=true -w .