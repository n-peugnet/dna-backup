#!/bin/bash

commits_file=$1
repo_path=$2
temp=$3

cat $commits_file | while read i
do
	hash=$(echo "$i" | cut -f1)
	git -C $repo_path checkout $hash
	../dna-backup commit -v 2 $repo_path $temp
done
