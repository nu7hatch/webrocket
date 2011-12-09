ASCIIDOC = asciidoc

DOCS_HTML = \
	WFP.html

all: docs
clean: clean-docs

docs: $(DOCS_HTML)

$(DOCS_HTML): %.html : %.txt

%.html: %.txt
	$(ASCIIDOC) -b html5 -d article -a linkcss -a stylesheet=../docs/style/custom.css -o $@ $<

clean-docs:
	rm -f $(DOCS_HTML)
