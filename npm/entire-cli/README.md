# @entireio/cli

NPM wrapper for [Entire CLI](https://github.com/entireio/cli).

On install, this package downloads the matching `entire` binary from GitHub Releases and exposes it as the `entire` command.

## Usage

```bash
npx @entireio/cli@latest status
```

## Notes

- Supported platforms: macOS and Linux (`amd64` / `arm64`).
- For development package versions (`0.0.0*`), the installer resolves the latest GitHub release automatically.

## Environment variables

- `ENTIRE_NPM_SKIP_DOWNLOAD=1`: Skip binary download in `postinstall`.
- `ENTIRE_NPM_VERSION=<version>`: Override release version to install.
- `ENTIRE_NPM_REPO=<owner/repo>`: Override release repository (default: `entireio/cli`).
- `GITHUB_TOKEN=<token>`: Optional token for GitHub API/release download requests.
