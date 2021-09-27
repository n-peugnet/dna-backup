priority 1
----------
- [x] add `deltaEncode` chunks function
    - [x] do not merge consecutive smaller chunks as these could be stored as
        chunks if no similar chunk is found. Thus, it will need to be of
        `chunkSize` or less. Otherwise, it could not be possibly used for
        deduplication.
- [ ] read individual files
- [ ] properly store information to be DNA encoded
    - [ ] tar source to keep files metadata ?
    - [x] store chunks compressed
        - [x] compress before storing
        - [x] decompress before loading
    - [ ] store compressed chunks into tracks of `trackSize` (1024o)
- [x] add chunk cache to uniquely store chunks in RAM
- [x] better tests for `(*Repo).Commit`
- [x] remove errored files from `fileList`
- [ ] **TODO: Priority 3** add superblock logic.
- [ ] **TODO: Priority 2** add version blocks or journal logic.
- [x] command line with subcommands (like, hmm... git ? for instance).

priority 2
----------
- [ ] use more the `Reader` API (which is analogous to the `IOStream` in Java)
- [ ] refactor `matchStream` as right now it is quite complex
- [x] better test for `(*Repo).matchStream`
- [ ] tail packing of `PartialChunks` (this Struct does not exist yet as it is
    in fact just `TempChunks` for now).
    This might not be useful if we store the recipe incrementally.
- [ ] option to commit without deltas to save new base chunks.
    This might not be useful if we store the recipe incrementally.
- [ ] custom binary marshal and unmarshal for chunks
- [x] use `loadChunkContent` in `loadChunks`
- [x] save hashes for faster maps rebuild
    - [x] store hashes for current version's chunks
    - [x] load hashes for each version
- [x] use store queue to asynchronously store `chunkData`
- [ ] try [Fdelta](https://github.com/amlwwalker/fdelta) and
    [Xdelta](https://github.com/nine-lives-later/go-xdelta) instead of Bsdiff
- [ ] maybe use an LRU cache instead of the current FIFO one.
- [x] remove `LoadedChunk` and only use `StoredChunk` instead now that the cache
    is implemented
- [ ] keep hash workers so that they reuse the same hasher and reset it instead
    of creating a new one each time. This could save some processing time
- [ ] support links (symbolic mainly and also hard)
    - [ ] store this metadata somewhere, tar could be the solution, but this
        would bury the metadata down into the chunks, storing it into the files
        listing could be another solution but with this approach we would have
        to think about what other metadata we want to store
    - [ ] use a symlink aware Walk function (easy enough)
- [ ] add quick progress bar to CLI

reunion 7/09
------------
- [ ] save recipe consecutive chunks as extents
- [x] store recipe incrementally.
- [x] store file list incrementally.
- [x] compress recipe
- [x] compress file list
- [ ] make size comparison between recipe and chunks with some datasets

ideas
-----
1. Would it be a good idea to store the compressed size for each chunk?
    Maybe this way we could only load the chunks needed for each file read.

2. Implement the `fs` interface of Go? Not sure if this will be useful.

3. If we don't need to reduce read amplification we could compress all chunks 
    together if it reduces the space used.

mystical bug 22/09
------------------

On the second run, delta chunks can be encoded against better matching chunks as
more of them are present in the `sketchMap`. But we don't want this to happen,
because this adds data to write again, even if it has already been written.

Possible solutions :

- keep IDs for delta chunks, calculate a hash of the target data and store it in
    a new map. Then, when a chunk is encoded, first check if it exists in
    the fingerprint map, then in the delta map, and only after that check for
    matches in the sketch map.
    This should also probably be used for `TempChunks` as they have more chance
    to be delta-encoded on a second run.
- wait the end of the stream before delta-encoding chunks. So if it is not found
    in the fingerprints map, but it is found in the sketch map, then we wait to
    see if we found a better candidate for delta-encoding.
    This would not fix the problem of `TempChunks` that become delta-encoded on
    the second run. So we would need IDs and a map for these. Tail packing
    `TempChunks` could also help solve this problem
    (see [priority 2](#priority-2)).

The first solution would have an advantage if we were directly streaming the
output of the program into DNA, as it could start DNA-encode it from the first
chunk. The second solution will probably have better space-saving performance as
waiting for better matches will probably lower the size of the patches.

This has been fixed by making multiple passes until no more blocks are added,
this way we are assured that the result will be the same on the following run.
