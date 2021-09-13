priority 1
----------
- [x] add `deltaEncode` chunks function
    - [x] do not merge consecutive smaller chunks as these could be stored as
        chunks if no similar chunk is found. Thus, it will need to be of
        `chunkSize` or less. Otherwise, it could not be possibly used for
        deduplication.
    ```
    for each new chunk:
        find similar in sketchMap
        if exists:
            delta encode
        else:
            calculate fingerprint
            store in fingerprintMap
            store in sketchMap
    ```
- [x] read from repo (Restore function)
    - [x] store recipe
    - [x] load recipe
    - [x] read chunks in-order into a stream
- [ ] read individual files
- [ ] properly store information to be DNA encoded
    - [ ] tar source to keep files metadata ?
    - [x] store chunks compressed
        - [x] compress before storing
        - [x] decompress before loading
    - [ ] store compressed chunks into tracks of `trackSize` (1024o)
- [x] add chunk cache... what was it for again ??
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
- [ ] use `loadChunkContent` in `loadChunks`
- [ ] store hashes for faster maps rebuild

reunion 7/09
------------
- [ ] save recipe consecutive chunks as extents
- [ ] store recipe and files incrementally
- [ ] compress recipe
- [ ] make size comparison between recipe and chunks with some datasets
