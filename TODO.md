priority 1
----------
- add deltaEncode chunks function
    - do not merge consecutive smaller chunks as these could be stored as chunks if no similar chunk is found. Thus it will need to be of `chunkSize` or less. Otherwise it could not be possibly used for deduplication.
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
- read from repo
    - store recipe
    - load recipe
    - read chunks in-order into a stream
- properly store informations to be DNA encoded

priority 2
----------
- use more the `Reader` API (which is analoguous to the `IOStream` in Java)
- refactor matchStream as right now it is quite complex
