REPO_URL  := https://club1.fr/~nicolas/git/dna-backup/

TEMP_TEMPLATE := dna-backup-bench-XXXXX
MKTEMP        := mktemp --tmpdir -d $(TEMP_TEMPLATE)

REPO_PATH := repo
GIT_PATH  := git
GITC      := git -C $(REPO_PATH)

.PHONY: apply
apply: TEMP := $(shell $(MKTEMP))
apply: commits
	$(MAKE) -C .. dna-backup
	cat $< | while read i; do \
		$(GITC) checkout `echo "$$i" | cut -f1`; \
		../dna-backup commit -v 2 $(REPO_PATH) $(TEMP); \
	done
	du -bad 2 $(TEMP)

commits: Makefile | repo git
	$(GITC) log --reverse --no-merges --pretty=tformat:"%H	%as" \
	| sort --unique --key=2 \
	> $@

repo git:
	git clone --separate-git-dir=$(GIT_PATH) $(REPO_URL) $(REPO_PATH)
# remove warning about detached head state
	$(GITC) config advice.detachedHead false

.PHONY: clean
clean:
	rm -rf $(REPO_PATH) $(GIT_PATH)
	rm -f commits

.FORCE: ;
