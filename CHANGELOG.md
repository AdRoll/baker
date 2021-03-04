# Changelog

All notable changes to Baker will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased

### Added

- upload: add S3 uploader component [#15](https://github.com/AdRoll/baker/pull/15)
- filter: add ClearFields filter [#19](https://github.com/AdRoll/baker/pull/19)
- output: add Stats output [#23](https://github.com/AdRoll/baker/pull/23)
- filter: add SetStringFromURL filter [#28](https://github.com/AdRoll/baker/pull/28)
- output: add FileWriter output in replacement of Files output  [#31](https://github.com/AdRoll/baker/pull/31)
- upload: s3: add `ExitOnError` configuration [#27](https://github.com/AdRoll/baker/pull/27)
- uploads now return an error instead of panicking and baker deals with it [#27](https://github.com/AdRoll/baker/pull/27)
- general: replace `${KEY}` in the TOML conf with the `$KEY` env var [#24](https://github.com/AdRoll/baker/pull/24)
- input: add KCL input. [#36](https://github.com/AdRoll/baker/pull/36)
- filter: add RegexMatch filter. [#37](https://github.com/AdRoll/baker/pull/37)
- filter: add NotNull filter [#43](https://github.com/AdRoll/baker/pull/43)
- filter: add Concatenate filter [#28](https://github.com/AdRoll/baker/pull/33)
- Required configuration fields are now handled by Baker rather than by each component. [#41](https://github.com/AdRoll/baker/pull/41)
- filter: add TimestampRange filter [#46](https://github.com/AdRoll/baker/pull/46)
- filter: add ReplaceFields filter [#49](https://github.com/AdRoll/baker/pull/49)
- filter: add Timestamp filter [#54](https://github.com/AdRoll/baker/pull/54)
- Add Record.Copy method [#53](https://github.com/AdRoll/baker/pull/53)
- Add pkg/splitwriter and pkg/buffercache reusable components [#55](https://github.com/AdRoll/baker/pull/55)
- output: add SQLite output [#56](https://github.com/AdRoll/baker/pull/56)
- README: document KCL input [#59](https://github.com/AdRoll/baker/pull/59)
- Document how to specialize baker.LogLine [#63](https://github.com/AdRoll/baker/pull/63)
- Add `baker.MainCLI` [#73](https://github.com/AdRoll/baker/pull/73)
- Implement markdown rendering of component help/configuration [#80](https://github.com/AdRoll/baker/pull/80)
- Add `[fields]` section in TOML in which use can define field indexes <-> names mapping [#84](https://github.com/AdRoll/baker/pull/84)
- Add StringMatch filter which discards/keeps records based on the result of string comparisons  [#102](https://github.com/AdRoll/baker/pull/102)
- Add PartialClone filter [#107](https://github.com/AdRoll/baker/pull/107)
- Add `[validation]` section in TOML in which users can define record validation through regex [#122](https://github.com/AdRoll/baker/pull/122)
- Add ExpandJSON filter [#128](https://github.com/AdRoll/baker/pull/128)
- Add Hash filter with the support of md5 and sha256 algorithms [#130](https://github.com/AdRoll/baker/pull/130)
- Add MetadataLastModified filter [#133](https://github.com/AdRoll/baker/pull/133)
- Add Dedup filter [#143](https://github.com/AdRoll/baker/pull/143)
- Add LUA filter [#110](https://github.com/AdRoll/baker/pull/110)

### Changed

- Do not force GOGC=800, let inputs decide and user have final word [#13](https://github.com/AdRoll/baker/pull/13)
- Move aws-specific utilities into a new `awsutils` package [#14](https://github.com/AdRoll/baker/pull/14)
- Outputs' `Run()` returns an error [#21](https://github.com/AdRoll/baker/pull/21)
- Fix 2 panics: ValidateRecord and errUnsuportedURLScheme [#29](https://github.com/AdRoll/baker/pull/29)
- Remove datadog-specific code from [general] section. Instead add [metrics] which can be extended with baker.MetricsClient interfaces. [#34](https://github.com/AdRoll/baker/pull/34)
- Remove duration parameter from baker.Main [#62](https://github.com/AdRoll/baker/pull/62)
- TimestampRange filter accepts 'now' as range [#106](https://github.com/AdRoll/baker/pull/106)
- Standardize the components' structs names [#105](https://github.com/AdRoll/baker/pull/105)
- **Breaking** Change func FieldName to FieldNames (slice) as it allows to know the number of defined fields [#110](https://github.com/AdRoll/baker/pull/110)

### Removed

- output: remove the Files output in favor of the more generic FileWriter [#31](https://github.com/AdRoll/baker/pull/31)

### Fixed

- Fix a bug in `logline.Copy` [#64](https://github.com/AdRoll/baker/pull/64)
- Fix building on windows [#115](https://github.com/AdRoll/baker/issues/115)
- Fix `list_test` with file URI to be compatible with windows paths [#117](https://github.com/AdRoll/baker/pull/117)
- Fix `List` input, some `io.Reader`'s were left opened [#118](https://github.com/AdRoll/baker/pull/118)
- Fix some bugs in `s3.s3UploadFile` [#120](https://github.com/AdRoll/baker/pull/120)
- `SplitWriter` leaves some file descriptors open [#119](https://github.com/AdRoll/baker/pull/119) and [#121](https://github.com/AdRoll/baker/pull/121)
- `PrintHelper()` now supports map type as configuration parameter of a Baker component [#138](https://github.com/AdRoll/baker/pull/138)
- `List` input did not consider drive letter on Windows paths [#139](https://github.com/AdRoll/baker/pull/139)
- Do not insert newline after dots in generated help markdown [#140](https://github.com/AdRoll/baker/pull/140)

### Maintenance

- input: Fixes `List` input not managing S3 "folders" [#35](https://github.com/AdRoll/baker/pull/35)
- input: with [#35](https://github.com/AdRoll/baker/pull/35) we introduced a regression that has been fixed with [#39](https://github.com/AdRoll/baker/pull/39)
- upload: fixes a severe concurrency issue in the uploader [#38](https://github.com/AdRoll/baker/pull/38)
- remove `output.RawChanSize`
