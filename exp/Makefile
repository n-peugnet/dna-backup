REPO_URL  := https://club1.fr/~nicolas/git/dna-backup/

SHELL := /bin/bash

# Reset LC_NUMERIC to make printf thousands separator work as expected
export LC_NUMERIC  :=

export DNA_BACKUP  ?= ../dna-backup
export MAX_VERSION ?= 5
export COMMITS     ?= commits
export DNA_4K      ?= dna_4k
export DNA_8K      ?= dna_8k
export DIFFS       ?= diffs

export REPO_PATH   ?= repo
GIT_PATH  := git
GITC      := git -C $(REPO_PATH)

DNADIRS   := $(DNA_4K) $(DNA_8K)
DATADIRS  := $(DNA_4K) $(DNA_8K) $(DIFFS)
SIZEFILES := $(DATADIRS:%=%.size)

.PHONY: all exp
all exp: summary.csv $(SIZEFILES)
	@echo "=============== SUMMARY ==============="
	@head -n 1 $< | while IFS="," read -r $(DATADIRS); do \
		printf "$(DATADIRS:%=\%13s)\n" $(DATADIRS:%=$$%); \
	done
	@tail -n +2 $< | while IFS="," read -r $(DATADIRS); do \
		printf "$(DATADIRS:%=\%'13d)\n" $(DATADIRS:%=$$%); \
	done
	@echo "================ TOTAL ================"
	@for i in $(SIZEFILES); do cat $$i | paste -sd+ | bc; done | while read j; do \
		printf "%'13d " $$j; \
	done
	@echo

summary.csv: $(SIZEFILES)
	echo $(DATADIRS) | tr ' ' ',' > $@
	paste -d "," $^ >> $@

$(DNADIRS:%=%.size): %.size: %.versions
	rm -rf $@
	for i in $</*; do \
		cut -f1 $$i | paste -sd+ | bc >> $@; \
	done

%.size: % run
	find $< -type f -exec du -ba {} + | sort -k2 | cut -f1 > $@

%.versions: %.results
	rm -rf $@
	mkdir -p $@
	for i in $*/*; do \
		v=`echo $$i | cut -d / -f2`; \
		grep $$i $< > $@/$$v ; \
	done

%.results: % run
	find $< -type f -exec du -ba {} + \
	| grep -v hashes \
	| sort -k2 \
	> $@

run: $(COMMITS) $(DNA_BACKUP) | $(DATADIRS)
	rm -rf $(DATADIRS:%=%/*)
	./exp.sh
	touch $@

$(DATADIRS):
	mkdir $@

$(DNA_BACKUP): .FORCE
	@$(MAKE) -C $(@D) --no-print-directory $(@F)

$(COMMITS): | repo git
	$(GITC) log --reverse --no-merges --pretty=tformat:"%H	%as" \
	| sort --unique --key=2 \
	> $@

repo git &:
	git clone --separate-git-dir=$(GIT_PATH) $(REPO_URL) $(REPO_PATH)
# remove warning about detached head state
	$(GITC) config advice.detachedHead false

.PHONY: clean
clean: mostlyclean
	rm -rf $(REPO_PATH) $(GIT_PATH)
	rm -f $(COMMITS)

mostlyclean: resultsclean
	rm -rf $(DATADIRS) $(DNADIRS:%=%.versions) run

resultsclean:
	rm -f summary.csv $(SIZEFILES)

.FORCE: ;
