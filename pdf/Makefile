FILES = doc
PDF = $(FILES:%=%.pdf)

# Debug flags
ifndef DEBUG
LATEXMK_FLAGS += -quiet
endif

all pdf: $(PDF)

.SECONDEXPANSION:
%.pdf: %.tex $$(wildcard $$*.bib)  assets/*.tex
	latexmk $(LATEXMK_FLAGS) -pdf -f $<

mostlyclean:
	latexmk $(LATEXMK_FLAGS) -c
	rm -f $(FILES:%=%.synctex.*) $(FILES:%=%.bbl)
	rm -f assets/*.aux

clean: mostlyclean
	rm -f $(PDF)

.PHONY: all pdf mostlyclean clean
