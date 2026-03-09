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

Ran against a mixed-language monorepo (~1300 text files, node_modules excluded,
max 20KB per file):

```
Extension         Files     Original   Compressed  Avg Ratio     Avg Time
------------------------------------------------------------------------
.backend              1          229          167      27.1%        288us
.bat                  1         4176         2320      44.4%       38.4ms
.bin                  2          198          260     -31.3%        224us
.cfg                  3         1134          853      24.3%        742us
.code-workspace       1          926          331      64.3%        669us
.conf                 5         1149          810      27.0%        405us
.css                 13        54931        24291      47.6%       48.2ms
.d                   18        22634         4446      74.1%        1.8ms
.dockerignore         1           71           57      19.7%         51us
.editorconfig         1          216          169      21.8%        1.6ms
.example              3          830          633      19.2%        482us
.expr                 1          216          134      38.0%        146us
.gitattributes        1           19           18       5.3%         11us
.gitignore           22         4021         2987      21.7%        419us
.go                 368      1809638       872409      48.3%       58.3ms
.hcl                  4         5222         3437      23.4%        8.3ms
.html                55        76823        43440      36.6%        9.3ms
.j2                   5         2088         1266      40.2%        1.0ms
.java                 3        50732        12521      75.1%      150.2ms
.jpg                  4           61           58       4.5%          5us
.js                  46        97108        44590      44.9%       15.7ms
.json                98        41639        26263      30.9%        1.3ms
.kt                  58       207636        80738      56.3%       18.8ms
.kts                  8         6543         3287      40.2%        3.5ms
.list                 4          248          217      11.6%         28us
.lock                 1         3917         1519      61.2%       13.1ms
.md                 160       538321       330877      33.4%       32.5ms
.mf                   1           25           25       0.0%          7us
.mjs                  4        11434         5702      46.3%       26.2ms
.mod                 33        20950        10096      30.1%        1.8ms
.png                 24          386          362       6.2%          6us
.pro                  2          769          363      32.3%        1.7ms
.properties           6          939          629      26.0%         89us
.pub                  1          146          127      13.0%         60us
.rb                   1         5766         2666      53.8%       53.9ms
.rego                 1         1362          571      58.1%        3.5ms
.rs                   5        10277         3091      42.8%        6.4ms
.service              4         1336         1028      21.7%        372us
.sh                  13        24921        17813      33.4%       18.0ms
.sum                 21        88913        53924      36.3%       32.4ms
.svg                  7         7256         4712      33.8%        4.1ms
.tab                  2            6            8     -33.3%          6us
.tag                  2          354          284      19.8%         62us
.tf                   8         4884         2694      38.7%        1.2ms
.tfstate              1          182          135      25.8%        142us
.tfvars               1           25           23       8.0%         15us
.timestamp           28         1344         1204      10.4%         11us
.toml                 4         2240         1005      33.7%        1.9ms
.ts                  30        33137        17101      37.6%       10.6ms
.txt                  5        24567        18371      26.7%       39.3ms
.vue                 96       524359       286116      41.8%       73.4ms
.xml                 28        26291        11263      43.5%        2.9ms
.yaml                44        41082        16847      57.1%        2.1ms
.yml                 73        49656        24233      35.7%        2.1ms
------------------------------------------------------------------------
TOTAL              1332      3813333      1938491      39.8%       29.6ms
```

### Avg Compression Time vs Avg File Size (sorted by size)

