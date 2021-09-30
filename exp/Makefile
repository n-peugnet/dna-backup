REPO_URL  := https://club1.fr/~nicolas/git/dna-backup/

REPO_PATH := repo
GIT_PATH  := git
GITC      := git -C $(REPO_PATH)

.PHONY: all
all: versions
	@echo "total size    : `awk '{s+=$$1}END{print s}' $<`"
	@echo "version count : `wc -l $<`"

versions: results
	grep [0-9]$$ $< | cut -f1 > $@

results: commits ../dna-backup
	./exp.sh $< $(REPO_PATH) 1| sort -k2 >$@

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
	rm -rf $(REPO_PATH) $(GIT_PATH)
	rm -f commits results versions

.FORCE: ;
