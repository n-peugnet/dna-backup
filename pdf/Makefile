FILES = doc
TEX = $(FILES:%=%.tex)
PDF = $(FILES:%=%.pdf)

# Debug flags
ifdef DEBUG
LATEXMK_FLAGS += -verbose
else
LATEXMK_FLAGS += -quiet
endif

all pdf: $(PDF)

%.pdf: %.tex assets/*.tex
	latexmk $(LATEXMK_FLAGS) -pdf -f $<

mostlyclean:
	latexmk $(LATEXMK_FLAGS) -c
	rm -f $(FILES:%=%.synctex.*)
	rm -f assets/*.aux

clean: mostlyclean
	rm -f $(PDF)

.PHONY: all pdf mostlyclean clean
