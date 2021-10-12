# Performance evaluation experiences

## Help

```bash
# run experiences
make [SKIP_COMMITS=<count>] [MAX_VERSION=<count>] [RANGE=<range>]

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
- `MAX_VERSION` = 5
- `RANGE` = daily
