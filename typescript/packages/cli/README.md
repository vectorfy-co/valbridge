<div align="center">

# ![valbridge-cli](https://img.shields.io/static/v1?label=&message=%40vectorfyco%2Fvalbridge-cli&color=3178C6&style=for-the-badge&logo=npm&logoColor=white)

Installable valbridge CLI launcher for npm and npx. Downloads the correct platform binary automatically.

<a href="https://www.npmjs.com/package/@vectorfyco/valbridge-cli"><img src="https://img.shields.io/npm/v/@vectorfyco/valbridge-cli?style=flat&logo=npm&logoColor=white" alt="npm" /></a>
<a href="https://github.com/vectorfy-co/valbridge/blob/main/LICENSE"><img src="https://img.shields.io/github/license/vectorfy-co/valbridge?style=flat" alt="License" /></a>

</div>

---

## Zero-install usage

Run directly without installing globally:

```bash
npx -y @vectorfyco/valbridge-cli generate
npx -y @vectorfyco/valbridge-cli generate --help
```

## Global installation

```bash
# npm
npm install -g @vectorfyco/valbridge-cli

# pnpm
pnpm add -g @vectorfyco/valbridge-cli
```

Then use the `valbridge` command directly:

```bash
valbridge generate
valbridge extract --schema user:Profile
valbridge --help
```

## How it works

The launcher downloads the matching `valbridge` Go binary for the current platform on first run and caches it locally. Subsequent runs use the cached binary.

## Environment variables

| Variable | Description |
| --- | --- |
| `VALBRIDGE_CLI_BIN` | Override the binary path. Set to a local build to skip the download. |

## Related packages

| Package | Purpose |
| --- | --- |
| [`valbridge-cli`](https://pypi.org/project/valbridge-cli/) | Python/uvx CLI launcher (same binary, different installer) |
| [`@vectorfyco/valbridge`](https://www.npmjs.com/package/@vectorfyco/valbridge) | Runtime client for generated validators |
| [`@vectorfyco/valbridge-zod`](https://www.npmjs.com/package/@vectorfyco/valbridge-zod) | Zod adapter for code generation |

## Learn more

- [GitHub repository](https://github.com/vectorfy-co/valbridge)
- [Full documentation](https://github.com/vectorfy-co/valbridge#readme)
