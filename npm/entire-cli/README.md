# @entireio/cli wrapper

Wrapper for [Entire CLI](https://github.com/entireio/cli), intended for `npx` workflows.

When used from a full Git checkout (for example `npx github:<owner>/cli#main`), postinstall builds `entire` from source by default.  
Otherwise it downloads a release binary and exposes it as the `entire` command.

## Usage

```bash
npx -y github:entireio/cli#main status
```

## Notes

- Supported platforms: macOS and Linux (`amd64` / `arm64`).

## Environment variables

- `ENTIRE_NPM_SKIP_DOWNLOAD=1`: Skip binary download in `postinstall`.
- `ENTIRE_NPM_VERSION=<version>`: Override release version to install.
- `ENTIRE_NPM_REPO=<owner/repo>`: Override release repository (default: `entireio/cli`).
- `ENTIRE_NPM_BUILD_FROM_SOURCE=0`: Force release-binary mode even when source checkout is available.
- `GITHUB_TOKEN=<token>`: Optional token for GitHub API/release download requests.
