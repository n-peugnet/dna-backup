priority 1
----------
- [x] add deltaEncode chunks function
    - [x] do not merge consecutive smaller chunks as these could be stored as chunks if no similar chunk is found. Thus it will need to be of `chunkSize` or less. Otherwise it could not be possibly used for deduplication.
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
- [ ] read from repo
    - [x] store recipe
    - [x] load recipe
    - [ ] read chunks in-order into a stream
    - [ ] read individual files
- [ ] properly store informations to be DNA encoded
    - [ ] tar source to keep files metadata ?
    - [ ] store chunks compressed
        - [ ] compress before storing
        - [ ] uncompress before loading
    - [ ] store compressed chunks into tracks of trackSize (1024o)
- [x] add chunk cache... what was it for again ??

priority 2
----------
- [ ] use more the `Reader` API (which is analoguous to the `IOStream` in Java)
- [ ] refactor matchStream as right now it is quite complex
- [ ] better test for `Repo.matchStream`
- [ ] tail packing of PartialChunks (this Struct does not exist yet as it is in fact just `TempChunks` for now)
- [ ] option to commit without deltas to save new base chunks

réunion 7/09
------------
- [ ] save recipe consecutive chunks as extents
- [ ] store recipe and files incrementally
- [ ] compress recipe
- [ ] make size comparision between recipe and chunks with some datasets
