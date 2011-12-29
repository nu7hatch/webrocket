ASCIIDOC = asciidoc
ASCIIDOC_OPTS = -b html5 -d manpage -f ./asciidoc.conf
A2X = a2x

DOCS_MAN1 = \
	webrocket-server.1
DOCS_MAN1_HTML = \
	webrocket-server.1.html

DOCS_MAN3 = \
	webrocket.3
DOCS_MAN3_HTML = \
	webrocket.3.html

all: man
clean: clean-man

man: man1 man3
man1: $(DOCS_MAN1) $(DOCS_MAN1_HTML)
man3: $(DOCS_MAN3) $(DOCS_MAN3_HTML)

$(DOCS_MAN1): % : %.txt
$(DOCS_MAN1_HTML): %.html : %.txt
$(DOCS_MAN3): % : %.txt
$(DOCS_MAN3_HTML): %.html : %.txt

%: %.txt
	$(A2X) -L -f manpage $<

%.html: %.txt
	$(ASCIIDOC) $(ASCIIDOC_OPTS) -o $@ $<

clean-man: clean-man1 clean-man3

clean-man1:
	rm -f $(DOCS_MAN1) $(DOCS_MAN1_HTML)

clean-man3:
	rm -f $(DOCS_MAN3) $(DOCS_MAN3_HTML)