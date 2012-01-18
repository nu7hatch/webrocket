include $(GOROOT)/src/Make.inc

TARG = webrocket

GOFILES = \
	webrocket.go \
	permission.go \
	channel.go \
	subscription.go \
	context.go \
	endpoint.go \
	vhost.go \
	statuses.go \
	storage.go \
	websocket_serve_mux.go \
	websocket_connection.go \
	websocket_endpoint.go \
	websocket_handler.go \
	websocket_message.go \
	backend_worker.go \
	backend_identity.go \
	backend_endpoint.go \
	backend_lobby.go \
	backend_lobby_mux.go \
	backend_request.go \
	backend_connection.go \
	admin_endpoint.go

include $(GOROOT)/src/Make.pkg

format:
	gofmt -s=true -w .