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

### Changed

- Do not force GOGC=800, let inputs decide and user have final word [#13](https://github.com/AdRoll/baker/pull/13)
- Move aws-specific utilities into a new `awsutils` package [#14](https://github.com/AdRoll/baker/pull/14)
- Outputs' `Run()` returns an error [#21](https://github.com/AdRoll/baker/pull/21)
- Fix 2 panics: ValidateRecord and errUnsuportedURLScheme [#29](https://github.com/AdRoll/baker/pull/29)
- Remove datadog-specific code from [general] section. Instead add [metrics] which can be extended with baker.MetricsClient interfaces. [#34](https://github.com/AdRoll/baker/pull/34)

### Removed

- output: remove the Files output in favor of the more generic FileWriter [#31](https://github.com/AdRoll/baker/pull/31)

### Maintenance

- input: Fixes `List` input not managing S3 "folders" [#35](https://github.com/AdRoll/baker/pull/35)
- input: with [#35](https://github.com/AdRoll/baker/pull/35) we introduced a regression that has been fixed with [#39](https://github.com/AdRoll/baker/pull/39)
- upload: fixes a severe concurrency issue in the uploader [#38](https://github.com/AdRoll/baker/pull/38)
