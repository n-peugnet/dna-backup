#!/bin/bash

# This script expects the following variables to be exported:
# - REPO_PATH: the path of the repo the experiment is based on
# - MAX_VERSION: the max number for versions for the experiment
# - LO_DEV: the loopbak device used to mount the UDF dir
# - COMMITS: the name of the file that contains the lists of versions
# - BACKUP: the path fo the dna-backup dir
# - DIFFS: the path of the git diff dir
# - UDF: the path of the UDF dir

GITC="git -C $REPO_PATH"

# "empty tree" commit
prev="4b825dc642cb6eb9a060e54bf8d69288fbee4904"
last=$(tail --lines=1 $COMMITS | cut -f1)

sudo mount -t udf -o loop $LO_DEV $UDF
sudo chown $USER:$USER $UDF

i=0
cat $COMMITS | while read line
do
	hash=$(echo "$line" | cut -f1)
	$GITC checkout $hash

	# create diff for this version
	$GITC diff --minimal --binary --unified=0 $prev | gzip > $DIFFS/$i.diff.gz

	# create backup for this version
	../dna-backup commit -v 2 $REPO_PATH $BACKUP

	# write to udf
	$GITC diff | git -C $UDF apply -
	sudo umount $UDF

	prev=$hash
	let i++
	if [[ $i == $MAX_VERSION ]]
	then
		break
	fi
done

# cleanup
$GITC checkout $last
