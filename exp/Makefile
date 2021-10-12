REPO_URL  := https://club1.fr/~nicolas/git/dna-backup/

SHELL := /bin/bash

# Reset LC_NUMERIC to make printf thousands separator work as expected
export LC_NUMERIC  :=

export DNA_BACKUP  ?= ../dna-backup
export DNA_PARAMS  ?= dna_params
export MAX_VERSION ?= 5
export COMMITS     ?= commits.daily
DNA_4K             ?= dna_4k
DNA_8K             ?= dna_8k
NOPACK             ?= nopack
export GIT_NOPACK  ?= $(abspath $(NOPACK))
export TARGZ       ?= targz
export REAL        ?= real
export DIFFS       ?= diffs

export REPO_PATH   ?= repo
export GIT_PATH    ?= $(abspath git)
GITC      := git -C $(REPO_PATH)

# Number of initial commits to skip. This option has been added for the
# Linux kernel repo, as the first commits are too far away date wise.
SKIP_COMMITS ?= 30

DNADIRS   := $(DNA_4K) $(DNA_8K)
DATADIRS  := $(DNA_4K) $(DNA_8K) $(DIFFS) $(NOPACK) $(TARGZ) $(REAL)
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

$(DNADIRS:%=%.size) $(REAL).size: %.size: %.versions run
	rm -rf $@
	for i in $</*; do \
		cut -f1 $$i | paste -sd+ | bc >> $@; \
	done

$(NOPACK).size: $(NOPACK).versions run
	rm -rf $@
	prev=/dev/null; for i in $</*; do \
		diff -u0 $$prev $$i \
		| sed -e '1,2d' -e '/^-/d' -e 's/^+//' -e '/^@@/d' \
		| grep /objects/ \
		| cut -f1 \
		| paste -sd+ \
		| bc \
		>> $@; \
		prev=$$i; \
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

run: $(COMMITS) $(DNA_BACKUP) $(DNA_PARAMS) | printvars $(DATADIRS) $(NOPACK).versions $(REAL).versions
	rm -rf $(DATADIRS:%=%/*) $(DATADIRS:%=%.versions/*)
	./exp.sh
	touch $@

.PHONY: printvars
printvars:
	@$(foreach V,$(sort $(.VARIABLES)),\
		$(if $(filter-out environment% default automatic,$(origin $V)),\
		$(info $V=$($V))))

$(DNA_PARAMS):
	echo "$(DNA_4K)	-c 4096" >> $@
	echo "$(DNA_8K)	-c 8192" >> $@

$(DATADIRS) $(NOPACK).versions $(REAL).versions:
	mkdir $@

$(DNA_BACKUP): .FORCE
	@$(MAKE) -C $(@D) --no-print-directory $(@F)

commits.monthly: commits.daily
	sed -n '0~30p' $< > $@

commits.weekly: commits.daily
	sed -n '0~7p' $< > $@

commits.daily: | repo git
	$(GITC) log --reverse --no-merges --pretty=tformat:"%H	%as" \
	| sort --unique --key=2 \
	| tail -n +$(SKIP_COMMITS) \
	> $@

repo git &:
	git clone --separate-git-dir=$(GIT_PATH) $(REPO_URL) $(REPO_PATH)
# remove warning about detached head state
	$(GITC) config advice.detachedHead false

.PHONY: clean
clean: mostlyclean
	rm -rf $(REPO_PATH) $(GIT_PATH)
	rm -f commits.*

mostlyclean: resultsclean
	rm -rf $(DATADIRS) $(DATADIRS:%=%.versions)
	rm -f run $(DATADIRS:%=%.results) $(DNA_PARAMS)

resultsclean:
	rm -f summary.csv $(SIZEFILES)

.FORCE: ;
