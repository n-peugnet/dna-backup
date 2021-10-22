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

%.pdf: %.tex
	latexmk $(LATEXMK_FLAGS) -pdf -f $<

mostlyclean:
	latexmk $(LATEXMK_FLAGS) -c

clean:
	latexmk $(LATEXMK_FLAGS) -C

.PHONY: all pdf mostlyclean clean
