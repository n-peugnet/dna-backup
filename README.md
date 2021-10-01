# DNA Backup

[![build][build-img]][build-url]

_Deduplicated versioned backups for DNA._

## Details (FR)
<!-- LTeX: language=fr -->
Le système part du principe qu'on a une copie des données stockées en ADN
sur un support de stockage classique : le _repo_.

```
 +----------------------------------+
 | +---------+          +---------+ |          +-----------+
 | |         |          |         | |          |           |
 | | Source  |--------->|  Repo   |----------->| DNA-Drive |
 | |         |  COMMIT  |         | | SYMTHESE |           |
 | +---------+          +---------+ |          +-----------+
 |                                  |
 | Ordinateur                       |
 +----------------------------------+
```

La structure du _repo_ est la suivante :

```
repo/
├── 00000/
│   ├── chunks/
│   │   ├── 000000000000000
│   │   ├── 000000000000001
│   │   ├── 000000000000002
│   │   └── 000000000000003
│   ├── files
│   ├── hashes
│   └── recipe
└── 00001/
    ├── chunks/
    │   ├── 000000000000000
    │   └── 000000000000001
    ├── files
    ├── hashes
    └── recipe
```

Pour un repo d'une taille totale de 401 Mio :

```
/tmp/test-1/00000/recipe    5076011  (1.20%)
/tmp/test-1/00000/files	      24664  (0.06%)
/tmp/test-1/00000/hashes    3923672  (0.93%)
/tmp/test-1/00000/chunks  412263137  (97.8%)
/tmp/test-1/00000         421287604  ( 100%)

```

-   On considère que le _repo_ est toujours présent lors d'une écriture (_commit_).
-   Le _repo_ peut être reconstruit à partir des données présentes dans le
    _DNA-Drive_.
-   Les _hashes_ ne sont pas écrits en ADN, car ils peuvent être reconstruits à
    partir des données des _chunks_.
-   L'ensemble des données écrites en ADN sont compressées, pour le moment via
    _ZLib_.
-   Les métadonnées sont stockées de manière incrémentale, chaque version stocke
    donc ses métadonnées sous la forme de delta par rapport à la version
    précédente.

On imagine le _DNA-Drive_ comme un segment de _pools_ :

```
     +---+---+---+---+---+---------+---+---+---+
     | 0 | 1 | 2 | 3 | 4 |-->   <--| 61| 62| 63|
     +---+---+---+---+---+---------+---+---+---+
|versions|    chunks     |         | metadata  |
                                   (recipe+files)
```

### Commit algorithme

1.  Chargement des métadonnées du _repo_ afin de reconstruire en mémoire l'état
    de la dernière version :
    -   Reconstruction de la _recipe_ à partir des deltas de chaque version.
    -   Reconstruction du listage des fichiers à partir des deltas de chaque
        version (fichier _files_).
    -   Reconstruction en mémoire des _maps_ de _fingerprints_ et de _sketches_
        à partir des fichiers _hashes_ de chaque version.
2.  Listage des fichiers de la _source_.
3.  Concaténation de l'ensemble des fichiers de la source en un disque virtuel
    continu.
4.  Lecture du _stream_ de ce disque virtuel et découpage en _chunk_ (de 8 Kio
    actuellement).
5.  Pour chaque _chunk_ du _stream_ :
    1.  Calculer sa _fingerprint_ (hash classique), si elle est présente dans la
        _map_ : le stocker de manière dé-dupliquée (sous la forme d'identifiant
        faisant référence au _chunk_ trouvé dans la map).
    2.  Sinon, calculer son _sketch_ (hash de ressemblance),
        s'il est présent dans la _map_, le stocker sous la forme de delta (calcul
        de sa différence par rapport au _chunk_ trouvé dans la map).
    3.  Sinon, le stocker sous la forme de nouveau bloc (ajout
        de sa _fingerprint_ et de son _sketch_ dans les _maps_ et stockage du
        contenu complet dans un nouveau _chunk_).
6.  Calcul des différences entre la nouvelle version et la précédente pour les
    métadonnées (_files_ et _recipe_) et stockage des deltas ainsi obtenus.

### Restore algorithme

1.  Chargement des métadonnées du _repo_ afin de reconstruire en mémoire l'état
    de la dernière version :
    -   Reconstruction de la _recipe_ à partir des deltas de chaque version.
    -   Reconstruction du listage des fichiers à partir des deltas de chaque
        version.
2.  À partir de la _recipe_, reconstruire le disque virtuel (sous la forme d'un
    _stream_).
3.  Découper ce _stream_ en fonction du listage des fichiers (_files_) et
    réécrire les données dans les fichiers correspondants dans le répertoire
    _destination_.

### Restaurer sans le _repo_

#### Reconstruction complète du _repo_

Il est possible de reconstruire le _repo_ en entier en lisant la totalité du
_DNA-Drive_.

#### Restauration de la dernière version

Il est possible de ne restaurer que la dernière version en lisant dans un
premier temps le _pool_ de versions et les quelques _pools_ de métadonnées
(environ 2% de la totalité des données écrites), puis en lisant tous les _pools_
contenant des _chunks_ référencés par la _recipe_ de cette version.

#### Restauration d'un seul fichier

Il pourrait être possible (pas pour le moment) de ne restaurer qu'un seul fichier
d'une version en ayant moins de données à lire que pour restaurer la version
complète.

Pour cela, il faudrait en plus stocker en ADN un mapping _chunk_ décompressé →
_pool_ contenant ce _chunk_ et ainsi n'avoir à lire que les _pools_ contenant
des _chunks_ de ce fichier.

<!-- LTeX: language=en -->
## Build instructions

### Requirements

- Go >= 1.16

### Instructions

```bash
# Build
make build

# Test
make test

# Run
./dna-backup commit <source-dir> <repository>
```

[build-img]: https://github.com/n-peugnet/dna-backup/actions/workflows/build.yml/badge.svg
[build-url]: https://github.com/n-peugnet/dna-backup/actions/workflows/build.yml
