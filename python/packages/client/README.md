<div align="center">

# ![valbridge](https://img.shields.io/static/v1?label=&message=valbridge&color=3776AB&style=for-the-badge&logo=python&logoColor=white)

Runtime client for valbridge-generated Python validators with type-safe schema lookup.

<a href="https://pypi.org/project/valbridge/"><img src="https://img.shields.io/pypi/v/valbridge?style=flat&logo=pypi&logoColor=white" alt="PyPI" /></a>
<a href="https://github.com/vectorfy-co/valbridge/blob/main/LICENSE"><img src="https://img.shields.io/github/license/vectorfy-co/valbridge?style=flat" alt="License" /></a>

</div>

---

## Installation

```bash
pip install valbridge
# or
uv add valbridge
```

## Quick start

```python
from valbridge import create_valbridge
from _valbridge import schemas

valbridge = create_valbridge(schemas)

# Full namespace:id lookup
user_validator = valbridge("user:Profile")

# Validate data (Pydantic under the hood)
user_validator.validate_python({"name": "Alice", "email": "alice@example.com"})
```

## Default namespace

```python
valbridge = create_valbridge(schemas, default_namespace="user")

# Shorthand for "user:Profile"
profile = valbridge("Profile")

# Full key for other namespaces
config = valbridge("settings:AppConfig")
```

## Key behaviors

- **Runtime lookup** -- only entries with a runtime validator appear in the schemas object
- **Type safety** -- invalid keys raise clear errors pointing to `valbridge generate`
- **Pydantic integration** -- generated validators are native Pydantic BaseModel classes

## Related packages

| Package | Purpose |
| --- | --- |
| [`valbridge-core`](https://pypi.org/project/valbridge-core/) | Core IR and JSON Schema parser |
| [`valbridge-pydantic`](https://pypi.org/project/valbridge-pydantic/) | Pydantic adapter for code generation |
| [`valbridge-cli`](https://pypi.org/project/valbridge-cli/) | CLI to generate validators |
| [`@vectorfyco/valbridge`](https://www.npmjs.com/package/@vectorfyco/valbridge) | TypeScript equivalent |

## Learn more

- [GitHub repository](https://github.com/vectorfy-co/valbridge)
- [Full documentation](https://github.com/vectorfy-co/valbridge#readme)
