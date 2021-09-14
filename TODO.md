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

priority 2
----------
- [ ] use more the `Reader` API (which is analogous to the `IOStream` in Java)
- [ ] refactor `matchStream` as right now it is quite complex
- [x] better test for `(*Repo).matchStream`
- [ ] compress partial chunks (`TempChunks` for now)
- [ ] tail packing of `PartialChunks` (this Struct does not exist yet as it is in
    fact just `TempChunks` for now)
- [ ] option to commit without deltas to save new base chunks
- [ ] custom binary marshal and unmarshal for chunks
- [x] use `loadChunkContent` in `loadChunks`
- [ ] TODO: store hashes for faster maps rebuild
- [ ] try [Fdelta](https://github.com/amlwwalker/fdelta) and
    [Xdelta](https://github.com/nine-lives-later/go-xdelta) instead of Bsdiff
- [ ] maybe use an LRU cache instead of the current FIFO one.
- [x] remove `LoadedChunk` and only use `StoredChunk` instead now that the cache
    is implemented
- [ ] store file list compressed

reunion 7/09
------------
- [ ] save recipe consecutive chunks as extents
- [ ] store recipe and files incrementally
- [ ] compress recipe
- [ ] make size comparison between recipe and chunks with some datasets

ideas
-----
1. Would it be a good idea to store the compressed size for each chunk?
    Maybe this way we could only load the chunks needed for each file read.

2. Implement the `fs` interface of Go? Not sure if this will be useful.

3. If we don't need to reduce read amplification we could compress all chunks if
    it reduces the space used.
