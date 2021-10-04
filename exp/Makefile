REPO_URL  := https://club1.fr/~nicolas/git/dna-backup/

export DNA_BACKUP  ?= ../dna-backup
export MAX_VERSION ?= 5
export COMMITS     ?= commits
export BACKUP      ?= backup
export DIFFS       ?= diffs

export REPO_PATH   ?= repo
GIT_PATH  := git
GITC      := git -C $(REPO_PATH)

DATADIRS  := $(BACKUP) $(DIFFS)
SIZEFILES := $(DATADIRS:%=%-size)

.PHONY: all exp
all exp: summary.csv $(SIZEFILES)
	@echo "============== SUMMARY =============="
	@cat $< | tr ',' '\t'
	@echo "==============  TOTAL  =============="
	@for i in $(SIZEFILES); do cat $$i | paste -sd+ | bc; done | tr '\n' '\t'
	@echo

summary.csv: $(SIZEFILES)
	echo $(DATADIRS) | tr ' ' ',' > $@
	paste -d "," $^ >> $@

backup-size: versions
	rm -rf $@
	for i in $</*; do \
		cut -f1 $$i | paste -sd+ | bc >> $@; \
	done

diffs-size: diffs run
	find $< -type f -exec du -ba {} + | cut -f1 > $@

versions: results
	rm -rf $@
	mkdir -p $@
	for i in backup/*; do \
		v=`echo $$i | cut -d / -f2`; \
		grep $$i $< > $@/$$v ; \
	done

results: backup run
	find $< -type f -exec du -ba {} + \
	| grep -v hashes \
	| sort -k2 \
	> $@

run: $(COMMITS) $(DNA_BACKUP) | $(DATADIRS)
	rm -rf $(DATADIRS:%=%/*)
	./exp.sh
	touch $@

backup diffs:
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
	rm -rf $(DATADIRS) versions run

resultsclean:
	rm -f results summary.csv $(SIZEFILES)

.FORCE: ;
