<div align="center">

# ![valbridge-cli](https://img.shields.io/static/v1?label=&message=valbridge-cli&color=3776AB&style=for-the-badge&logo=pypi&logoColor=white)

Installable valbridge CLI launcher for pip and uvx. Downloads the correct platform binary automatically.

<a href="https://pypi.org/project/valbridge-cli/"><img src="https://img.shields.io/pypi/v/valbridge-cli?style=flat&logo=pypi&logoColor=white" alt="PyPI" /></a>
<a href="https://github.com/vectorfy-co/valbridge/blob/main/LICENSE"><img src="https://img.shields.io/github/license/vectorfy-co/valbridge?style=flat" alt="License" /></a>

</div>

---

## Zero-install usage

Run directly without installing globally:

```bash
uvx valbridge-cli generate
uvx valbridge-cli generate --help
```

## Global installation

```bash
# pip
pip install valbridge-cli

# uv (recommended)
uv tool install valbridge-cli
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
| [`@vectorfyco/valbridge-cli`](https://www.npmjs.com/package/@vectorfyco/valbridge-cli) | npm/npx CLI launcher (same binary, different installer) |
| [`valbridge`](https://pypi.org/project/valbridge/) | Runtime client for generated validators |
| [`valbridge-pydantic`](https://pypi.org/project/valbridge-pydantic/) | Pydantic adapter for code generation |

## Learn more

- [GitHub repository](https://github.com/vectorfy-co/valbridge)
- [Full documentation](https://github.com/vectorfy-co/valbridge#readme)
