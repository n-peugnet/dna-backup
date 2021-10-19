# Copyright (C) 2021 Nicolas Peugnet <n.peugnet@free.fr>

# This file is part of dna-backup.

# dna-backup is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.

# dna-backup is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.

# You should have received a copy of the GNU General Public License
# along with dna-backup.  If not, see <https://www.gnu.org/licenses/>. */

SHELL := /bin/bash

################# Principal parameters #################

REPO_URL     ?= https://git.kernel.org/pub/scm/linux/kernel/git/stable/linux.git
REPO_BRANCH  ?= master
RANGE        ?= daily
# Number of initial commits to skip. This option has been added for the
# Linux kernel repo, as the first commits are too far away date wise.
SKIP_COMMITS ?= 0

# Reset LC_NUMERIC to make printf thousands separator work as expected
export LC_NUMERIC  :=

export DNA_BACKUP  ?= ../dna-backup
export DNA_PARAMS  ?= dna_params
export MAX_VERSION ?= 5
export SKIP_CHECK  ?= 4
export COMMITS     ?= commits.$(RANGE)
DNA_4K             ?= dna_4k
DNA_8K             ?= dna_8k
NOPACK             ?= nopack
export GIT_NOPACK  ?= $(abspath $(NOPACK))
# export BORG        ?= borg
export TARGZ       ?= targz
export REAL        ?= real
export DIFFS       ?= diffs

export REPO_PATH   ?= repo
export GIT_PATH    ?= $(abspath git)
GITC      := git -C $(REPO_PATH)

DNADIRS   := $(DNA_4K) $(DNA_8K)
DNAEXPORT := $(DNADIRS:%=%_export)
DATADIRS  := $(DNADIRS) $(DNAEXPORT) $(DIFFS) $(NOPACK) $(BORG) $(TARGZ) $(REAL)
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

$(BORG).size $(DNAEXPORT:%=%.size): %.size: %.versions run
	rm -rf $@
	prev=0; for i in $</*; do \
		last=`cut -f1 $$i`; \
		let curr=$$last-$$prev; \
		echo $$curr >> $@; \
		prev=$$last; \
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

run: $(COMMITS) $(DNA_BACKUP) $(DNA_PARAMS) | printvars $(DATADIRS) $(NOPACK).versions $(BORG).versions $(REAL).versions $(DNAEXPORT:%=%.versions)
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

$(DATADIRS) $(NOPACK).versions $(BORG).versions $(REAL).versions $(DNAEXPORT:%=%.versions):
	mkdir $@

$(DNA_BACKUP): .FORCE
	@$(MAKE) -C $(@D) --no-print-directory $(@F)

.INTERMEDIATE: commits.monthly
commits.monthly: commits.daily
	sed -n '0~30p' $< > $@

.INTERMEDIATE: commits.weekly
commits.weekly: commits.daily
	sed -n '0~7p' $< > $@

.INTERMEDIATE: commits.daily
commits.daily: commits
	tail -n +$(SKIP_COMMITS) $< > $@

commits: | repo git
	$(GITC) log $(REPO_BRANCH) --reverse --date-order --first-parent --pretty=tformat:"%H	%as" \
	| sort --unique --key=2 \
	> $@

repo git &:
	git clone --separate-git-dir=$(GIT_PATH) $(REPO_URL) $(REPO_PATH)
# remove warning about detached head state
	$(GITC) config advice.detachedHead false

.PHONY: clean
clean: mostlyclean
	rm -rf $(REPO_PATH) $(GIT_PATH)
	rm -f commits

mostlyclean: resultsclean
	rm -rf $(DATADIRS) $(DATADIRS:%=%.versions)
	rm -f run $(DATADIRS:%=%.results) $(DNA_PARAMS) commits.*

resultsclean:
	rm -f summary.csv $(SIZEFILES)

.FORCE: ;
