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
 | |         |  COMMIT  |         | | SYNTHESE |           |
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
     | 0 | 1 | 2 | 3 | 4 |-->   <--| 93| 94| 95|
     +---+---+---+---+---+---------+---+---+---+
|versions|    chunks     |         | metadata  |
                                   (recipe+files)
```

### Algorithme du commit

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

### Algorithme du restore

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

## Évaluation de performances

Le dossier `exp` contient les scripts permettant de reproduire les expériences.
Les scripts ne sont prévus pour fonctionner que sur Linux.

On utilise le dépôt Git du noyau Linux comme base de donnée de test. Il s'agit
en effet d'une bonne simulation de modification de dossiers, car l'historique
contient toutes les modifications qui ont été apportées petit à petit à
l'ensemble des fichiers. 

### Bases de comparaison

Pour évaluer les performances du système DNA-Backup, quatre autres systèmes de
stockage versionnés ont été choisis comme base de comparaison :

- **Git diffs**
- **Git objects**
- **Targz**
- **Taille réelle**

#### Git diffs

Ce système utilise le delta généré par la commande `git diff` pour sauvegarder
une nouvelle version. Les données à stocker consistent donc en une somme de
deltas. Pour restaurer les données, il faut appliquer séquentiellement
l'ensemble des deltas jusqu'à obtenir l'état de la version voulue.

#### Git objects

Ce système nous permet de simuler un système de fichier qui ne serait pas
autorisé à modifier des données sur le support tout en gardant la possibilité de
modifier les données.
Il s'agit de la manière dont Git sauvegarde les données des fichiers d'un dépôt.
Le contenu de chaque fichier et de chaque dossier est hashé afin d'en obtenir
une signature. Il est ensuite compressé et stocké sous la forme d'_object_
immuable, référencé par la signature obtenue.
Si un fichier est modifié, il produira une signature différente et sera donc
stocké sous la forme d'un nouvel _object_.
Par contre, si deux fichiers ont un contenu strictement identique, ils produiront
alors la même signature et seront donc automatiquement dé-dupliqués.
Les dossiers sont également stockés en tant qu'_objects_, mais les fichiers
qu'ils contiennent sont référencés non pas par leur nom, mais par leur signature.
La modification d'un fichier entrainera donc l'ajout de nouveaux _objects_ pour
l'ensemble des dossiers de la branche contenant ce fichier.
C'est de cette manière que Git est capable de créer un système de fichiers
modifiable à partir d'objets immuables.

#### Targz

Une technique d'archivage assez classique à laquelle il peut être intéressant de
nous comparer est de stocker chaque version en tant qu'une nouvelle archive Tar
elle-même compressée à l'aide de Gzip. Cette technique produit des archives
d'une taille très réduite, car la compression est appliquée à l'ensemble des
fichiers d'un seul coup, contrairement à une compression fichier par fichier.

Elle a cependant l'inconvénient de ne pas faire de dé-duplication ni d'encodage
delta, et ne tire donc pas du tout parti des données déjà écrites sur le support.

#### Taille réelle

Cette base de comparaison n'est en réalité pas un système viable. Elle
correspond à la taille que prend en réalité le dossier _source_ au moment de la
sauvegarde.
C'est un indicateur qui permet de se rendre compte du poids que prendrait la
sauvegarde de multiples versions sans aucune déduplication ou compression.

#### Tableau récapitulatif

<table>
<tr>
<th>Feature\Système</th>
<th>DNA-Backup</th>
<th>Git diffs</th>
<th>Git objects</th>
<th>Targz</th>
<th>Taille réelle</th>
</tr>
<tr>
<th rowspan="2">Déduplication</th>
<td>Niveau chunk</td>
<td rowspan="2">❌</td>
<td>Niveau fichier</td>
<td rowspan="2">❌</td>
<td rowspan="2">❌</td>
</tr>
<tr>
<td>Transversal aux versions</td>
<td>Transversal aux versions</td>
</tr>
<tr>
<th rowspan="2">Encodage delta</th>
<td>Niveau chunk</td>
<td>Niveau version</td>
<td rowspan="2">❌</td>
<td rowspan="2">❌</td>
<td rowspan="2">❌</td>
</tr>
<tr>
<td>Transversal aux versions</td>
<td>Par rapport à la précédente</td>
</tr>

<tr>
<th>Compression</th>
<td>Niveau chunk</td>
<td>Niveau version</td>
<td>Niveau fichier</td>
<td>Niveau version</td>
<td>❌</td>
</tr>
<tr>
<th>Restauration de la dernière version</th>
<td>
Lecture des metadonnées puis des chunks de cette version
(répartis dans différents pools)
</td>
<td>Lecture de la totalité du DNA-Drive</td>
<td>
Lecture récursive des différents objets composant la version
(répartis dans différents pools)
</td>
<td>Lecture de la zone correspondant à la dernière version</td>
<td>Lecture de la zone correspondant à la dernière version</td>
</tr>
</table>

### Nombre d'octets par version

#### Légende

-   `dna_4K` : le système DNA-Backup avec des blocs de 4 Kio.
-   `dna_8K` : le système DNA-Backup avec des blocs de 8 Kio.
-   `diffs`  : une somme de diffs Git minimales Gzippées.
-   `nopack` : le dossier `objects de Git, contenant l'ensemble des données
    des fichiers et dossiers d'un dépôt.
