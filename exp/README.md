# Performance evaluation experiences

## Help

```bash
# run experiences
make [SKIP_COMMITS=<count>] [MAX_VERSION=<count>] [COMMITS=<file>]

# clean results
make mostlyclean

# clean all
make clean
```

Les 3 fichiers de commits suivants peuvent être automatiquement générés :

- `commits.daily`
- `commits.weekly`
- `commits.monthly`

Par défaut les 30 premiers commits journaliers sont skippés, le nombre
max de version utilisés pour l'expérience est 5 et le fichier
`commits.daily` est utiliser comme source de commits.
