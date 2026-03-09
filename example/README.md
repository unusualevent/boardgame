# boardgame compression example

Walks a directory tree, compresses each text source file with boardgame,
and reports per-extension compression ratios, timing, and ASCII histograms.

## Usage

```
go run ./example /path/to/project
go run ./example -exclude vendor /path/to/project
go run ./example -include-vendored -max-size 0 -workers 8 /path/to/project
```

Flags:
- `-exclude` — additional directory name to skip
- `-include-vendored` — include `node_modules` and `vendor` (excluded by default)
- `-max-size` — maximum file size in bytes (default 20KB, 0 = unlimited)
- `-workers` — parallel compression workers (default: number of CPUs)

## Sample output

Ran against a mixed-language monorepo (~1100 text files, node_modules excluded,
max 20KB per file):

```
Extension         Files     Original   Compressed  Avg Ratio     Avg Time
------------------------------------------------------------------------
.backend              1          229          167      27.1%        787us
.bat                  1         4176         2321      44.4%      103.0ms
.cfg                  3         1134          853      24.3%        1.3ms
.code-workspace       1          926          331      64.3%        671us
.conf                 4          853          622      24.6%        878us
.css                 13        54931        24272      47.6%      102.1ms
.d                   18        22634         4446      74.1%       10.2ms
.dockerignore         1           71           57      19.7%         68us
.editorconfig         1          216          169      21.8%        607us
.example              3          830          634      19.2%        681us
.expr                 1          216          134      38.0%        476us
.gitattributes        1           19           18       5.3%          8us
.gitignore           22         4021         2988      21.7%        487us
.go                 307      1425566       669404      48.8%      118.8ms
.hcl                  4         5222         3440      23.4%       15.4ms
.html                23        20280        11630      29.2%        8.8ms
.j2                   5         2088         1265      40.2%        2.1ms
.java                 3        50732        12535      75.0%      414.6ms
.jpg                  4           61           58       4.5%          5us
.js                  42        78831        35586      44.4%       25.0ms
.json                98        41487        26217      30.8%        2.5ms
.kt                  57       205671        79861      56.3%       47.2ms
.kts                  8         6543         3282      40.3%        6.8ms
.list                 4          248          217      11.6%        326us
.lock                 1         3917         1519      61.2%       30.4ms
.md                  84       265960       152613      34.7%       62.1ms
.mf                   1           25           25       0.0%         25us
.mjs                  4        11434         5703      46.3%       50.7ms
.mod                 33        20923        10071      30.0%        3.5ms
.png                 24          386          362       6.2%          4us
.pro                  2          769          363      32.3%        2.0ms
.properties           6          939          629      26.0%        251us
.pub                  1          146          127      13.0%         50us
.rb                   1         5766         2671      53.7%      116.2ms
.rego                 1         1362          571      58.1%        9.2ms
.rs                   4        10224         3037      51.3%       20.2ms
.service              4         1336         1026      21.8%        829us
.sh                  10         7639         4600      35.9%        4.8ms
.sum                 21        88913        53914      36.3%       67.6ms
.svg                  7         7256         4716      33.8%        9.0ms
.tab                  2            6            8     -33.3%          4us
.tag                  2          354          284      19.8%        100us
.tf                   8         4884         2683      38.9%        2.7ms
.tfstate              1          182          135      25.8%        144us
.tfvars               1           25           23       8.0%          8us
.timestamp           28         1344         1204      10.4%         16us
.toml                 4         2240         1013      33.2%        3.1ms
.ts                  29        32053        16548      37.3%       20.5ms
.txt                  5        24566        18363      26.7%       64.1ms
.vue                 80       397941       219390      40.8%      133.9ms
.xml                 27        24800        10619      43.0%        8.0ms
.yaml                44        40856        16700      57.1%        5.2ms
.yml                 73        49656        24233      35.7%        4.8ms
------------------------------------------------------------------------
TOTAL              1133      2932887      1433657      40.0%       56.0ms
```

### Avg Compression Time vs Avg File Size (sorted by size)

