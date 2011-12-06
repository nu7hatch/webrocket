ASCIIDOC = asciidoc
A2X = a2x

DOCS_MAN1 = \
	webrocket-server.1

DOCS_MAN1_HTML = \
	webrocket-server.1.html

DOCS_MAN7 = \
	webrocket.7

DOCS_MAN7_HTML = \
	webrocket.7.html

all: man
clean: clean-man

man: man1 man7
man1: $(DOCS_MAN1) $(DOCS_MAN1_HTML)
man7: $(DOCS_MAN7) $(DOCS_MAN7_HTML)

$(DOCS_MAN1): % : %.txt
$(DOCS_MAN1_HTML): %.html : %.txt
$(DOCS_MAN7): % : %.txt
$(DOCS_MAN7_HTML): %.html : %.txt

%: %.txt
	$(A2X) -L -f manpage $<

%.html: %.txt
	$(ASCIIDOC) -b html5 -d manpage -a linkcss -a stylesheet=style/custom.css \
	-a disable-javascript -o $@ $<

clean-man:
	rm -f $(DOCS_MAN1) $(DOCS_MAN1_HTML) $(DOCS_MAN7) $(DOCS_MAN7_HTML)
