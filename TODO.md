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
- [ ] command line with subcommands (like, hmm... git ? for instance).

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

reunion 7/09
------------
- [ ] save recipe consecutive chunks as extents
- [ ] **TODO: Priority 1** store recipe incrementally.
    - [x] patch and diff for recipes
    - [ ] store recipe updates per version.
    - [ ] load all recipes incrementally.
- [ ] **TODO: Priority 2** store file list incrementally.
    - [ ] patch and diff for slices
    - [ ] store file lists updates per version.
    - [ ] load all file lists incrementally.
- [x] compress recipe
- [x] compress file list
- [ ] make size comparison between recipe and chunks with some datasets

ideas
-----
1. Would it be a good idea to store the compressed size for each chunk?
    Maybe this way we could only load the chunks needed for each file read.

2. Implement the `fs` interface of Go? Not sure if this will be useful.

3. If we don't need to reduce read amplification we could compress all chunks if
    it reduces the space used.
