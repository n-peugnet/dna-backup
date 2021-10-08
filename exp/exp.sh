#!/bin/bash

# This script expects the following variables to be exported:
# - DNA_BACKUP: the path to dna-backup binary
# - REPO_PATH: the path of the repo the experiment is based on
# - GIT_PATH: the path of the repo git-dir
# - MAX_VERSION: the max number for versions for the experiment
# - COMMITS: the name of the file that contains the lists of versions
# - DNA_4K: the path fo the dna-backup dir with 4K chunksize
# - DNA_8K: the path fo the dna-backup dir with 8K chunksize
# - DIFFS: the path of the git diff dir
# - GIT_NOPACK: the path of the git nopack dir

log() {
	echo -e "\033[90m$(date +%T.%3N)\033[0m" $*
}

set-git-dir() {
	echo gitdir: $1 > $REPO_PATH/.git
}

GITC="git -C $REPO_PATH"
OUT=/tmp/dna-backup-exp-out

# Init git nopack dir
rm $REPO_PATH/.git
$GITC init --separate-git-dir=$GIT_NOPACK
$GITC --git-dir=$GIT_NOPACK config gc.auto 0
set-git-dir $GIT_PATH
nopack_prev=0

# "empty tree" commit
prev="4b825dc642cb6eb9a060e54bf8d69288fbee4904"
last=$(tail --lines=1 $COMMITS | cut -f1)

i=0
cat $COMMITS | while read line
do
	# Get hash
	hash=$(echo "$line" | cut -f1)

	# Check out repo
	log "check out $hash"
	$GITC checkout $hash 2> $OUT \
	|| (log "error checking out"; cat $OUT; exit 1)

	# Create git diff for this version
	log "create git diff for this version"
	diff=$(printf "%s/%05d.diff.gz" $DIFFS $i)
	$GITC diff --minimal --binary --unified=0 -l0 $prev \
	| gzip \
	> $diff

	# Create git nopack for this version
	log "create git nopack for this version"
	set-git-dir $GIT_NOPACK
	$GITC add .
	$GITC commit -m $hash &> $OUT \
	|| (log "error commiting to nopack"; cat $OUT; exit 1)
	ls $GIT_NOPACK/objects/pack
	find $GIT_NOPACK -type f -exec du -ba {} + \
	| grep -v /logs/ \
	| cut -f1 \
	| paste -sd+ \
	| xargs -i echo {} - $nopack_prev \
	| bc \
	> $(printf "%s.versions/%05d" $GIT_NOPACK $i)
	set-git-dir $GIT_PATH

	# Create 4k dna backup for this version
	log "create 4k dna backup for this version"
	$DNA_BACKUP commit -v 2 -c 4096 $REPO_PATH $DNA_4K

	# Create 8k dna backup for this version
	log "create 8k dna backup for this version"
	$DNA_BACKUP commit -v 2 $REPO_PATH $DNA_8K

	if [[ $(( $i % 4 )) == 0 ]]
	then
		# Check restore from git diffs
		log "restore from git diffs"
		TEMP=$(mktemp -d)
		for n in $(seq 0 $i)
		do
			diff=$(printf "%s/%05d.diff.gz" $DIFFS $n)
			cat $diff \
			| gzip --decompress \
			| git -C $TEMP apply --binary --unidiff-zero --whitespace=nowarn -
		done
		cp $REPO_PATH/.git $TEMP/
		log "check restore from diffs"
		diff --brief --recursive $REPO_PATH $TEMP \
		|| log "git patchs restore do not match source"
		rm -rf $TEMP

		# Check restore from 4k dna backup
		log "restore from 4k dna backup"
		TEMP=$(mktemp -d)
		$DNA_BACKUP restore -v 2 -c 4096 $DNA_4K $TEMP
		log "check restore from backup"
		diff --brief --recursive $REPO_PATH $TEMP \
		|| log "dna backup restore do not match source"
		rm -rf $TEMP

		# Check restore from 8k dna backup
		log "restore from 8k dna backup"
		TEMP=$(mktemp -d)
		$DNA_BACKUP restore -v 2 $DNA_8K $TEMP
		log "check restore from backup"
		diff --brief --recursive $REPO_PATH $TEMP \
		|| log "dna backup restore do not match source"
		rm -rf $TEMP
	fi

	prev=$hash
	let i++
	if [[ $i == $MAX_VERSION ]]
	then
		break
	fi
done

# cleanup
log "clean up $REPO_PATH"
$GITC checkout $last 2> $OUT \
|| (log "error checking out back to latest commit"; cat $OUT; exit 2)
