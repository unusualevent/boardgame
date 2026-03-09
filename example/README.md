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
.backend              1          229          167      27.1%        2.5ms
.bat                  1         4176         2321      44.4%       91.6ms
.bin                  2          198          260     -31.3%         58us
.cfg                  3         1134          853      24.3%        2.0ms
.code-workspace       1          926          331      64.3%        1.5ms
.conf                 5         1149          809      27.0%        623us
.css                 13        54931        24272      47.6%      104.6ms
.d                   18        22634         4446      74.1%       10.1ms
.dockerignore         1           71           57      19.7%        159us
.editorconfig         1          216          169      21.8%        591us
.example              3          830          634      19.2%        1.0ms
.expr                 1          216          134      38.0%        255us
.gitattributes        1           19           18       5.3%         13us
.gitignore           22         4021         2988      21.7%        467us
.go                 364      1798835       868110      48.4%      133.9ms
.hcl                  4         5222         3440      23.4%       15.2ms
.html                55        76823        43520      36.5%       17.4ms
.j2                   5         2088         1265      40.2%        1.9ms
.java                 3        50732        12535      75.0%      418.1ms
.jpg                  4           61           58       4.5%          8us
.js                  46        97108        44521      45.0%       32.4ms
.json                98        41511        26222      30.8%        2.4ms
.kt                  58       207636        80771      56.3%       42.0ms
.kts                  8         6543         3282      40.3%        5.9ms
.list                 4          248          217      11.6%         34us
.lock                 1         3917         1519      61.2%       28.0ms
.md                 158       529443       325811      33.3%       60.5ms
.mf                   1           25           25       0.0%         12us
.mjs                  4        11434         5703      46.3%       63.0ms
.mod                 33        20923        10071      30.0%        3.7ms
.png                 24          386          362       6.2%          5us
.pro                  2          769          363      32.3%        3.7ms
.properties           6          939          629      26.0%        294us
.pub                  1          146          127      13.0%         74us
.rb                   1         5766         2671      53.7%       99.0ms
.rego                 1         1362          571      58.1%        7.3ms
.rs                   4        10224         3037      51.3%       25.5ms
.service              4         1336         1026      21.8%        838us
.sh                  13        24921        17812      33.4%       32.4ms
.sum                 21        88913        53914      36.3%       68.1ms
.svg                  7         7256         4716      33.8%        8.8ms
.tab                  2            6            8     -33.3%          3us
.tag                  2          354          284      19.8%         97us
.tf                   8         4884         2683      38.9%        3.4ms
.tfstate              1          182          135      25.8%        167us
.tfvars               1           25           23       8.0%         17us
.timestamp           28         1344         1204      10.4%         17us
.toml                 4         2240         1013      33.2%        3.4ms
.ts                  30        33137        17133      37.6%       22.4ms
.txt                  5        19535        14071      26.7%       48.3ms
.vue                 96       524359       286458      41.7%      151.9ms
.xml                 28        26291        11253      43.5%        7.4ms
.yaml                44        40856        16700      57.1%        5.0ms
.yml                 73        49656        24233      35.7%        5.0ms
------------------------------------------------------------------------
TOTAL              1325      3788186      1924955      39.8%       64.4ms
```

### Avg Compression Time vs Avg File Size (sorted by size)

```
.tab                 3B | ###                                      3us
.jpg                15B | ######                                   8us
.png                16B | ####                                     5us
.gitattributes      19B | #######                                  13us
.mf                 25B | #######                                  12us
.tfvars             25B | ########                                 17us
.timestamp          48B | ########                                 17us
.list               62B | ##########                               34us
.dockerignore       71B | ###############                          159us
.bin                99B | ############                             58us
.pub               146B | #############                            74us
.properties        156B | #################                        294us
.tag               177B | ##############                           97us
.tfstate           182B | ###############                          167us
.gitignore         182B | ##################                       467us
.editorconfig      216B | ###################                      591us
.expr              216B | #################                        255us
.backend           229B | ########################                 2.5ms
.conf              229B | ###################                      623us
.example           276B | #####################                    1.0ms
.service           334B | ####################                     838us
.cfg               378B | #######################                  2.0ms
.pro               384B | #########################                3.7ms
.j2                417B | #######################                  1.9ms
.json              423B | ########################                 2.4ms
.toml              560B | #########################                3.4ms
.tf                610B | #########################                3.4ms
.mod               634B | #########################                3.7ms
.yml               680B | ##########################               5.0ms
.kts               817B | ##########################               5.9ms
.code-workspace     926B | ######################                   1.5ms
.yaml              928B | ##########################               5.0ms
.xml               938B | ###########################              7.4ms
.svg              1.0KB | ############################             8.8ms
.ts               1.1KB | ##############################           22.4ms
.d                1.2KB | ############################             10.1ms
.hcl              1.3KB | #############################            15.2ms
.rego             1.3KB | ###########################              7.3ms
.html             1.4KB | ##############################           17.4ms
.sh               1.9KB | ################################         32.4ms
.js               2.1KB | ################################         32.4ms
.rs               2.5KB | ###############################          25.5ms
.mjs              2.8KB | ##################################       63.0ms
.md               3.3KB | ##################################       60.5ms
.kt               3.5KB | ################################         42.0ms
.txt              3.8KB | #################################        48.3ms
.lock             3.8KB | ###############################          28.0ms
.bat              4.1KB | ###################################      91.6ms
.css              4.1KB | ###################################      104.6ms
.sum              4.1KB | ##################################       68.1ms
.go               4.8KB | ####################################     133.9ms
.vue              5.3KB | ####################################     151.9ms
.rb               5.6KB | ###################################      99.0ms
.java            16.5KB | ######################################## 418.1ms
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
.service           334B | ########                                 21.8%
.cfg               378B | #########                                24.3%
.pro               384B | ############                             32.3%
.j2                417B | ################                         40.2%
.json              423B | ############                             30.8%
.toml              560B | #############                            33.2%
.tf                610B | ###############                          38.9%
.mod               634B | ############                             30.0%
.yml               680B | ##############                           35.7%
.kts               817B | ################                         40.3%
.code-workspace     926B | #########################                64.3%
.yaml              928B | ######################                   57.1%
.xml               938B | #################                        43.5%
.svg              1.0KB | #############                            33.8%
.ts               1.1KB | ###############                          37.6%
.d                1.2KB | #############################            74.1%
.hcl              1.3KB | #########                                23.4%
.rego             1.3KB | #######################                  58.1%
.html             1.4KB | ##############                           36.5%
.sh               1.9KB | #############                            33.4%
.js               2.1KB | #################                        45.0%
.rs               2.5KB | ####################                     51.3%
.mjs              2.8KB | ##################                       46.3%
.md               3.3KB | #############                            33.3%
.kt               3.5KB | ######################                   56.3%
.txt              3.8KB | ##########                               26.7%
.lock             3.8KB | ########################                 61.2%
.bat              4.1KB | #################                        44.4%
.css              4.1KB | ###################                      47.6%
.sum              4.1KB | ##############                           36.3%
.go               4.8KB | ###################                      48.4%
.vue              5.3KB | ################                         41.7%
.rb               5.6KB | #####################                    53.7%
.java            16.5KB | ##############################           75.0%
```

### Observations

- **Overall**: 39.8% average compression across 1325 files (3.8MB -> 1.9MB)
- **UTF-8 text detection**: 192 more files now processed after switching from ASCII-only to null-byte binary detection (matching Git's heuristic)
- **Notable gains**: `.go` 295 -> 364 files, `.md` 84 -> 158 files, `.html` 23 -> 55 files, `.vue` 80 -> 96 files
- **Time scales superlinearly**: a 5x file size increase costs ~30x more compression time
- **Ratio improves with size**: files under ~200B barely compress; larger files with more repeated patterns reach 50-75%
- **Best compressors**: `.java` (75%), `.d` (74%), `.code-workspace` (64%), `.yaml` (57%), `.kt` (56%)
- **Worst compressors**: `.tab` (-33%, too small), `.bin` (-31%), `.mf` (0%), `.jpg`/`.png` (4-6%)
