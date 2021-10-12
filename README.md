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

Pour évaluer les performances du système DNA-Backup, trois autres systèmes de
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
<th>Déduplication</th>
<td><ul><li>Niveau chunk</li><li>Transversal aux versions</li></ul></td>
<td>Aucune</td>
<td><ul><li>Niveau fichier</li><li>Transversal aux versions</li></ul></td>
<td>Aucune</td>
<td>Aucune</td>
</tr>
<tr>
<th>Encodage delta</th>
<td><ul><li>Niveau chunk</li><li>Transversal aux versions</li></ul></td>
<td><ul><li>Niveau version</li><li>Par rapport à la précédente</li></ul></td>
<td>Aucun</td>
<td>Aucun</td>
<td>Aucun</td>
</tr>
<tr>
<th>Compression</th>
<td>Niveau chunk</td>
<td>Niveau version</td>
<td>Niveau fichier</td>
<td>Niveau version</td>
<td>Aucune</td>
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
      dna_4k       dna_8k        diffs       nopack        targz           real
  66,399,072   60,446,750   47,304,261   63,594,844   47,877,584    202,628,344
  21,873,958   24,422,316    9,911,248   43,598,822   50,529,069    214,276,336
       4,724        4,747        1,175       26,968   50,529,943    214,278,164
     347,460      242,271    9,912,728       76,072   47,893,781    202,636,078
     159,814       16,899        3,307       56,203   47,894,249    202,639,214
   2,918,706    2,762,164    9,911,148       41,606   50,531,041    214,283,553
     349,216      253,022    9,910,361      139,911   47,890,923    202,641,133
   2,865,631    2,794,693    9,912,341       69,243   50,530,875    214,285,297
     233,171        2,504          214        9,789   50,531,578    214,285,426
     343,483      234,834    9,914,620       13,703   47,894,718    202,641,398
     833,810      871,476      266,905    4,764,839   47,843,104    202,455,083
     237,510      270,660      142,110    1,976,823   47,793,819    202,272,761
     167,305       87,432       18,611        6,256   47,791,775    202,244,796
         113          140          289        2,009   47,791,778    202,244,779
   2,930,957    2,796,716   10,165,351       64,393   50,534,975    214,286,010
      20,364       32,883        7,841       48,219   50,533,061    214,283,569
     351,923      235,867    9,923,334      117,287   47,896,130    202,642,396
================================ TOTAL ================================
 100,037,217   95,475,374  127,305,844  114,606,987  832,288,403  3,525,024,337 
```

Commits hebdomadaires :

```
=============================== SUMMARY ===============================
        dna_4k         dna_8k          diffs         nopack           real
    70,192,809     63,852,374     49,917,523     67,132,003    214,292,720
        31,567         28,668          8,822         90,423    214,301,810
        27,389         31,485         10,920         99,194    214,299,953
    18,135,507     20,861,135      9,918,650     40,011,293    202,624,903
       907,939      1,209,389        285,459      4,920,733    202,470,023
       113,871        152,351        293,731        137,519    202,618,267
       294,810        367,701        272,263      2,304,224    203,092,308
     2,112,921      2,540,859      1,148,513      9,636,016    201,476,675
       252,068        288,241        609,369        857,331    202,282,568
       782,812        981,296        697,995      2,758,951    201,420,809
       136,493        161,325        398,494        727,346    202,065,360
        62,677         80,290        134,441        458,130    202,251,722
       162,061        196,716        365,229        230,404    202,465,009
         7,665          9,678         10,625         77,034    202,457,471
        71,731         80,298        152,999        187,241    202,615,704
       307,109        222,474        241,092         12,081    203,083,912
       305,795        228,540         35,494        740,246    203,113,279
================================ TOTAL ================================
    93,905,224     91,292,820     64,501,619    130,380,169  3,476,932,493 
```

Commits mensuels :

```
=============================== SUMMARY ===============================
        dna_4k         dna_8k          diffs         nopack           real
    66,344,139     60,414,327     47,255,410     63,531,013    202,455,244
       268,382        293,432         71,579      2,114,221    202,438,437
       288,294        288,397        137,081      2,625,834    202,477,165
     2,617,048      2,989,196      1,106,365     11,273,622    203,355,330
     4,219,402      5,065,795      1,485,211     14,062,635    206,087,365
     6,925,148      8,177,404      3,102,478     20,489,609    209,450,906
     1,931,351      2,314,294        771,998      6,811,409    209,646,120
     9,775,191     11,577,926      3,335,990     26,532,154    213,287,798
     7,783,071      9,101,660      2,505,353     20,687,252    216,420,188
     9,445,609     10,977,253      3,479,709     25,758,937    217,852,953
       701,911        905,423        164,682      4,517,360    217,851,223
    14,385,992     16,467,969      4,380,280     32,949,448    222,875,080
     3,389,340      4,347,527        817,894     14,054,849    223,352,903
    13,307,722     15,446,179      4,060,874     32,889,854    225,760,003
     3,219,293      3,895,349      1,301,487     10,953,334    225,577,911
     1,876,709      2,451,988        390,110      9,171,030    225,848,365
    12,995,018     15,561,939      4,204,779     32,837,755    227,575,213
================================ TOTAL ================================
   159,473,620    170,276,058     78,571,280    331,260,316  3,652,312,204 
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
