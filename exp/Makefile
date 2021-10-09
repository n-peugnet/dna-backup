REPO_URL  := https://club1.fr/~nicolas/git/dna-backup/

SHELL := /bin/bash

# Reset LC_NUMERIC to make printf thousands separator work as expected
export LC_NUMERIC  :=

export DNA_BACKUP  ?= ../dna-backup
export DNA_PARAMS  ?= dna_params
export MAX_VERSION ?= 5
export COMMITS     ?= commits
DNA_4K             ?= dna_4k
DNA_8K             ?= dna_8k
NOPACK             ?= nopack
export GIT_NOPACK  ?= $(abspath $(NOPACK))
export DIFFS       ?= diffs

export REPO_PATH   ?= repo
export GIT_PATH    ?= $(abspath git)
GITC      := git -C $(REPO_PATH)

DNADIRS   := $(DNA_4K) $(DNA_8K)
DATADIRS  := $(DNA_4K) $(DNA_8K) $(DIFFS) $(NOPACK)
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

$(DNADIRS:%=%.size) $(NOPACK).size: %.size: %.versions run
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

run: $(COMMITS) $(DNA_BACKUP) $(DNA_PARAMS) | $(DATADIRS) $(NOPACK).versions
	rm -rf $(DATADIRS:%=%/*)
	./exp.sh
	touch $@

$(DNA_PARAMS):
	echo "$(DNA_4K)	-c 4096" >> $@
	echo "$(DNA_8K)	-c 8192" >> $@

$(DATADIRS) $(NOPACK).versions:
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
	rm -rf $(DATADIRS) $(DATADIRS:%=%.versions)
	rm -f run $(DATADIRS:%=%.results) $(DNA_PARAMS)

resultsclean:
	rm -f summary.csv $(SIZEFILES)

.FORCE: ;
