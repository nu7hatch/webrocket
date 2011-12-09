ASCIIDOC = asciidoc

DOCS_HTML = \
	WFP.html

all: docs
clean: clean-docs

docs: $(DOCS_HTML)

$(DOCS_HTML): %.html : %.txt

%.html: %.txt
	$(ASCIIDOC) -b html5 -d article -a linkcss -a icons \
	-a pygments -a stylesdir=../docs/style/ -o $@ $<

clean-docs:
	rm -f $(DOCS_HTML)
