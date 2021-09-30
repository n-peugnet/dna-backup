#!/bin/bash

commits_file=$1
repo_path=$2

temp_templ="dna-backup-bench-XXXXX"
mktemp="mktemp --tmpdir -d $temp_templ"

temp=$($mktemp)
echo temp dir: $temp >&2

cat $commits_file | while read i
do
	hash=$(echo "$i" | cut -f1)
	git -C $repo_path checkout $hash
	../dna-backup commit -v 2 $repo_path $temp
done
du -bad 2 $temp
