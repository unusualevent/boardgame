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

Ran against a mixed-language monorepo (~1335 text files, node_modules excluded,
max 20KB per file):

```
Extension         Files     Original   Compressed  Avg Ratio     Avg Time
------------------------------------------------------------------------
.backend              1          229          167      27.1%        232us
.bat                  1         4176         2289      45.2%      138.5ms
.bin                  2          198          260     -31.3%         20us
.cfg                  3         1134          857      23.8%        599us
.code-workspace       1          926          331      64.3%        929us
.conf                 5         1149          803      27.4%        189us
.css                 13        54931        24051      47.8%       62.0ms
.d                   18        22634         4446      74.1%        1.6ms
.editorconfig         1          216          169      21.8%        184us
.example              3          830          631      19.7%        831us
.expr                 1          216          129      40.3%        100us
.gitattributes        1           19           18       5.3%          7us
.gitignore           22         4021         2987      21.7%        837us
.go                 368      1814428       860655      49.2%       97.5ms
.hcl                  4         5222         3385      23.8%       26.9ms
.html                55        76823        43373      36.0%       11.7ms
.j2                   5         2088         1261      40.3%        7.4ms
.java                 3        50732        13733      72.6%      240.2ms
.jpg                  4           61           58       4.5%          7us
.js                  46        97108        44638      44.9%       23.8ms
.json                98        41682        27027      29.6%        2.2ms
.kt                  58       207636        80781      56.8%       30.9ms
.kts                  8         6543         3324      40.3%        2.7ms
.list                 4          248          217      11.6%         20us
.lock                 1         3917         1505      61.6%        8.8ms
.md                 162       559553       337073      34.7%       55.6ms
.mf                   1           25           25       0.0%         17us
.mjs                  4        11434         5758      46.0%       54.6ms
.mod                 33        21156        10184      30.7%        4.0ms
.png                 24          386          362       6.2%          4us
.pro                  2          769          363      32.3%        2.4ms
.properties           6          939          629      26.0%        938us
.pub                  1          146          127      13.0%       12.4ms
.rb                   1         5766         2865      50.3%       47.6ms
.rego                 1         1362          571      58.1%        4.0ms
.rs                   5        10277         3976      41.5%       14.4ms
.service              4         1336         1033      21.4%        787us
.sh                  13        24921        17783      33.4%       27.9ms
.sum                 22        89607        54226      36.4%       50.6ms
.svg                  7         7256         4688      34.0%       14.4ms
.tab                  2            6            8     -33.3%         20us
.tag                  2          354          284      19.8%         79us
.tf                   8         4884         2677      39.1%        4.5ms
.tfstate              1          182          135      25.8%         80us
.tfvars               1           25           23       8.0%          8us
.timestamp           28         1344         1204      10.4%        332us
.toml                 4         2240          995      33.9%        6.8ms
.ts                  30        33137        17056      37.8%       19.1ms
.txt                  5        24567        18301      26.7%       88.9ms
.vue                 96       524359       278369      43.0%      123.7ms
.xml                 28        26291        11236      43.9%        4.6ms
.yaml                44        41082        17399      56.0%        3.2ms
.yml                 73        49656        25142      34.4%        4.9ms
------------------------------------------------------------------------
TOTAL              1335      3840298      1929644      40.1%       49.6ms
```

### Avg Compression Time vs Avg File Size (sorted by size)

