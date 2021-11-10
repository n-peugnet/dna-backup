#!/bin/bash

# Copyright (C) 2021 Nicolas Peugnet <n.peugnet@free.fr>

# This file is part of dna-backup.

# dna-backup is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.

# dna-backup is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.

# You should have received a copy of the GNU General Public License
# along with dna-backup.  If not, see <https://www.gnu.org/licenses/>. */

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
# - BORG: the path to borg dir
# - GIT_NOPACK: the path of the git nopack dir

log() {
	echo -e "\033[90m$(date +%T.%3N)\033[0m" $*
}

set-git-dir() {
	echo gitdir: $1 > $REPO_PATH/.git
}

GITC="git -C $REPO_PATH"
OUT=/tmp/dna-backup-exp-out

if [ -n "$GIT_NOPACK" ]
then
	# Init git nopack dir
	rm $REPO_PATH/.git
	$GITC init --separate-git-dir=$GIT_NOPACK
	$GITC --git-dir=$GIT_NOPACK config gc.auto 0
	set-git-dir $GIT_PATH
fi

if [ -n "$BORG" ]
then
	# Init borg dir
	borg init -e none $BORG
fi
if [ -n "$DNA_PARAMS" ]
then
	# Clean dna backups for this version
	while read name flags
	do
		rm -rf $name
	done < $DNA_PARAMS
fi


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

	if [ -n "$REAL" ]
	then
		# Save real size for this version
		log "save real size for this version"
		du -b --summarize $REPO_PATH > $(printf "%s.versions/%05d" $REAL $i)
	fi

	if [ -n "$TARGZ" ]
	then
		# Create tar.gz for this version
		log "create targ.gz for this version"
		tar -czf $(printf "%s/%05d.tar.gz" $TARGZ $i) $REPO_PATH
	fi

	if [ -n "$DIFFS" ]
	then
		# Create git diff for this version
		log "create git diff for this version"
		diff=$(printf "%s/%05d.diff.gz" $DIFFS $i)
		$GITC diff --minimal --binary --unified=0 -l0 $prev \
		| gzip \
		> $diff
	fi

	if [ -n "$GIT_NOPACK" ]
	then
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
	fi

	if [ -n "$BORG" ]
	then
		# Create borg backup for this versions
		log "create borg backup for this versions"
		borg create $BORG::$i $REPO_PATH
		find $BORG/data -type f -exec du -ba {} + \
		| cut -f1 \
		| paste -sd+ \
		| bc \
		> $(printf "%s.versions/%05d" $BORG $i)
	fi


	if [ -n "$DNA_PARAMS" ]
	then
		# Create dna backups for this version
		while read name flags
		do
			log "create $name backup for this version"
			$DNA_BACKUP commit -v 2 $flags $REPO_PATH $name

			log "create $name export for this version"
			export=/tmp/dna-backup-exp-export
			rm -rf $export
			$DNA_BACKUP export -v 2 $flags $name $export
			find $export -type f -exec du -ba {} + \
			| cut -f1 \
			| paste -sd+ \
			| bc \
			> $(printf "%s_export.versions/%05d" $name $i)
		done < $DNA_PARAMS
	fi

	if [[ $(( $i % $SKIP_CHECK )) == 0 ]]
	then

		if [ -n "$DIFFS" ]
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
		fi


		if [ -n "$DNA_PARAMS" ]
		then
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
	fi

	prev=$hash
	let i++
done

# cleanup
log "clean up $REPO_PATH"
$GITC checkout $last 2> $OUT \
|| (log "error checking out back to latest commit"; cat $OUT; exit 2)
