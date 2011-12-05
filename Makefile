ASCIIDOC = asciidoc

all: lib server man
clean: clean-lib clean-server clean-man
install: install-lib install-server install-man
check: check-lib

lib:
	@cd webrocket && $(MAKE)
	cp webrocket/_obj/*.a .
clean-lib:
	@cd webrocket && $(MAKE) clean
	rm -f *.a
install-lib:
	@cd webrocket && $(MAKE) install
check-lib:
	@cd webrocket && $(MAKE) test

server:
	@cd webrocket-server && $(MAKE)
clean-server:
	@cd webrocket-server && $(MAKE) clean
install-server:
	@cd webrocket-server && $(MAKE) install

man:
	-@cd docs && $(MAKE)
clean-man:
	-@cd docs && $(MAKE) clean
install-man:
	-@cd docs && $(MAKE) install

papers:
	-$(ASCIIDOC) -d article -o INSTALL.html INSTALL
	-$(ASCIIDOC) -d article -o NEWS.html NEWS
	-$(ASCIIDOC) -d article -o CONTRIBUTE.html CONTRIBUTE
	-$(ASCIIDOC) -d article -o README.html README

.PHONY: lib server man