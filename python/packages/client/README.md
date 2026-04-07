# valbridge

Runtime client for valbridge-generated validators with type-safe schema lookup.

## Installation

```bash
pip install valbridge
```

## Usage

```python
from valbridge import create_valbridge
from _valbridge import schemas

valbridge = create_valbridge(schemas)

# Full key lookup
user_validator = valbridge("user:Profile")

# Validate data
user_validator.validate_python({"name": "Alice", "email": "alice@example.com"})
```
