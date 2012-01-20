UNAME = $(shell uname -s)
BUILD_DIR = $(shell pwd)/_build

ifndef VERBOSE
MAKEFLAGS += --no-print-directory
endif

ifeq ($(UNAME),Darwin)
ECHO=echo
else
ECHO=echo -e
endif

ASCIIDOC = asciidoc

all: clean gouuid gocabinet gostepper server admin
	@rm -rf _build
	@$(ECHO) "\033[35mgathering things together\033[0m"
	mkdir -p $(BUILD_DIR)/bin $(BUILD_DIR)/share
	cp webrocket-admin/webrocket-admin $(BUILD_DIR)/bin
	cp webrocket-server/webrocket-server $(BUILD_DIR)/bin
	@$(ECHO) "\n\033[32mHurray! WebRocket has been built into \033[1;32m$(BUILD_DIR)\033[0m\n"

clean: clean-lib clean-server clean-admin clean-deps
	rm -rf build

clean-deps: clean-gouuid clean-gocabinet clean-gostepper
install: install-server install-admin install-man
check: all check-lib

lib:
	@export __webrocket_st=100
	@$(ECHO) "\033[35mbuilding \033[1;32m./webrocket\033[0m"
	@$(MAKE) -C webrocket
	cp webrocket/_obj/*.a .
clean-lib:
	@$(MAKE) -C webrocket clean
	rm -f *.a
check-lib:
	@$(MAKE) -C webrocket test

server: lib
	@$(ECHO) "\033[35mbuilding \033[1;32m./webrocket-server\033[0m"
	@$(MAKE) -C webrocket-server
clean-server:
	@$(MAKE) -C webrocket-server clean
install-server:
	@$(MAKE) -C webrocket-server install

admin: lib server
	@$(ECHO) "\033[35mbuilding \033[1;32m./webrocket-admin\033[0m"
	@$(MAKE) -C webrocket-admin
clean-admin:
	@$(MAKE) -C webrocket-admin clean
install-admin:
	@$(MAKE) -C webrocket-admin install

man:
	@$(ECHO) "\033[35mbuilding \033[1;32m./docs\033[0m"
	-@$(MAKE) -C docs
clean-man:
	-@$(MAKE) -C docs clean
install-man:
	-@$(MAKE) -C docs install

gouuid:
	@$(ECHO) "\033[35mbuilding \033[1;32m./deps/gouuid\033[0m"
	@$(MAKE) -C deps/gouuid
	cp deps/gouuid/_obj/github.com/nu7hatch/*.a .
clean-gouuid:
	$(MAKE) -C deps/gouuid clean

gocabinet:
	@$(ECHO) "\033[35mbuilding \033[1;32m./deps/gocabinet\033[0m"
	@$(MAKE) -C deps/gocabinet
	cp deps/gocabinet/_obj/github.com/nu7hatch/*.a .
clean-gocabinet:
	$(MAKE) -C deps/gocabinet clean

gostepper:
	@$(ECHO) "\033[35mbuilding \033[1;32m./deps/gostepper\033[0m"
	@$(MAKE) -C deps/gostepper
	cp deps/gostepper/_obj/github.com/nu7hatch/*.a .
clean-gostepper:
	$(MAKE) -C deps/gostepper clean

papers:
	-$(ASCIIDOC) -d article -o INSTALL.html INSTALL
	-$(ASCIIDOC) -d article -o NEWS.html NEWS
	-$(ASCIIDOC) -d article -o CONTRIBUTE.html CONTRIBUTE
	-$(ASCIIDOC) -d article -o README.html README