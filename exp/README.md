# Performance evaluation experiences

## Help

```bash
# run experiences
make [SKIP_COMMITS=<count>] [SKIP_CHECK=<count>] [MAX_VERSION=<count>] [RANGE=<range>]

# clean results
make mostlyclean

# clean all
make clean
```

`<range>` can be one of these: 
- `daily`
- `weekly`
- `monthly`

By default:

- `SKIP_COMMITS` = 0
- `SKIP_CHECK` = 4
- `MAX_VERSION` = 5
- `RANGE` = daily

It is possible to select which system will appear in the results by setting the folder of each one, by default:

```makefile
NOPACK = nopack
BORG   =
TARGZ  = targz
REAL   = real
DIFFS  = diffs
```

For instance, the following will run Borg as part of the benchmark and will not make git diffs:

```
make DIFFS= BORG=borg
```
