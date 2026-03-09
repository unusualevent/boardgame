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
.backend              1          229          167      27.1%        1.4ms
.cfg                  3         1134          853      24.3%        1.3ms
.code-workspace       1          926          331      64.3%        1.8ms
.conf                 4          853          622      24.6%        279us
.css                 13        54931        24272      47.6%       99.4ms
.d                   18        22634         4446      74.1%        8.4ms
.dockerignore         1           71           57      19.7%         49us
.editorconfig         1          216          169      21.8%        380us
.example              3          830          634      19.2%        1.7ms
.expr                 1          216          134      38.0%        197us
.gitignore           22         4021         2988      21.7%        475us
.go                 295      1237541       586503      48.5%       91.0ms
.hcl                  4         5222         3440      23.4%       12.5ms
.html                22        15028         9478      27.8%        6.3ms
.j2                   5         2088         1265      40.2%        1.6ms
.java                 3        50732        12535      75.0%      343.6ms
.js                  41        72288        32242      44.3%       20.0ms
.json                97        38713        24749      30.6%        1.7ms
.kt                  57       205671        79861      56.3%       38.8ms
.kts                  8         6543         3282      40.3%        5.1ms
.list                 4          248          217      11.6%         22us
.lock                 1         3917         1519      61.2%       23.4ms
.md                  83       248334       147054      34.3%       53.6ms
.mjs                  4        11434         5703      46.3%       52.8ms
.mod                 33        20923        10071      30.0%        3.6ms
.pro                  2          769          363      32.3%        1.7ms
.properties           6          939          629      26.0%        525us
.pub                  1          146          127      13.0%         62us
.rego                 1         1362          571      58.1%        8.5ms
.rs                   4        10224         3037      51.3%       19.9ms
.service              4         1336         1026      21.8%        953us
.sh                  10         7639         4600      35.9%        4.6ms
.sum                 21        88913        53914      36.3%       62.3ms
.svg                  7         7256         4716      33.8%        8.8ms
.tf                   8         4884         2683      38.9%        2.4ms
.toml                 4         2240         1013      33.2%        3.2ms
.ts                  29        32053        16548      37.3%       18.6ms
.txt                  5        24566        18363      26.7%       56.6ms
.vue                 78       366845       202563      40.7%      117.3ms
.xml                 27        24800        10619      43.0%        6.2ms
.yaml                44        40856        16700      57.1%        5.1ms
.yml                 73        49630        24223      35.7%        4.5ms
------------------------------------------------------------------------
TOTAL              1112      2671578      1316379      39.8%       44.8ms
```

### Avg Compression Time vs Avg File Size (sorted by size)

```
.tab                 3B | ###                                      3us
.timestamp          48B | #############                            86us
.list               62B | #########                                22us
.dockerignore       71B | ############                             49us
.pub               146B | ############                             62us
.properties        156B | ###################                      525us
.tag               177B | ##############                           88us
.gitignore         182B | ###################                      475us
.conf              213B | #################                        279us
.expr              216B | ################                         197us
.editorconfig      216B | ##################                       380us
.backend           229B | ######################                   1.4ms
.example           276B | #######################                  1.7ms
.service           334B | #####################                    953us
.cfg               378B | ######################                   1.3ms
.pro               384B | #######################                  1.7ms
.json              399B | #######################                  1.7ms
.j2                417B | #######################                  1.6ms
.toml              560B | #########################                3.2ms
.tf                610B | ########################                 2.4ms
.mod               634B | #########################                3.6ms
.yml               679B | ##########################               4.5ms
.html              683B | ###########################              6.3ms
.sh                763B | ##########################               4.6ms
.kts               817B | ##########################               5.1ms
.xml               918B | ###########################              6.2ms
.code-workspace     926B | #######################                  1.8ms
.yaml              928B | ##########################               5.1ms
.svg              1.0KB | ############################             8.8ms
.ts               1.1KB | ##############################           18.6ms
.d                1.2KB | ############################             8.4ms
.hcl              1.3KB | #############################            12.5ms
.rego             1.3KB | ############################             8.5ms
.js               1.7KB | ###############################          20.0ms
.rs               2.5KB | ###############################          19.9ms
.mjs              2.8KB | ##################################       52.8ms
.md               2.9KB | ##################################       53.6ms
.kt               3.5KB | #################################        38.8ms
.lock             3.8KB | ###############################          23.4ms
.go               4.1KB | ###################################      91.0ms
.css              4.1KB | ####################################     99.4ms
.sum              4.1KB | ##################################       62.3ms
.vue              4.6KB | ####################################     117.3ms
.txt              4.8KB | ##################################       56.6ms
.java            16.5KB | ######################################## 343.6ms
```

### Avg Compression Ratio vs Avg File Size (sorted by size)

```
.tab                 3B |                                          -33.3%
.timestamp          48B | ####                                     10.4%
.list               62B | ####                                     11.6%
.dockerignore       71B | #######                                  19.7%
.pub               146B | #####                                    13.0%
.properties        156B | ##########                               26.0%
.tag               177B | #######                                  19.8%
.gitignore         182B | ########                                 21.7%
.conf              213B | #########                                24.6%
.expr              216B | ###############                          38.0%
.editorconfig      216B | ########                                 21.8%
.backend           229B | ##########                               27.1%
.example           276B | #######                                  19.2%
.service           334B | ########                                 21.8%
.cfg               378B | #########                                24.3%
.pro               384B | ############                             32.3%
.json              399B | ############                             30.6%
.j2                417B | ################                         40.2%
.toml              560B | #############                            33.2%
.tf                610B | ###############                          38.9%
.mod               634B | ############                             30.0%
.yml               679B | ##############                           35.7%
.html              683B | ###########                              27.8%
.sh                763B | ##############                           35.9%
.kts               817B | ################                         40.3%
.xml               918B | #################                        43.0%
.code-workspace     926B | #########################                64.3%
.yaml              928B | ######################                   57.1%
.svg              1.0KB | #############                            33.8%
.ts               1.1KB | ##############                           37.3%
.d                1.2KB | #############################            74.1%
.hcl              1.3KB | #########                                23.4%
.rego             1.3KB | #######################                  58.1%
.js               1.7KB | #################                        44.3%
.rs               2.5KB | ####################                     51.3%
.mjs              2.8KB | ##################                       46.3%
.md               2.9KB | #############                            34.3%
.kt               3.5KB | ######################                   56.3%
.lock             3.8KB | ########################                 61.2%
.go               4.1KB | ###################                      48.5%
.css              4.1KB | ###################                      47.6%
.sum              4.1KB | ##############                           36.3%
.vue              4.6KB | ################                         40.7%
.txt              4.8KB | ##########                               26.7%
.java            16.5KB | ##############################           75.0%
```

### Observations

- **Overall**: 39.8% average compression across 1112 files (2.6MB -> 1.3MB)
- **Time scales superlinearly**: a 5x file size increase costs ~30x more compression time
- **Ratio improves with size**: files under ~200B barely compress; larger files with more repeated patterns reach 50-75%
- **Best compressors**: `.java` (75%), `.d` (74%), `.code-workspace` (64%), `.yaml` (57%), `.kt` (56%)
- **Worst compressors**: `.tab` (-33%, too small), `.timestamp` (10%), `.list` (12%)
