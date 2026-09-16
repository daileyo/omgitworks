# Changelog

## [3.1.0](https://github.com/daileyo/omgitworks/compare/v3.0.4...v3.1.0) (2026-09-16)


### Features

* **worktree:** accept . as the current repo for list and align ([3311047](https://github.com/daileyo/omgitworks/commit/33110475acff764ea6917c65f0b0376f22074d91))
* **worktree:** add current-repository resolver ([9187872](https://github.com/daileyo/omgitworks/commit/91878722ae00ad6f3e41506cbcfe42005b41f668))
* **worktree:** complete arguments for add, list, and align ([df60992](https://github.com/daileyo/omgitworks/commit/df60992289391beb48c85707b2b61274116bfdc5))
* **worktree:** resolve the repo for worktree add ([79b3d95](https://github.com/daileyo/omgitworks/commit/79b3d95ce716322a6dbc171c1641879d9cc9ae45))

## [3.0.4](https://github.com/daileyo/omgitworks/compare/v3.0.3...v3.0.4) (2026-09-15)


### Bug Fixes

* **user:** read git identity through git config ([d9777a7](https://github.com/daileyo/omgitworks/commit/d9777a742793f4b9533177d0be68502d97d538df))
* **user:** run git config with a context ([4503799](https://github.com/daileyo/omgitworks/commit/4503799b5e129d257ba1a80f4bf78b79b9268386))

## [3.0.3](https://github.com/daileyo/omgitworks/compare/v3.0.2...v3.0.3) (2026-09-15)


### Bug Fixes

* **cli:** render each command's real flags in help ([d2ac0a3](https://github.com/daileyo/omgitworks/commit/d2ac0a3aff99f167cead820263f4dbe749a85a48))

## [3.0.2](https://github.com/daileyo/omgitworks/compare/v3.0.1...v3.0.2) (2026-09-15)


### Bug Fixes

* **shell:** route worktree help flags to the binary ([03203eb](https://github.com/daileyo/omgitworks/commit/03203eb545e190a825108045278d18c5d6b0ee0e))

## [3.0.1](https://github.com/daileyo/omgitworks/compare/v3.0.0...v3.0.1) (2026-09-15)


### Bug Fixes

* **release:** correct the repo name and add a retry path ([0e10e2c](https://github.com/daileyo/omgitworks/commit/0e10e2cfa6cb39fd5305a782a6ff4eebc9abee75))

## [3.0.0](https://github.com/daileyo/omgitworks/compare/v2.22.0...v3.0.0) (2026-09-15)


### ⚠ BREAKING CHANGES

* the binary is now omgitworks and the command is omgw. The gws function is still defined by shell-init, and a git-workspace symlink is documented, so existing shell configs keep working.

### Code Refactoring

* rename the binary and command to omgitworks and omgw ([8b79dd4](https://github.com/daileyo/omgitworks/commit/8b79dd414e1b19be5fd0da91a6518a4761b26eb0))

## [2.22.0](https://github.com/daileyo/gws/compare/v2.21.0...v2.22.0) (2026-09-14)


### Features

* **config:** migrate config to the XDG location ([b74a68b](https://github.com/daileyo/gws/commit/b74a68b618fb6738c0eb8f48385a62808097f301))
* **config:** resolve paths via XDG base directories ([f632a13](https://github.com/daileyo/gws/commit/f632a13b386f6a76922082e3c42acfd2c6ae6785))
* **worktree:** migrate legacy .wt layouts via align ([8b0a443](https://github.com/daileyo/gws/commit/8b0a443ca568a0a0ab0da1e7190c532dd8c8a923))
* **worktree:** relocate worktrees to the projects root ([428212d](https://github.com/daileyo/gws/commit/428212de70d7703c129ae2bc521a86889f214601))


### Bug Fixes

* **config:** drop dead branch in migration guard ([860c4d8](https://github.com/daileyo/gws/commit/860c4d88c167a57fe5d330497d84ae259585be41))
* **test:** clear XDG vars when isolating tests ([d832fec](https://github.com/daileyo/gws/commit/d832fec841f3820ab3d091129f9b1c8bd061fb5e))

## [2.21.0](https://github.com/daileyo/gws/compare/v2.20.0...v2.21.0) (2026-09-13)


### Features

* **cmd:** add cd command for workspace root ([480c049](https://github.com/daileyo/gws/commit/480c049ca3c8c72d04ac70b19f0ce4bad5423426))
* **shell:** route cd through the gws function ([309719e](https://github.com/daileyo/gws/commit/309719e5829664354ffe2b9521ebe56f28e0ae1e))

## [2.20.0](https://github.com/daileyo/gws/compare/v2.19.2...v2.20.0) (2026-08-16)


### Features

* **shell:** add PowerShell shell-init template with full navigation support ([0794e63](https://github.com/daileyo/gws/commit/0794e6349e72f8ac3e28de8d92101bcb74a5ec26))

## [2.19.2](https://github.com/daileyo/gws/compare/v2.19.1...v2.19.2) (2026-04-23)


### Bug Fixes

* use visible width for multi-column alignment with ANSI codes ([df38172](https://github.com/daileyo/gws/commit/df3817254798f6fc91833d71751c2efde7a8d391))

## [2.19.1](https://github.com/daileyo/gws/compare/v2.19.0...v2.19.1) (2026-04-20)


### Bug Fixes

* ensure bash-completion is sourced before cobra completions ([c300596](https://github.com/daileyo/gws/commit/c3005961cb53577fdbbc5009a3cfbf2b85a073d4))
* use correct cobra function name for bash completion ([a4a0ed2](https://github.com/daileyo/gws/commit/a4a0ed2016534490c86f2f4c0c159685011c427a))

## [2.19.0](https://github.com/daileyo/gws/compare/v2.18.0...v2.19.0) (2026-04-20)


### Features

* add --worktree flag for navigating to repo worktrees ([a3d1552](https://github.com/daileyo/gws/commit/a3d1552c3ad97f2e3c76c1af5e9a85c64aa7f1f9))
* add (wt) indicator to list output for repos with worktrees ([3b3c419](https://github.com/daileyo/gws/commit/3b3c41996b8b72d4a2bded444219999541f257a5))
* add global worktree navigation via gws worktree &lt;branch&gt; ([d5b7f9c](https://github.com/daileyo/gws/commit/d5b7f9c096a9c99a0bed9a9c6bd26f53ab47b372))
* add make use-dev and use-release targets for switching builds ([998b79a](https://github.com/daileyo/gws/commit/998b79af9c3fea663e0dd52d1ec9e4424ce41d19))
* add shell integration for worktree navigation (-wt flag) ([a4af20c](https://github.com/daileyo/gws/commit/a4af20ccbff935978a80d7d0bac32a94e322e8ee))
* add worktree data model, discovery, and refresh integration ([219a078](https://github.com/daileyo/gws/commit/219a0782995b3f3261d63f9554d0cf54db040d6f))
* add worktree subcommands (list, add, align) ([e81ddaa](https://github.com/daileyo/gws/commit/e81ddaa51366682520bf8bd8ef28b327a9520ff6))
* color (wt) indicator orange across all gws output ([10e8504](https://github.com/daileyo/gws/commit/10e8504fc97850e4476bd65bcc88ece28b67789d))


### Bug Fixes

* add rollback and retry logic to MoveWorktree for partial failures ([ca048d9](https://github.com/daileyo/gws/commit/ca048d9a6140b3bb0122280afe97c2e1c4d9f928))
* format worktree_test.go with gofmt ([04af01f](https://github.com/daileyo/gws/commit/04af01fda8fe5a8e31321ffbedf0b40c40290971))
* include git stderr in error messages for better diagnostics ([d8a3041](https://github.com/daileyo/gws/commit/d8a30416c441ec8956faa8512adfe259191a5738))
* proactively prune stale worktrees and handle detached HEAD display ([0a1452e](https://github.com/daileyo/gws/commit/0a1452e27ded5acdae98db4f812d5f56772accd1))
* remove deprecated -g flag from shell templates, add worktree shortcut ([9d20423](https://github.com/daileyo/gws/commit/9d20423a6e441144ece314d5d44b1ee192008d69))
* resolve ineffassign lint error in worktree align test ([79c84a1](https://github.com/daileyo/gws/commit/79c84a1d6c3ca836d3d9462c3c41854aba579a1a))
* restore missing 'n' in release-please.yml name field ([5b744eb](https://github.com/daileyo/gws/commit/5b744ebf9d4c106a0b99f99aa3222a025ddfd073))
* run git worktree repair/prune during move recovery ([ddfda92](https://github.com/daileyo/gws/commit/ddfda921b2f240b2156a556a0b4a40f81884758e))
* use repair-before-prune strategy and skip locked worktrees ([848e7b2](https://github.com/daileyo/gws/commit/848e7b22c25dee12fed322c21673af5074f7a52c))

## [2.18.0](https://github.com/daileyo/gws/compare/v2.17.1...v2.18.0) (2026-03-13)


### Features

* add compact status display with -s flag ([e87c9f1](https://github.com/daileyo/gws/commit/e87c9f119524a9b599854ebe54c6a4e21cb84670))

## [2.17.1](https://github.com/daileyo/gws/compare/v2.17.0...v2.17.1) (2026-03-12)


### Bug Fixes

* --go deprecation warning ([84bc1f5](https://github.com/daileyo/gws/commit/84bc1f59a7f58fd87e84b80ad34e0d040a78b2b0))

## [2.17.0](https://github.com/daileyo/gws/compare/v2.16.0...v2.17.0) (2026-03-10)


### Features

* implement consistent lowercase/uppercase flag convention for list command ([40e8d5f](https://github.com/daileyo/gws/commit/40e8d5fb6f0c4ca90426dd157b75770b1707d980))


### Bug Fixes

* resolve unparam lint for reassignTrailingArg ([4bfc193](https://github.com/daileyo/gws/commit/4bfc1932679a2e57b3dce89ef319b67b9da74150))
* use exact matching for tag, type, and visibility filters ([840f432](https://github.com/daileyo/gws/commit/840f4321ba1345e0bc898ceb1a45bf0a13e651e7))

## [2.16.0](https://github.com/daileyo/gws/compare/v2.15.0...v2.16.0) (2026-03-09)


### Features

* add --remote-raw/-R flag and format remote URLs by default ([ebf2186](https://github.com/daileyo/gws/commit/ebf2186101845db34681387f733851f2bb417346))
* add FormatRemoteURL utility for clean HTTPS URL display ([e89b5d7](https://github.com/daileyo/gws/commit/e89b5d7c7f1662b4ccbfd0fa7dd4d10932a65372))

## [2.15.0](https://github.com/daileyo/gws/compare/v2.14.0...v2.15.0) (2026-03-06)


### Features

* add status column alignment, truncation, and colored icons ([4696958](https://github.com/daileyo/gws/commit/4696958aedf5e459311ef859064e58cfcd5de607))
* reorder status icons to behind, ahead, clean/dirty ([99020da](https://github.com/daileyo/gws/commit/99020da86d12c9628887df8afc3e5d1d6bfb3cb5))


### Bug Fixes

* address lint issues in status alignment code ([fcacac7](https://github.com/daileyo/gws/commit/fcacac74fb504f76caab61a793a4df5617f89677))

## [2.14.0](https://github.com/daileyo/gws/compare/v2.13.0...v2.14.0) (2026-03-06)


### Features

* add background prefetch and stale cache support ([5547ccd](https://github.com/daileyo/gws/commit/5547ccdf014395b6200a1003242cb1501cc0de4f))
* add parallel status fetching with configurable worker pool ([5e4fe6a](https://github.com/daileyo/gws/commit/5e4fe6a78b363eb289c7d3ea8fc8c32b86aed16e))
* add progress spinner with count during status fetching ([dbc26ae](https://github.com/daileyo/gws/commit/dbc26aeeb1b3e5ed946a6cf8b130b979d36c3c78))


### Performance Improvements

* switch status backend from go-git to git CLI ([0e80cab](https://github.com/daileyo/gws/commit/0e80cab4388a447ea0b17c92b237bd61d042c6ec))

## [2.13.0](https://github.com/daileyo/gws/compare/v2.12.0...v2.13.0) (2026-03-06)


### Features

* add JSON column selection and clean help text for list command ([0eeb304](https://github.com/daileyo/gws/commit/0eeb304a7272b7891656b60c0012ccfb76f1f73b))
* add multi-column default view for list command ([04f6cd9](https://github.com/daileyo/gws/commit/04f6cd9fd31ff1194341e3f3b9ca7a242777d299))
* default gws (no args) to list repos in multi-column view ([bb16519](https://github.com/daileyo/gws/commit/bb1651927b66ddc7670cb18c81b3f389e5888ece))
* implement dual-purpose flag infrastructure for list command ([194762d](https://github.com/daileyo/gws/commit/194762d8f90ce1b5e3210d3d5ed47950cecc0877))


### Bug Fixes

* cap multi-column width to ensure at least 3 rows ([9ad5ccc](https://github.com/daileyo/gws/commit/9ad5ccc7f77ba256acdc58e74418de29bdc48aff))
* handle no-args case in shell function for gws default list ([66f72f5](https://github.com/daileyo/gws/commit/66f72f54c97aa84b0a5f51e17f02e1788dc96c3a))
* use column-first (top-to-bottom) layout for multi-column view ([b7efdbe](https://github.com/daileyo/gws/commit/b7efdbee4a3d9d8f11dc54ddb5f64825045b5e3f))

## [2.12.0](https://github.com/daileyo/gws/compare/v2.11.2...v2.12.0) (2026-03-05)


### Features

* add --remote/-r flag to list command for displaying origin URL ([3005260](https://github.com/daileyo/gws/commit/3005260c8fabe6a278217ffefdce3e19dec7d572))
* add asterisk indicator for multiple remotes via live git inspection ([91bd679](https://github.com/daileyo/gws/commit/91bd679337b41ddc1d11661b5726a4054490d59a))
* add remote info to JSON output with has_multiple_remotes field ([532f2b5](https://github.com/daileyo/gws/commit/532f2b512619743bc615b09186bac3e2abed83fa))


### Bug Fixes

* pad PATH column when REMOTE column follows for proper alignment ([4b78c95](https://github.com/daileyo/gws/commit/4b78c95006e524173183af16bd37cb7c45bf2ae7))
* use dynamic width for REMOTE column separator ([81d8124](https://github.com/daileyo/gws/commit/81d8124029c1267236c1ab020fb6ea01ce2f4b47))

## [2.11.2](https://github.com/daileyo/gws/compare/v2.11.1...v2.11.2) (2026-03-04)


### Bug Fixes

* rename Homebrew formula from gws to git-workspace ([327ef59](https://github.com/daileyo/gws/commit/327ef596b9a91261e668e2e42a5a87f1c23d4111))

## [2.11.1](https://github.com/daileyo/gws/compare/v2.11.0...v2.11.1) (2026-03-04)


### Bug Fixes

* run GoReleaser from release-please workflow ([890cd6e](https://github.com/daileyo/gws/commit/890cd6e2651e09c51b55db4ccf33109016b49652))

## [2.11.0](https://github.com/daileyo/gws/compare/v2.10.1...v2.11.0) (2026-03-04)


### Features

* configure GoReleaser to auto-publish Homebrew formula ([86f6dec](https://github.com/daileyo/gws/commit/86f6dec9c84d84b74b206885479017589b5411a2))


### Bug Fixes

* correct Homebrew formula license from MIT to Apache-2.0 ([bddb71f](https://github.com/daileyo/gws/commit/bddb71fadff1775dffc7095221fd67b20fc04050))

## [2.10.1](https://github.com/daileyo/gws/compare/v2.10.0...v2.10.1) (2026-03-03)


### Bug Fixes

* fix shell navigation hanging on multiple repo matches ([27d1b20](https://github.com/daileyo/gws/commit/27d1b20c7b59f3f0edfc2cd569f2294e19bc0990))

## [2.10.0](https://github.com/daileyo/gws/compare/v2.9.3...v2.10.0) (2026-03-03)


### Features

* add parent navigation command and shell variants ([fede029](https://github.com/daileyo/gws/commit/fede029666324dcc6e69ddab32fbe5042b9bba47))

## [2.9.3](https://github.com/daileyo/gws/compare/v2.9.2...v2.9.3) (2026-03-03)


### Bug Fixes

* fix user profile refresh ([e3951b5](https://github.com/daileyo/gws/commit/e3951b5c36d4c8eaa944194974bfdb72ea16c5d0))

## [2.9.2](https://github.com/daileyo/gws/compare/v2.9.1...v2.9.2) (2026-03-03)


### Bug Fixes

* correct the shorthand for recursive ([9666006](https://github.com/daileyo/gws/commit/9666006b863f62976da74afc6ef8f26f5d83f515))

## [2.9.1](https://github.com/daileyo/gws/compare/v2.9.0...v2.9.1) (2026-03-02)


### Bug Fixes

* path-first validation and symlink-aware workspace scan in refresh ([74af9f0](https://github.com/daileyo/gws/commit/74af9f0355ba72dda6036108911eeaf548de8383))

## [2.9.0](https://github.com/daileyo/gws/compare/v2.8.0...v2.9.0) (2026-03-02)


### Features

* add --path and --repo targeting flags to tag add/remove ([dc775cc](https://github.com/daileyo/gws/commit/dc775cce594a5f5e2de480bc09aa9849ee47e68e))
* add findRepositoriesWithFilters with path and repo filter logic ([868cab5](https://github.com/daileyo/gws/commit/868cab5c8f00b676dda63ed0e4c00780d196f751))
* complete tag path targeting - proofs and ci verification ([35d53df](https://github.com/daileyo/gws/commit/35d53dfaf83a1623abc45e7deea0ccc2dc4c3047))

## [2.8.0](https://github.com/daileyo/gws/compare/v2.7.0...v2.8.0) (2026-03-02)


### Features

* add short-flag aliases (-a, -d, -l, -s) to user subcommand ([56e8f0f](https://github.com/daileyo/gws/commit/56e8f0fa0f715d0cca45f1cb9e70b78153175bb0))
* add short-flag tests and tab completion for user subcommand ([e14ce5d](https://github.com/daileyo/gws/commit/e14ce5d35f54020b928f34f83fdd63f199f8baf2))
* clean up shared state and update help text ([b13864f](https://github.com/daileyo/gws/commit/b13864ff8177c2c5579293cf0061596219e4c8fe))
* deprecate root-level user flags to deprecated.go ([0957660](https://github.com/daileyo/gws/commit/0957660d8f93b96f4959805eca0e7f0b43a5b98a))


### Bug Fixes

* prevent TestUserProfileCompletion from corrupting real config ([9a49d7e](https://github.com/daileyo/gws/commit/9a49d7ea534131d7bf5768539dfab5cd178b9083))

## [2.7.0](https://github.com/daileyo/gws/compare/v2.6.0...v2.7.0) (2026-03-02)


### Features

* add tab completion for tag operations ([e8c3497](https://github.com/daileyo/gws/commit/e8c3497aca7f85f190437586ed9b5d73650f6326))
* create tag subcommand with add/remove sub-operations ([1af4150](https://github.com/daileyo/gws/commit/1af41501077136577c26598a4bb818f1a4deebeb))
* deprecate old tag flags and clean up root command ([d5639b9](https://github.com/daileyo/gws/commit/d5639b9f53fac05bc6786b9a9ea7ec6ea9875cfb))


### Bug Fixes

* resolve lint issues in main.go and deprecated_test.go ([2bf107c](https://github.com/daileyo/gws/commit/2bf107cd2792837380f5e605953869ea81c391dd))
* validate deprecated tag flag args before workspace check ([b556d2e](https://github.com/daileyo/gws/commit/b556d2ef01686a06fe59c1d6f684fa283c1c0c86))

## [2.6.0](https://github.com/daileyo/gws/compare/v2.5.0...v2.6.0) (2026-03-02)


### Features

* add deprecation layer for old flag forms ([47af8ea](https://github.com/daileyo/gws/commit/47af8ea36f2c3169e2923883ffb84feed9f5beb3))
* create core subcommands (list, init, add, refresh, print-workspace) ([6ef1318](https://github.com/daileyo/gws/commit/6ef1318d8cf7b9e7b86273c444a1f2fe019d5763))
* scope filter flags to list subcommand ([cccb7a5](https://github.com/daileyo/gws/commit/cccb7a53505d73fb708b6b3a3854c4044da54e5e))
* update shell templates to route subcommand names to binary ([727c62b](https://github.com/daileyo/gws/commit/727c62b1dfb93a8c82aabc06da5c70bb95d2d47a))


### Bug Fixes

* resolve lint issues in main.go and deprecated_test.go ([cb06381](https://github.com/daileyo/gws/commit/cb06381755b83e878f5a5024233cc4dab79f958c))

## [2.5.0](https://github.com/daileyo/gws/compare/v2.4.1...v2.5.0) (2026-03-02)


### Features

* add repo user update/delete commands and profile sync ([a067e0e](https://github.com/daileyo/gws/commit/a067e0e6782a35a59493d1ea12114eab15d2ea6c))

## [2.4.1](https://github.com/daileyo/gws/compare/v2.4.0...v2.4.1) (2026-02-27)


### Bug Fixes

* correct license badge in docds ([697ccbe](https://github.com/daileyo/gws/commit/697ccbecd41e0f8fdd0488526af625c388a01dab))

## [2.4.0](https://github.com/daileyo/gws/compare/v2.3.0...v2.4.0) (2026-02-27)


### Features

* add logo and tagline to README ([5ec212d](https://github.com/daileyo/gws/commit/5ec212d93e261c375f7ddba377e707e8636dbe8c))
* add logo to mkdocs nav bar, favicon, and homepage hero ([84f8b84](https://github.com/daileyo/gws/commit/84f8b84a4c78e9637287f06476d20ee3dce4c02f))
* add optimized logo variants for branding ([09a19e5](https://github.com/daileyo/gws/commit/09a19e54acf6f4acc8e7eba1ef42810aa5bdb93f))
* configure mkdocs theme with branded colors and fonts ([c60b0aa](https://github.com/daileyo/gws/commit/c60b0aa4b9eea74e91dc9384ff48eae82a3435b5))

## [2.3.0](https://github.com/daileyo/gws/compare/v2.2.0...v2.3.0) (2026-02-27)


### Features

* add commit-msg hook for conventional commit alignment ([f0ef8bd](https://github.com/daileyo/gws/commit/f0ef8bda2f9311821e62f7e1a8019ce7d87eee77))
* update setup-hooks messaging and document in README ([d7dd3d7](https://github.com/daileyo/gws/commit/d7dd3d7ca598d93ffe7016bb184b6d915ed539c8))

## [2.2.0](https://github.com/daileyo/gws/compare/v2.1.1...v2.2.0) (2026-02-27)


### Features

* **config:** extend model with user profiles and repository user fields ([5fd4bf7](https://github.com/daileyo/gws/commit/5fd4bf7f9b8eaca8c68375f4d51d982c82f6e5dd))
* **git:** implement user detection from repositories ([11f7310](https://github.com/daileyo/gws/commit/11f73103a7295f6d91c28c48a45a48be93f72768))
* **list:** add --user flag to display git user information ([00476f9](https://github.com/daileyo/gws/commit/00476f9adf6cf402a4eaf55e3b0a405bb5358cb9))
* **user:** add assign and sync commands for repository user management ([3dda5f1](https://github.com/daileyo/gws/commit/3dda5f14c774c463b9bf19d2a5e5b6e1125d7547))
* **user:** add user profile management commands ([418e3ed](https://github.com/daileyo/gws/commit/418e3ed2edeb64de0e1a2235ab450c0720137876))
* **user:** implement profile auto-detection from gitconfig ([c560843](https://github.com/daileyo/gws/commit/c5608432b5b1118d07cd0c436f62364bcd8c7fe2))


### Bug Fixes

* resolve linting issues ([c846fe0](https://github.com/daileyo/gws/commit/c846fe0b26585215f4a73ea01b5dd33e111e6027))

## [2.1.1](https://github.com/daileyo/gws/compare/v2.1.0...v2.1.1) (2026-02-26)


### Bug Fixes

* correct project structure paths and remove premature brew install references ([ec1208e](https://github.com/daileyo/gws/commit/ec1208efb1ef1f52193e5735fa923cc7fd9187d1))

## [2.1.0](https://github.com/daileyo/gws/compare/v2.0.0...v2.1.0) (2026-02-26)


### Features

* refactor init/add, shell completions, and shell-init integration ([#6](https://github.com/daileyo/gws/issues/6)) ([e9d1564](https://github.com/daileyo/gws/commit/e9d156448ac012460050bca78dd2f06ff4d63bb5))

## [2.0.0](https://github.com/daileyo/gws/compare/v1.1.0...v2.0.0) (2026-02-25)


### ⚠ BREAKING CHANGES

* binary is now named git-workspace

### Features

* add repository navigation and rename binary to git-workspace ([#4](https://github.com/daileyo/gws/issues/4)) ([fb4a6ff](https://github.com/daileyo/gws/commit/fb4a6ff133241fb1f640278fe83a6bfe351f6713))

## [1.1.0](https://github.com/daileyo/gws/compare/v1.0.0...v1.1.0) (2026-02-13)


### Features

* rework CLI to flag-based command pattern ([#2](https://github.com/daileyo/gws/issues/2)) ([7256c97](https://github.com/daileyo/gws/commit/7256c97df4a19d18317dad84f2b40ff59b1d2a32))

## 1.0.0 (2026-01-06)


### Features

* complete questionare ([9d83e51](https://github.com/daileyo/gws/commit/9d83e51154943a387e89279b1ccde362d82471ca))
* generate initial spec ([49666c7](https://github.com/daileyo/gws/commit/49666c7ff4874050bcdd9d7807f832f38338fbc7))
* implement step 6 ([8787dae](https://github.com/daileyo/gws/commit/8787dae2d3bb6153297907cd0316b82e7f38d520))
* implement task 4 ([1b95999](https://github.com/daileyo/gws/commit/1b95999c99984a2496517342d35dfe389ee58abb))
* project setup and basic CLI framework ([8eb3ed1](https://github.com/daileyo/gws/commit/8eb3ed15636660a9d5dd6056dbc5e2653a3983d5))
* repository discovery and workspace initialization ([15cf8b4](https://github.com/daileyo/gws/commit/15cf8b474d9ca1c3beb0e5f5298cd6f8a4723343))


### Bug Fixes

* addres linting issues ([05c57bc](https://github.com/daileyo/gws/commit/05c57bcbe9e30ac80295e67ff300fdacb550e652))
* update target go version ([91636ae](https://github.com/daileyo/gws/commit/91636ae24f3436e2f6ec643947784cb8e6f9efe9))
