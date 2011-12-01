ASCIIDOC = asciidoc
A2X = a2x

DOCS_MAN1 = \
	webrocket-server.1

DOCS_MAN1_HTML = \
	webrocket-server.1.html

all: man
clean: clean-man

man: man1
man1: $(DOCS_MAN1) $(DOCS_MAN1_HTML)

$(DOCS_MAN1): % : %.txt
$(DOCS_MAN1_HTML): %.html : %.txt

%: %.txt
	$(A2X) -f manpage $<

%.html: %.txt
	$(ASCIIDOC) -d manpage -o $@ $<

clean-man:
	rm -f $(DOCS_MAN1) $(DOCS_MAN1_HTML)
