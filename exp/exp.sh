#!/bin/bash

# This script expects the following variables to be exported:
# - DNA_BACKUP: the path to dna-backup binary
# - REPO_PATH: the path of the repo the experiment is based on
# - MAX_VERSION: the max number for versions for the experiment
# - COMMITS: the name of the file that contains the lists of versions
# - BACKUP: the path fo the dna-backup dir
# - DIFFS: the path of the git diff dir

log() {
	echo -e "\033[90m$(date +%T.%3N)\033[0m" $*
}

GITC="git -C $REPO_PATH"
OUT=/tmp/dna-backup-exp-out

# "empty tree" commit
prev="4b825dc642cb6eb9a060e54bf8d69288fbee4904"
last=$(tail --lines=1 $COMMITS | cut -f1)

i=0
cat $COMMITS | while read line
do
	hash=$(echo "$line" | cut -f1)

	log "check out $hash"
	$GITC checkout $hash 2> $OUT \
	|| (log "error checking out"; cat $OUT; exit 1)

	log "create diff for this version"
	$GITC diff --minimal --binary --unified=0 -l0 $prev \
	| gzip \
	> "$DIFFS/$i.diff.gz"

	log "create backup for this version"
	$DNA_BACKUP commit -v 2 $REPO_PATH $BACKUP

	if [[ $(( $i % 4 )) == 0 ]]
	then
		log "restore from diffs"
		TEMP=$(mktemp -d)
		for n in $(seq 0 $i)
		do
			cat "$DIFFS/$n.diff.gz" \
			| gzip --decompress \
			| git -C $TEMP apply --binary --unidiff-zero --whitespace=nowarn -
		done
		cp $REPO_PATH/.git $TEMP/
		log "check restore from diffs"
		diff --brief --recursive $REPO_PATH $TEMP \
		|| log "git patchs restore do not match source"
		rm -rf $TEMP

		log "restore from backup"
		TEMP=$(mktemp -d)
		$DNA_BACKUP restore -v 2 $BACKUP $TEMP
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
