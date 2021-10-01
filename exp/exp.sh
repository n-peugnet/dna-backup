#!/bin/bash

commits="$1"
repo="$2"
max_count="$3"
backup="$4"
diffs="$5"

mkdir -p $backup $diffs

# "empty tree" commit
prev="4b825dc642cb6eb9a060e54bf8d69288fbee4904"
last=$(tail --lines=1 "$commits" | cut -f1)

i=0
cat "$commits" | while read line
do
	hash=$(echo "$line" | cut -f1)
	git -C "$repo" checkout "$hash"

	# create diff for this version
	git -C "$repo" diff --minimal --binary --unified=0 "$prev" | gzip > "$diffs/$i.diff.gz"

	# create backup for this version
	../dna-backup commit -v 2 "$repo" "$backup"

	prev="$hash"
	let i++
	if [[ $i == $max_count ]]
	then
		break
	fi
done

git -C "$repo" checkout "$last"
