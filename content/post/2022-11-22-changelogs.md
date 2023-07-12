+++
tags = ['cli', 'process', 'ci']
title = 'Changelog Driven Versioning'

+++

Versioning is one of the many hard problems when it comes to writing software.  There is no one correct way to do it, and all have various tradeoffs.

After reading [keep a changelog][keep-changelog], I was inspired to implement this into a couple of CLI tools that I am working on at the moment: [Flagon] (feature flags on the CLI, for CI usage), and [Cas] (Content Addressable Storage for Make), but I also wanted to solve my versioning and release process.

## Requirements

- Version number should be defined in only one place
- A changelog should be associated with the version number
- The binary should be able to print its version and changelog
- The release (Github Release in this case) should also have the changelog and version
- The commit released should be tagged with the version

I came up with an idea: **drive everything from the changelog.**

The changelog can be the source of truth: it contains the version number, date of release, and the actual changes within that version.  As the changelog is written in a standardised format it should be fairly easy to parse, and thus be handled by the binary itself.

## The Format

I decided to follow the format from [keep a changelog][keep-changelog] as it is pretty minimal, in markdown, and easily parsable with a regex.  As an example, here is one of the versions lifted from [flagon's changelog][flagon-changelog].

```markdown
# Changelog

## [0.0.1] - 2022-11-14

### Added

- Exit with code `0` if a flag is `true`, and `1` otherwise
- Add `--silent` flag, to suppress console information

### Changed

- Expand what information is written to traces
```

Each version entry follows the same format, which is parsable by a regex:

{{< git-embed
  user="Pondidum"
  repo="flagon"
  ref="master"
  file="version/changelog.go"
  start=13
  finish=13
>}}

The parser itself is very short, and the result is an array of `ChangelogEntry`, giving the `version`, `date`, and text of the changes.

## Using the changelog from the application

The changelog is embedded in the binary using the go `embed` package, and can then be exposed as CLI commands.  The application's `version` command exposes this information with several flags:

- no flags: print the version number and git short sha
- `--short`: only print the version number
  ```shell
  ./flagon version --short
  0.0.1
  ```
- `--changelog`: pretty print the current version's changelog entry
  ```shell
  ./flagon version --changelog
  ```
  ![flagon changelog as prettified markdown](/images/flagon-changelog.png)

- `--raw`: causes `--changelog` to print the markdown as written in the `changelog.md`
  ```shell
  ./flagon version --changelog
  0.0.1 - local
  ### Added

  - Exit with code `0` if a flag is `true`, and `1` otherwise
  - Add `--silent` flag, to suppress console information

  ### Changed

  - Expand what information is written to traces
  ```


## Using the changelog for Releases

In github actions when building the `main` branch, I use this to generate a version number, and write the current changelog entry to a temporary file:

```yaml
- name: Generate Release Notes
  if: github.ref_name == 'main'
  run: |
    echo "FLAGON_VERSION=$(./flagon version --short)" >> "${GITHUB_ENV}"
    ./flagon version --changelog --raw > release-notes.md
```

Which are then passed to the `action-gh-release` step:

```yaml
- name: Release
  if: github.ref_name == 'main'
  uses: softprops/action-gh-release@v1
  with:
    name: ${{ env.FLAGON_VERSION }}
    tag_name: ${{ env.FLAGON_VERSION }}
    body_path: release-notes.md
    files: flagon
```

Which makes my releases match the version number of the binary, and have the correct release notes.

## Further Work

This system isn't perfect (yet), but it works well for my projects.  I've considered extracting it into its own package, but so far with only two applications using it I haven't hit the [rule of 3 yet](https://en.wikipedia.org/wiki/Rule_of_three_(computer_programming)).

[keep-changelog]: https://keepachangelog.com/en/1.0.0/
[flagon]: https://github.com/pondidum/flagon
[flagon-changelog]: https://github.com/Pondidum/Flagon/blob/main/changelog.md
[cas]: https://github.com/pondidum/cas