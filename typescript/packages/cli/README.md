# @vectorfyco/valbridge-cli

Installable `valbridge` CLI launcher for npm and `npx`.

## Installation

```bash
npm install -g @vectorfyco/valbridge-cli
# or
pnpm add -g @vectorfyco/valbridge-cli
```

## Usage

```bash
valbridge --help
npx -y @vectorfyco/valbridge-cli generate --help
```

The launcher downloads the matching `valbridge` release binary for the current platform on first run and caches it locally.

## Test Override

Set `VALBRIDGE_CLI_BIN=/path/to/local/valbridge` to force the launcher to execute a local binary instead of downloading a release asset.