```
.tab                 3B | ######                                   6us
.jpg                15B | #####                                    5us
.png                16B | ######                                   6us
.gitattributes      19B | ########                                 11us
.mf                 25B | ######                                   7us
.tfvars             25B | #########                                15us
.timestamp          48B | ########                                 11us
.list               62B | ###########                              28us
.dockerignore       71B | #############                            51us
.bin                99B | ##################                       224us
.pub               146B | #############                            60us
.properties        156B | ###############                          89us
.tag               177B | #############                            62us
.tfstate           182B | ################                         142us
.gitignore         182B | ####################                     419us
.editorconfig      216B | ########################                 1.6ms
.expr              216B | ################                         146us
.backend           229B | ###################                      288us
.conf              229B | ####################                     405us
.example           276B | ####################                     482us
.service           334B | ###################                      372us
.cfg               378B | ######################                   742us
.pro               384B | ########################                 1.7ms
.j2                417B | #######################                  1.0ms
.json              424B | #######################                  1.3ms
.toml              560B | #########################                1.9ms
.tf                610B | #######################                  1.2ms
.mod               634B | #########################                1.8ms
.yml               680B | #########################                2.1ms
.kts               817B | ###########################              3.5ms
.code-workspace     926B | #####################                    669us
.yaml              933B | #########################                2.1ms
.xml               938B | ##########################               2.9ms
.svg              1.0KB | ###########################              4.1ms
.ts               1.1KB | ###############################          10.6ms
.d                1.2KB | #########################                1.8ms
.hcl              1.3KB | ##############################           8.3ms
.rego             1.3KB | ###########################              3.5ms
.html             1.4KB | ##############################           9.3ms
.sh               1.9KB | ################################         18.0ms
.rs               2.0KB | #############################            6.4ms
.js               2.1KB | ################################         15.7ms
.mjs              2.8KB | ##################################       26.2ms
.md               3.3KB | ##################################       32.5ms
.kt               3.5KB | #################################        18.8ms
.lock             3.8KB | ###############################          13.1ms
.bat              4.1KB | ###################################      38.4ms
.css              4.1KB | ####################################     48.2ms
.sum              4.1KB | ##################################       32.4ms
.txt              4.8KB | ###################################      39.3ms
.go               4.8KB | ####################################     58.3ms
.vue              5.3KB | #####################################    73.4ms
.rb               5.6KB | ####################################     53.9ms
.java            16.5KB | ######################################## 150.2ms
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
.expr              216B | ###############                          38.0%
.backend           229B | ##########                               27.1%
.conf              229B | ##########                               27.0%
.example           276B | #######                                  19.2%
.service           334B | ########                                 21.7%
.cfg               378B | #########                                24.3%
.pro               384B | ############                             32.3%
.j2                417B | ################                         40.2%
.json              424B | ############                             30.9%
.toml              560B | #############                            33.7%
.tf                610B | ###############                          38.7%
.mod               634B | ############                             30.1%
.yml               680B | ##############                           35.7%
.kts               817B | ################                         40.2%
.code-workspace     926B | #########################                64.3%
.yaml              933B | ######################                   57.1%
.xml               938B | #################                        43.5%
.svg              1.0KB | #############                            33.8%
.ts               1.1KB | ###############                          37.6%
.d                1.2KB | #############################            74.1%
.hcl              1.3KB | #########                                23.4%
.rego             1.3KB | #######################                  58.1%
.html             1.4KB | ##############                           36.6%
.sh               1.9KB | #############                            33.4%
.rs               2.0KB | #################                        42.8%
.js               2.1KB | #################                        44.9%
.mjs              2.8KB | ##################                       46.3%
.md               3.3KB | #############                            33.4%
.kt               3.5KB | ######################                   56.3%
.lock             3.8KB | ########################                 61.2%
.bat              4.1KB | #################                        44.4%
.css              4.1KB | ###################                      47.6%
.sum              4.1KB | ##############                           36.3%
.txt              4.8KB | ##########                               26.7%
.go               4.8KB | ###################                      48.3%
.vue              5.3KB | ################                         41.8%
.rb               5.6KB | #####################                    53.8%
.java            16.5KB | ##############################           75.1%
```

### Observations

- **Overall**: 39.8% average compression across 1332 files (3.8MB -> 1.9MB)
- **Avg encode time**: 29.6ms (down from 64.4ms after encoder optimizations)
- **Time scales superlinearly**: a 5x file size increase costs ~15x more compression time
- **Ratio improves with size**: files under ~200B barely compress; larger files with more repeated patterns reach 50-75%
- **Best compressors**: `.java` (75%), `.d` (74%), `.code-workspace` (64%), `.yaml` (57%), `.kt` (56%)
- **Worst compressors**: `.tab` (-33%, too small), `.bin` (-31%), `.mf` (0%), `.jpg`/`.png` (4-6%)
