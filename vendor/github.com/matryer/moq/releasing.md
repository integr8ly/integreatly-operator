# Releasing

This tool uses Go Releaser to manage release builds.

## Setup

Install Go Releaser.

```bash
brew install goreleaser/tap/goreleaser
```

* Make a [New personal access token on GitHub](https://github.com/settings/tokens/new) and set it as the `GITHUB_TOKEN` environment variable

## Releasing

Tag the repo:

```bash
$ git tag -a v0.1.0 -m "release tag."
$ git push origin v0.1.0
```

Then:

```bash
make release
```