-   `targz`  : une somme d'archives Tar Gzippées.
-   `real`   : le poids réel de chaque version et donc l'espace nécessaire à
    stocker l'ensemble des versions de manière non-dé-dupliquées.

#### Résultats

Commits journaliers :

```
=============================== SUMMARY ===============================
  4k_export    8k_export        diffs       nopack         targz           real
 46,080,540   46,021,380   47,011,621   63,237,214    47,582,036    201,476,675
      8,160       13,260        2,625       88,673    47,580,422    201,472,392
  6,453,540    8,091,660    3,270,798   26,668,802    48,699,094    206,006,230
    205,020      108,120          496       24,839    48,699,144    206,006,227
    214,200      121,380        1,475      318,053    48,699,266    206,006,571
    255,000      162,180    3,271,631      107,651    47,592,358    201,472,618
    393,720      358,020       99,337    2,758,951    47,579,804    201,420,809
     67,320       78,540      127,793      561,940    47,578,309    201,415,269
    155,040       75,480       19,221       10,035    47,590,552    201,467,847
    286,620      205,020      250,581    1,203,018    47,719,274    202,032,972
     39,780       38,760       19,555      550,478    47,721,098    202,042,129
    159,120       80,580          203       45,564    47,721,112    202,042,114
    182,580      115,260       12,419      284,765    47,725,696    202,057,720
     13,260       14,280        5,823       76,010    47,731,040    202,065,875
     23,460       28,560       13,370      528,743    47,738,360    202,078,613
     27,540       33,660       10,837      374,955    47,735,904    202,078,276
     68,340       81,600       69,707      498,918    47,771,130    202,216,013
================================ TOTAL ================================
 54,633,240   55,627,740   54,187,492   97,338,609   813,464,599  3,443,358,350
```

Commits hebdomadaires :

```
=============================== SUMMARY ===============================
  4k_export    8k_export        diffs       nopack         targz           real
 46,086,660   46,003,020   47,003,541   63,221,563    47,569,933    201,420,809
    701,760      820,080      395,080    6,358,050    47,723,749    202,065,875
  6,293,400    7,983,540    2,994,599   25,581,925    48,700,415    206,003,757
    206,040      109,140          407       50,815    48,700,637    206,003,795
    225,420      142,800        8,679      401,381    48,698,820    206,005,265
  1,299,480    1,707,480      579,422    6,943,222    48,733,791    206,098,060
    952,680    1,248,480      360,710    4,799,958    48,840,759    206,648,359
  1,425,960    1,831,920      738,359    4,983,831    48,892,096    206,834,840
  1,770,720    2,091,000    1,389,502    7,767,439    49,297,747    209,328,856
    479,400      727,260      146,129    2,899,286    49,331,055    209,479,362
    168,300      235,620       47,436    1,385,568    49,333,845    209,503,564
    134,640      236,640       37,183    1,808,603    49,338,373    209,509,777
     90,780      122,400       23,924    1,555,868    49,336,559    209,515,352
  3,088,560    3,953,520    1,404,256   11,037,484    49,933,159    211,878,380
  4,987,800    6,165,900    2,326,692   17,577,030    50,214,110    212,941,025
    993,480    1,378,020      304,617    6,594,520    50,293,382    213,254,405
    684,420      900,660      258,512    4,016,395    50,398,489    213,650,745
================================ TOTAL ================================
 69,589,500   75,657,480   58,019,048  166,982,938   835,336,919  3,540,142,226
```

Commits mensuels :

```
=============================== SUMMARY ===============================
  4k_export na_8k_export       diffs       nopack         targz           real
 47,297,400   47,244,360  48,249,466   64,900,653    48,828,605    206,662,692
  1,822,740    1,938,000   1,495,969    7,407,714    48,900,735    206,964,143
  1,525,920    1,808,460     797,390    9,856,043    49,326,511    209,515,646
  8,047,800    9,840,960   4,142,700   28,400,251    50,394,403    213,653,996
 10,730,400   13,230,420   5,489,832   34,132,686    51,315,648    217,862,957
  5,786,460    6,936,000   2,262,584   19,233,445    51,941,615    220,756,834
  7,816,260   10,320,360   2,999,817   28,983,950    52,574,107    223,306,219
  1,210,740    1,643,220     299,628    8,343,393    52,587,994    223,373,786
 11,002,740   13,589,460   4,759,088   34,259,652    53,210,823    226,113,059
  1,819,680    2,399,040     679,794   10,029,012    53,165,063    225,781,616
    622,200      858,840     138,547    4,375,159    53,183,197    225,870,650
 12,874,440   16,493,400   5,142,691   45,544,733    53,842,821    228,546,001
  1,169,940    1,591,200     247,526    8,491,133    53,876,401    228,653,615
  5,631,420    6,589,200   2,333,317   18,119,613    54,605,555    232,014,492
  9,988,860   12,876,480   3,989,065   37,945,661    55,206,806    234,571,285
 10,659,000   13,416,060   3,800,775   37,509,079    56,059,067    238,170,923
  8,796,480   11,079,240   3,030,148   32,387,325    56,716,443    241,420,002
================================ TOTAL ================================
146,802,480  171,854,700  89,858,337  429,919,502   895,735,794  3,803,237,916
```

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