```
.tab                 3B | #########                                20us
.jpg                15B | ######                                   7us
.png                16B | ####                                     4us
.gitattributes      19B | ######                                   7us
.mf                 25B | #########                                17us
.tfvars             25B | ######                                   8us
.timestamp          48B | ##################                       332us
.list               62B | #########                                20us
.dockerignore       71B | ############                             43us
.bin                99B | #########                                20us
.pub               146B | ##############################           12.4ms
.properties        156B | ######################                   938us
.tag               177B | ##############                           79us
.tfstate           182B | ##############                           80us
.gitignore         182B | #####################                    837us
.editorconfig      216B | ################                         184us
.expr              216B | ##############                           100us
.backend           229B | #################                        232us
.conf              229B | ################                         189us
.example           276B | #####################                    831us
.service           334B | #####################                    787us
.cfg               378B | ####################                     599us
.pro               384B | #########################                2.4ms
.j2                417B | ############################             7.4ms
.json              425B | ########################                 2.2ms
.toml              560B | ############################             6.8ms
.tf                610B | ###########################              4.5ms
.mod               641B | ##########################               4.0ms
.yml               680B | ###########################              4.9ms
.kts               817B | #########################                2.7ms
.code-workspace     926B | ######################                   929us
.yaml              933B | ##########################               3.2ms
.xml               938B | ###########################              4.6ms
.svg              1.0KB | ##############################           14.4ms
.ts               1.1KB | ###############################          19.1ms
.d                1.2KB | #######################                  1.6ms
.hcl              1.3KB | ################################         26.9ms
.rego             1.3KB | ##########################               4.0ms
.html             1.4KB | ##############################           11.7ms
.sh               1.9KB | #################################        27.9ms
.rs               2.0KB | ##############################           14.4ms
.js               2.1KB | ################################         23.8ms
.mjs              2.8KB | ###################################      54.6ms
.md               3.4KB | ###################################      55.6ms
.kt               3.5KB | #################################        30.9ms
.lock             3.8KB | #############################            8.8ms
.sum              4.0KB | ##################################       50.6ms
.bat              4.1KB | ######################################   138.5ms
.css              4.1KB | ###################################      62.0ms
.txt              4.8KB | ####################################     88.9ms
.go               4.8KB | #####################################    97.5ms
.vue              5.3KB | #####################################    123.7ms
.rb               5.6KB | ##################################       47.6ms
.java            16.5KB | ######################################## 240.2ms
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
.bin                99B |                                          -31.3%
.pub               146B | #####                                    13.0%
.properties        156B | ##########                               26.0%
.tag               177B | #######                                  19.8%
.tfstate           182B | ##########                               25.8%
.gitignore         182B | ########                                 21.7%
.editorconfig      216B | ########                                 21.8%
.expr              216B | ################                         40.3%
.backend           229B | ##########                               27.1%
.conf              229B | ##########                               27.4%
.example           276B | #######                                  19.7%
.service           334B | ########                                 21.4%
.cfg               378B | #########                                23.8%
.pro               384B | ############                             32.3%
.j2                417B | ################                         40.3%
.json              425B | ###########                              29.6%
.toml              560B | #############                            33.9%
.tf                610B | ###############                          39.1%
.mod               641B | ############                             30.7%
.yml               680B | #############                            34.4%
.kts               817B | ################                         40.3%
.code-workspace     926B | #########################                64.3%
.yaml              933B | ######################                   56.0%
.xml               938B | #################                        43.9%
.svg              1.0KB | #############                            34.0%
.ts               1.1KB | ###############                          37.8%
.d                1.2KB | #############################            74.1%
.hcl              1.3KB | #########                                23.8%
.rego             1.3KB | #######################                  58.1%
.html             1.4KB | ##############                           36.0%
.sh               1.9KB | #############                            33.4%
.rs               2.0KB | ################                         41.5%
.js               2.1KB | #################                        44.9%
.mjs              2.8KB | ##################                       46.0%
.md               3.4KB | #############                            34.7%
.kt               3.5KB | ######################                   56.8%
.lock             3.8KB | ########################                 61.6%
.sum              4.0KB | ##############                           36.4%
.bat              4.1KB | ##################                       45.2%
.css              4.1KB | ###################                      47.8%
.txt              4.8KB | ##########                               26.7%
.go               4.8KB | ###################                      49.2%
.vue              5.3KB | #################                        43.0%
.rb               5.6KB | ####################                     50.3%
.java            16.5KB | #############################            72.6%
```

### Observations

- **Overall**: 40.1% average compression across 1335 files (3.8MB -> 1.9MB)
- **Avg encode time**: 49.6ms (suffix-array candidate search is CPU-bound)
- **Time scales superlinearly**: a 5x file size increase costs ~15x more compression time
- **Ratio improves with size**: files under ~200B barely compress; larger files with more repeated patterns reach 50-75%
- **Best compressors**: `.d` (74%), `.java` (73%), `.code-workspace` (64%), `.lock` (62%), `.yaml` (56%), `.kt` (57%)
- **Worst compressors**: `.tab` (-33%, too small), `.bin` (-31%), `.mf` (0%), `.jpg`/`.png` (4-6%)
