UNAME = $(shell uname -s)

ifndef VERBOSE
MAKEFLAGS += --no-print-directory
endif

ifeq ($(UNAME),Darwin)
ECHO=echo
else
ECHO=echo -e
endif

ASCIIDOC = asciidoc

all: gozmq gouuid server
clean: clean-lib clean-server clean-man
install: install-server install-man
check: all check-lib

lib:
	@$(ECHO) "\033[35mbuilding \033[1;32m./webrocket\033[0m"
	@cd webrocket && $(MAKE)
	cp webrocket/_obj/*.a .
clean-lib:
	@cd webrocket && $(MAKE) clean
	rm -f *.a
check-lib:
	@cd webrocket && $(MAKE) test

server: lib
	@$(ECHO) "\033[35mbuilding \033[1;32m./webrocket-server\033[0m"
	@cd webrocket-server && $(MAKE)
clean-server:
	@cd webrocket-server && $(MAKE) clean
install-server:
	@cd webrocket-server && $(MAKE) install

man:
	@$(ECHO) "\033[35mbuilding \033[1;32m./docs\033[0m"
	-@cd docs && $(MAKE)
clean-man:
	-@cd docs && $(MAKE) clean
install-man:
	-@cd docs && $(MAKE) install

gozmq:
	@$(ECHO) "\033[35mbuilding \033[1;32m./deps/gozmq\033[0m"
	@cd deps/gozmq && $(MAKE)
	cp deps/gozmq/_obj/github.com/alecthomas/*.a .

gouuid:
	@$(ECHO) "\033[35mbuilding \033[1;32m./deps/gouuid\033[0m"
	@cd deps/gouuid && $(MAKE)
	cp deps/gouuid/_obj/github.com/nu7hatch/*.a .

papers:
	-$(ASCIIDOC) -d article -o INSTALL.html INSTALL
	-$(ASCIIDOC) -d article -o NEWS.html NEWS
	-$(ASCIIDOC) -d article -o CONTRIBUTE.html CONTRIBUTE
	-$(ASCIIDOC) -d article -o README.html README

.PHONY: lib server man