REPO_URL  := https://club1.fr/~nicolas/git/dna-backup/

REPO_PATH := repo
GIT_PATH  := git
GITC      := git -C $(REPO_PATH)

TEMP      := temp

.PHONY: all
all: versions results
	@echo "total size    : `head --lines=1 results | cut -f1`"
	@echo "version count : `wc -l versions`"

versions: results
	grep [0-9]$$ $< | cut -f1 > $@

results: | $(TEMP)
	du -bad 2 $(TEMP) | sort -k2 > $@

$(TEMP): commits ../dna-backup
	rm -rf $(TEMP)
	./exp.sh $< $(REPO_PATH) $(TEMP)

../dna-backup: .FORCE
	@$(MAKE) -C .. --no-print-directory dna-backup

commits: | repo git
	$(GITC) log --reverse --no-merges --pretty=tformat:"%H	%as" \
	| sort --unique --key=2 \
	> $@

repo git:
	git clone --separate-git-dir=$(GIT_PATH) $(REPO_URL) $(REPO_PATH)
# remove warning about detached head state
	$(GITC) config advice.detachedHead false

.PHONY: clean
clean:
	rm -rf $(REPO_PATH) $(GIT_PATH) $(TEMP)
	rm -f commits results versions

.FORCE: ;
