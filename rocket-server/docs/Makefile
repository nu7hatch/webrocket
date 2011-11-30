ASCIIDOC = asciidoc
A2X = a2x

DOCS_MAN1 = \
	rocket-server.1

DOCS_MAN1_HTML = \
	rocket-server.1.html

man: man1
man1: $(DOCS_MAN1) $(DOCS_MAN1_HTML)

$(DOCS_MAN1): % : %.txt
$(DOCS_MAN1_HTML): %.html : %.txt

%: %.txt
	$(A2X) -f manpage $<

%.html: %.txt
	$(ASCIIDOC) -d manpage -o $@ $<

clean:
	rm -f $(DOCS_MAN1) $(DOCS_MAN1_HTML)