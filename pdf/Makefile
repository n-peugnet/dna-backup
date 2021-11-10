FILES = doc
PDF = $(FILES:%=%.pdf)

export max_print_line = 100000
PDFLATEX_FLAGS = -shell-escape -file-line-error -interaction=nonstopmode -halt-on-error

all pdf: $(PDF)

.SECONDEXPANSION:
%.pdf: %.tex $$(wildcard $$*.bib)  assets/*.tex
	latexmk $(LATEXMK_FLAGS) $(PDFLATEX_FLAGS) -pdf -f $< \
	| grep --color=always -oE '(.+:[0-9]+:|warning|(Und|Ov)erfull).*' || true

mostlyclean:
	latexmk $(LATEXMK_FLAGS) -c
	rm -f $(FILES:%=%.synctex.*) $(FILES:%=%.bbl)
	rm -f assets/*.aux

clean: mostlyclean
	rm -f $(PDF)

.PHONY: all pdf mostlyclean clean
