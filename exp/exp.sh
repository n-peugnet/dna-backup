#!/bin/bash

# This script expects the following variables to be exported:
# - DNA_BACKUP: the path to dna-backup binary
# - REPO_PATH: the path of the repo the experiment is based on
# - MAX_VERSION: the max number for versions for the experiment
# - COMMITS: the name of the file that contains the lists of versions
# - BACKUP: the path fo the dna-backup dir
# - DIFFS: the path of the git diff dir

GITC="git -C $REPO_PATH"

# "empty tree" commit
prev="4b825dc642cb6eb9a060e54bf8d69288fbee4904"
last=$(tail --lines=1 $COMMITS | cut -f1)

i=0
cat $COMMITS | while read line
do
	hash=$(echo "$line" | cut -f1)
	$GITC checkout $hash

	# create diff for this version
	$GITC diff --minimal --binary --unified=0 $prev \
	| gzip \
	> "$DIFFS/$i.diff.gz"

	# create backup for this version
	$DNA_BACKUP commit -v 2 $REPO_PATH $BACKUP

	if [[ $(( $i % 4 )) == 0 ]]
	then
		# check diff correctness
		TEMP=$(mktemp -d)
		for n in $(seq 0 $i)
		do
			cat "$DIFFS/$n.diff.gz" \
			| gzip --decompress \
			| git -C $TEMP apply --binary --unidiff-zero --whitespace=nowarn - \
			|| echo "Git patchs do not match source"
		done
		cp $REPO_PATH/.git $TEMP/
		diff --brief --recursive $REPO_PATH $TEMP \
		|| echo "dna-backup restore do not match source"
		rm -rf $TEMP

		# check backup correctness
		TEMP=$(mktemp -d)
		$DNA_BACKUP restore -v 2 $BACKUP $TEMP
		diff --brief --recursive $REPO_PATH $TEMP
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
$GITC checkout $last
