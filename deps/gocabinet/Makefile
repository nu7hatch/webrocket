include $(GOROOT)/src/Make.inc

TARG=github.com/nu7hatch/gocabinet
CGOFILES=cabinet.go

include $(GOROOT)/src/Make.pkg

format:
	gofmt -s=true -w=true .

docs:
	gomake clean
	godoc ${TARG} > README.txt
