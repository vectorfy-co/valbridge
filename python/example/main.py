"""valbridge example for Python runtime client usage."""

import _valbridge
from valbridge import create_valbridge

valbridge: "_valbridge.ValbridgeClient" = create_valbridge(_valbridge.schemas)

# ============================================
# Type extraction using type annotations
# ============================================

# Use "namespace:id" format to get schemas.
calendar = valbridge("user:Calendar")

# Validate with Pydantic TypeAdapter
valid_calendar = calendar.validate_python(
    {
        "dtstart": "2024-01-01",
        "summary": "New Year's Day",
    }
)

print(f"Valid calendar event: {valid_calendar.summary}")

# Try invalid data
try:
    calendar.validate_python({"invalid": "data"})
except Exception as e:
    print(f"Validation failed as expected: {type(e).__name__}")

# ============================================
# User schema
# ============================================

user = valbridge("user:User")
valid_user = user.validate_python(
    {
        "id": "123",
        "email": "alice@example.com",
        "name": "Alice",
        "age": 30,
    }
)

print(f"Valid user: {valid_user.name} ({valid_user.email})")
