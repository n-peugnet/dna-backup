REPO_URL  := https://club1.fr/~nicolas/git/dna-backup/

export MAX_VERSION ?= 3
export LO_DEV      ?= /dev/loop0
export COMMITS     ?= commits
export BACKUP      ?= backup
export DIFFS       ?= diffs
export UDF         ?= udf

export REPO_PATH   ?= repo
GIT_PATH  := git
GITC      := git -C $(REPO_PATH)

DATADIRS  := $(BACKUP) $(DIFFS) $(UDF)
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

udf-size: udf run
	du -b $< | cut -f1 > $@

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

run: commits ../dna-backup | $(DATADIRS)
	rm -rf $(DATADIRS:%=%/*)
	./exp.sh
	touch $@

backup diffs:
	mkdir $@

udf: | udffile.img
	mkdir $@
	sudo mount -t udf $(LO_DEV) $@
	sudo chown $$USER:$$USER $@

udffile.img:
	dd if=/dev/zero of=$@ bs=100M count=10
	sudo losetup $(LO_DEV) $@
	sudo cdrwtool -d $(LO_DEV) -q
	sudo mkfs -t udf --media-type=dvdrw $(LO_DEV)

../dna-backup: .FORCE
	@$(MAKE) -C .. --no-print-directory dna-backup

commits: | repo git
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
	rm -f commits

mostlyclean: resultsclean
	-sudo umount -fqd udf
	-sudo losetup -d $(LO_DEV)
	rm -rf $(DATADIRS) versions
	rm -f udffile.img run

resultsclean:
	rm -f results summary.csv $(SIZEFILES)

.FORCE: ;
