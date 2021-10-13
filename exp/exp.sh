#!/bin/bash

# This script expects the following variables to be exported:
# - DNA_BACKUP: the path to dna-backup binary
# - DNA_PARAMS: the path of the files that desscribes the multiple parameters to test
# - REPO_PATH: the path of the repo the experiment is based on
# - GIT_PATH: the path of the repo git-dir
# - MAX_VERSION: the max number for versions for the experiment
# - SKIP_CHECK: the number of versions to skip checking
# - COMMITS: the name of the file that contains the lists of versions
# - TARGZ: the path of the tar.gz dir
# - DIFFS: the path of the git diff dir
# - REAL: the path of the real size dir
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

# "empty tree" commit
prev="4b825dc642cb6eb9a060e54bf8d69288fbee4904"
last=$(tail --lines=1 $COMMITS | cut -f1)

i=0
head -n $MAX_VERSION $COMMITS | while read line
do
	# Get hash
	hash=$(echo "$line" | cut -f1)

	# Check out repo
	log "check out $hash"
	$GITC checkout $hash 2> $OUT \
	|| (log "error checking out"; cat $OUT; exit 1)

	# Save real size for this version
	log "save real size for this version"
	du -b --summarize $REPO_PATH > $(printf "%s.versions/%05d" $REAL $i)

	# Create tar.gz for this version
	log "create targ.gz for this version"
	tar -czf $(printf "%s/%05d.tar.gz" $TARGZ $i) $REPO_PATH

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
	> $(printf "%s.versions/%05d" $GIT_NOPACK $i)
	set-git-dir $GIT_PATH

	# Create dna backups for this version
	cat $DNA_PARAMS | while read name flags
	do
		log "create $name backup for this version"
		$DNA_BACKUP commit -v 2 $flags $REPO_PATH $name
	done

	if [[ $(( $i % $SKIP_CHECK )) == 0 ]]
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


		# Check restore from dna backups
		cat $DNA_PARAMS | while read name flags
		do
			log "restore from $name backup"
			TEMP=$(mktemp -d)
			$DNA_BACKUP restore -v 2 $flags $name $TEMP
			log "check restore from backup"
			diff --brief --recursive $REPO_PATH $TEMP \
			|| log "dna backup restore do not match source"
			rm -rf $TEMP
		done
	fi

	prev=$hash
	let i++
done

# cleanup
log "clean up $REPO_PATH"
$GITC checkout $last 2> $OUT \
|| (log "error checking out back to latest commit"; cat $OUT; exit 2)
