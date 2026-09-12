# Changelog

## 0.10.1 - 2026-09-12

- New themes @joshbeard (#152)

## 0.10.0 - 2026-09-10

### Changes

- Load environment variables with viper @huncrys (#144)
- Honor sort_by field when order is desc @huncrys (#142)

### Chores

- Migrate to aws-sdk-go-v2 @huncrys (#143)
- chore(deps): migrate to charm.land/log/v2 @huncrys (#145)
- build(deps): bump the go group with 3 updates @[dependabot[bot]](https://github.com/apps/dependabot) (#151)
- chore: go and gh actions updates @joshbeard (#150)
- Chore/updates 2026 07 14 @joshbeard (#135)
- Chore/updates 2026 07 @joshbeard (#130)
- Dependency updates @joshbeard (#127)
- build(deps): bump docker/setup-buildx-action from 3 to 4 @[dependabot[bot]](https://github.com/apps/dependabot) (#117)
- build(deps): bump actions/github-script from 7 to 9 @[dependabot[bot]](https://github.com/apps/dependabot) (#119)
- build(deps): bump stefanzweifel/git-auto-commit-action from 6 to 7 @[dependabot[bot]](https://github.com/apps/dependabot) (#118)
- build(deps): bump docker/setup-qemu-action from 3 to 4 @[dependabot[bot]](https://github.com/apps/dependabot) (#116)
- build(deps): bump github/codeql-action from 3 to 4 @[dependabot[bot]](https://github.com/apps/dependabot) (#115)
- build(deps): bump aws-actions/configure-aws-credentials from 4 to 6 @[dependabot[bot]](https://github.com/apps/dependabot) (#109)
- build(deps): bump actions/setup-go from 5 to 6 @[dependabot[bot]](https://github.com/apps/dependabot) (#108)
- maint: dependency updates @joshbeard (#114)
- build(deps): bump github.com/go-viper/mapstructure/v2 from 2.3.0 to 2.4.0 in the go_modules group across 1 directory @[dependabot[bot]](https://github.com/apps/dependabot) (#112)

## 0.9.0 - 2026-01-10

- feat: directory stats @huncrys (#90)
- chore: build(deps): bump github.com/go-viper/mapstructure/v2 from 2.2.1 to 2.3.0 in the go_modules group @[dependabot[bot]](https://github.com/apps/dependabot) (#100)
- chore: build(deps): bump stefanzweifel/git-auto-commit-action from 5 to 6 @[dependabot[bot]](https://github.com/apps/dependabot) (#98)

## 0.8.1 - 2025-06-17

* Expose link-up-from-root args in GH action

## 0.8.0 - 2025-06-17

- feat: link up from root @joshbeard (#99)
- feat: align S3 and Docker timezone handling with local filesystem @huncrys (#89)
- feat: allow specifying custom S3 endpoint @huncrys (#88)
- feat/generate demos (ci) @joshbeard (#93)

## 0.7.2 - 2025-05-10

* ci fixes for multiarch builds

## 0.7.1 - 2025-05-10

- fix: respect IndexFile and LinkToIndexes configuration for "Go Up" link @huncrys (#86)
- build(deps): bump github.com/aws/aws-sdk-go from 1.55.6 to 1.55.7 @[dependabot[bot]](https://github.com/apps/dependabot) (#84)

## 0.7.0 - 2025-04-24

- feat: skip index @joshbeard (#81)
- maint: dependency updates
- ci: build on dev PRs; misc updates

## 0.6.1 - 2025-03-15

- fix: relative paths with base URL @joshbeard (#72)

maintenance:

- linting: check for unused, update arg @joshbeard (#73)
- build(deps): bump crazy-max/ghaction-import-gpg from 6.1.0 to 6.2.0 @[dependabot[bot]](https://github.com/apps/dependabot) (#64)
- build(deps): bump goreleaser/goreleaser-action from 6.0.0 to 6.1.0 @[dependabot[bot]](https://github.com/apps/dependabot) (#65)

## 0.6.0 - 2025-03-15

- feat: add built-in themes @joshbeard (#71)

## 0.5.2 - 2025-03-15

- fix: expose noindex to action, entrypoint @joshbeard (#70)

## 0.5.1 - 2025-03-15

- fix: CI package tests @joshbeard (#69)

## 0.5.0 - 2025-03-15

- feat: Add noindex file functionality to skip dirs @joshbeard (#68)
- build(deps): bump github.com/golangci/golangci-lint from 1.59.1 to 1.60.1 @[dependabot[bot]](https://github.com/apps/dependabot) (#56)
- build(deps): bump github.com/aws/aws-sdk-go from 1.54.20 to 1.55.5 @[dependabot[bot]](https://github.com/apps/dependabot) (#54)
- build(deps): bump golang.org/x/vuln from 1.1.2 to 1.1.3 @[dependabot[bot]](https://github.com/apps/dependabot) (#52)
- build(deps): bump github.com/aws/aws-sdk-go from 1.54.11 to 1.54.20 @[dependabot[bot]](https://github.com/apps/dependabot) (#51)
- build(deps): bump github.com/aws/aws-sdk-go from 1.54.6 to 1.54.11 @[dependabot[bot]](https://github.com/apps/dependabot) (#48)

## 0.4.2 - 2024-06-26

- build(deps): bump github.com/spf13/cobra from 1.8.0 to 1.8.1 @dependabot (#46)
- build(deps): bump golang.org/x/vuln from 1.1.0 to 1.1.2 @dependabot (#43)
- build(deps): bump goreleaser/goreleaser-action from 5.0.0 to 6.0.0 @dependabot (#41)
- build(deps): bump github.com/spf13/viper from 1.18.2 to 1.19.0 @dependabot (#39)
- build(deps): bump securego/gosec from 2.19.0 to 2.20.0 @dependabot (#33)
- build(deps): bump github.com/golangci/golangci-lint from 1.58.0 to 1.59.1 @dependabot (#44)
- build(deps): bump github.com/aws/aws-sdk-go from 1.52.2 to 1.54.6 @dependabot (#47)
- build(deps): bump github.com/aws/aws-sdk-go from 1.51.25 to 1.52.2 @dependabot (#28)
- build(deps): bump github.com/golangci/golangci-lint from 1.57.2 to 1.58.0 @dependabot (#27)
- build(deps): bump golang.org/x/vuln from 1.0.4 to 1.1.0 @dependabot (#24)
- build(deps): bump github.com/aws/aws-sdk-go from 1.51.16 to 1.51.25 @dependabot (#25)
- build(deps): bump github.com/aws/aws-sdk-go from 1.51.11 to 1.51.16 @dependabot (#20)
- build(deps): bump github.com/golangci/golangci-lint from 1.56.2 to 1.57.2 @dependabot (#18)
- build(deps): bump github.com/aws/aws-sdk-go from 1.51.6 to 1.51.11 @dependabot (#19)
- build(deps): bump github.com/charmbracelet/log from 0.3.1 to 0.4.0 @dependabot (#16)
- build(deps): bump github.com/aws/aws-sdk-go from 1.50.35 to 1.51.6 @dependabot (#17)
- build(deps): bump google.golang.org/protobuf from 1.31.0 to 1.33.0 @dependabot (#13)
- build(deps): bump github.com/aws/aws-sdk-go from 1.50.30 to 1.50.35 @dependabot (#12)
- build(deps): bump github.com/stretchr/testify from 1.8.4 to 1.9.0 @dependabot (#11)
- build(deps): bump github.com/aws/aws-sdk-go from 1.50.25 to 1.50.30 @dependabot (#10)
- build(deps): bump github.com/aws/aws-sdk-go from 1.50.20 to 1.50.25 @dependabot (#9)
- maint: remove unused tools @joshbeard (#8)

## 0.4.1 - 2024-02-25

- Update GitHub action for sort, order @joshbeard (#7)
- build(deps): bump the go_modules group group with 1 update @dependabot (#5)

## 0.4.0 - 2024-02-25

- feature: sorting and tests @joshbeard (#6)
- ci: Add release drafter, changelog, and test stub-ins @joshbeard (#4)

## 0.3.1 - 2024-02-21

* Add package tests

## 0.3.0 - 2024-02-21

* Sign artifacts with GPG
* Create RPM, Deb, APK, Arch Linux packages
* Publish image to GitHub package registry
* Properly set the `version` variable at build
* Use `icon` span on parent directory icon

## 0.2.1 - 2024-02-18

* Pass extra arguments to entrypoint command in Docker image

## 0.2.0 - 2024-02-17

* Refactored and added support for local directory sources.
* Configuration options and CLI arguments have changed.
* Renamed from "s3-web-indexer" to "web-indexer".
* Use MIT license

## 0.1.0 - 2024-02-04

* Initial release
