include ${GOROOT}/src/Make.inc

TARG = webrocket
GOFILES = \
	webrocket.go \
	protocol.go

include ${GOROOT}/src/Make.pkg

serv:
	@cd server && make
serv_install:
	@cd server && make install
serv_clean:
	@cd server && make clean