```
.tab                 3B | ####                                     4us
.jpg                15B | ####                                     5us
.png                16B | ####                                     4us
.gitattributes      19B | ######                                   8us
.mf                 25B | #########                                25us
.tfvars             25B | ######                                   8us
.timestamp          48B | ########                                 16us
.list               62B | #################                        326us
.dockerignore       71B | #############                            68us
.pub               146B | ############                             50us
.properties        156B | #################                        251us
.tag               177B | ##############                           100us
.tfstate           182B | ###############                          144us
.gitignore         182B | ###################                      487us
.conf              213B | ####################                     878us
.expr              216B | ###################                      476us
.editorconfig      216B | ###################                      607us
.backend           229B | ####################                     787us
.example           276B | ####################                     681us
.service           334B | ####################                     829us
.cfg               378B | ######################                   1.3ms
.pro               384B | #######################                  2.0ms
.j2                417B | #######################                  2.1ms
.json              423B | ########################                 2.5ms
.toml              560B | ########################                 3.1ms
.tf                610B | ########################                 2.7ms
.mod               634B | #########################                3.5ms
.yml               680B | ##########################               4.8ms
.sh                763B | ##########################               4.8ms
.kts               817B | ###########################              6.8ms
.html              881B | ############################             8.8ms
.xml               918B | ###########################              8.0ms
.code-workspace     926B | ####################                     671us
.yaml              928B | ##########################               5.2ms
.svg              1.0KB | ############################             9.0ms
.ts               1.1KB | ##############################           20.5ms
.d                1.2KB | ############################             10.2ms
.hcl              1.3KB | #############################            15.4ms
.rego             1.3KB | ############################             9.2ms
.js               1.8KB | ###############################          25.0ms
.rs               2.5KB | ##############################           20.2ms
.mjs              2.8KB | #################################        50.7ms
.md               3.1KB | ##################################       62.1ms
.kt               3.5KB | #################################        47.2ms
.lock             3.8KB | ###############################          30.4ms
.bat              4.1KB | ###################################      103.0ms
.css              4.1KB | ###################################      102.1ms
.sum              4.1KB | ##################################       67.6ms
.go               4.5KB | ####################################     118.8ms
.txt              4.8KB | ##################################       64.1ms
.vue              4.9KB | ####################################     133.9ms
.rb               5.6KB | ####################################     116.2ms
.java            16.5KB | ######################################## 414.6ms
```

### Avg Compression Ratio vs Avg File Size (sorted by size)

```
.tab                 3B |                                          -33.3%
.jpg                15B | #                                        4.5%
.png                16B | ##                                       6.2%
.gitattributes      19B | ##                                       5.3%
.mf                 25B |                                          0.0%
.tfvars             25B | ###                                      8.0%
.timestamp          48B | ####                                     10.4%
.list               62B | ####                                     11.6%
.dockerignore       71B | #######                                  19.7%
.pub               146B | #####                                    13.0%
.properties        156B | ##########                               26.0%
.tag               177B | #######                                  19.8%
.tfstate           182B | ##########                               25.8%
.gitignore         182B | ########                                 21.7%
.conf              213B | #########                                24.6%
.expr              216B | ###############                          38.0%
.editorconfig      216B | ########                                 21.8%
.backend           229B | ##########                               27.1%
.example           276B | #######                                  19.2%
.service           334B | ########                                 21.8%
.cfg               378B | #########                                24.3%
.pro               384B | ############                             32.3%
.j2                417B | ################                         40.2%
.json              423B | ############                             30.8%
.toml              560B | #############                            33.2%
.tf                610B | ###############                          38.9%
.mod               634B | ############                             30.0%
.yml               680B | ##############                           35.7%
.sh                763B | ##############                           35.9%
.kts               817B | ################                         40.3%
.html              881B | ###########                              29.2%
.xml               918B | #################                        43.0%
.code-workspace     926B | #########################                64.3%
.yaml              928B | ######################                   57.1%
.svg              1.0KB | #############                            33.8%
.ts               1.1KB | ##############                           37.3%
.d                1.2KB | #############################            74.1%
.hcl              1.3KB | #########                                23.4%
.rego             1.3KB | #######################                  58.1%
.js               1.8KB | #################                        44.4%
.rs               2.5KB | ####################                     51.3%
.mjs              2.8KB | ##################                       46.3%
.md               3.1KB | #############                            34.7%
.kt               3.5KB | ######################                   56.3%
.lock             3.8KB | ########################                 61.2%
.bat              4.1KB | #################                        44.4%
.css              4.1KB | ###################                      47.6%
.sum              4.1KB | ##############                           36.3%
.go               4.5KB | ###################                      48.8%
.txt              4.8KB | ##########                               26.7%
.vue              4.9KB | ################                         40.8%
.rb               5.6KB | #####################                    53.7%
.java            16.5KB | ##############################           75.0%
```

### Observations

- **Overall**: 40.0% average compression across 1133 files (2.9MB -> 1.4MB)
- **UTF-8 support**: 21 more files now encodable (307 .go vs 295 before) since UTF-8 comments are DEL-escaped instead of rejected
- **New file types**: `.bat` (44.4%), `.rb` (53.7%), `.gitattributes` (5.3%), `.jpg`/`.png` (pass-through with minimal compression)
- **Time scales superlinearly**: a 5x file size increase costs ~30x more compression time
- **Ratio improves with size**: files under ~200B barely compress; larger files with more repeated patterns reach 50-75%
- **Best compressors**: `.java` (75%), `.d` (74%), `.code-workspace` (64%), `.yaml` (57%), `.kt` (56%)
- **Worst compressors**: `.tab` (-33%, too small), `.mf` (0%), `.jpg`/`.png` (4-6%, binary-like)
