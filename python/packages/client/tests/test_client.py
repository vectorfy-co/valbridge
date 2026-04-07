"""Tests for valbridge."""

import pytest
from valbridge import ValbridgeError, create_valbridge


class MockValidator:
    """Mock validator for testing."""

    def __init__(self, name: str):
        self.name = name

    def model_validate(self, data):
        return data


def test_full_key_lookup():
    """Test looking up schemas with full namespace:id keys."""
    schemas = {
        "user:Profile": MockValidator("Profile"),
        "another:TSConfig": MockValidator("TSConfig"),
    }
    valbridge = create_valbridge(schemas)

    # Should find by full key
    assert valbridge("user:Profile").name == "Profile"
    assert valbridge("another:TSConfig").name == "TSConfig"


def test_schema_not_found():
    """Test error when schema doesn't exist."""
    schemas = {
        "user:Profile": MockValidator("Profile"),
    }
    valbridge = create_valbridge(schemas)

    # Non-existent key
    with pytest.raises(ValbridgeError) as exc_info:
        valbridge("user:NonExistent")
    assert "Unknown schema: user:NonExistent" in str(exc_info.value)


def test_empty_schemas():
    """Test client with empty schemas dict."""
    valbridge = create_valbridge({})

    with pytest.raises(ValbridgeError):
        valbridge("anything")


def test_type_inference():
    """Test that client preserves type information."""
    schemas = {
        "user:Profile": MockValidator("Profile"),
    }
    valbridge = create_valbridge(schemas)

    # Should return the exact validator instance
    validator = valbridge("user:Profile")
    assert isinstance(validator, MockValidator)
    assert validator.name == "Profile"


def test_nonexistent_namespace():
    """Test error when namespace doesn't exist."""
    schemas = {
        "user:Profile": MockValidator("Profile"),
        "another:Config": MockValidator("Config"),
    }
    valbridge = create_valbridge(schemas)

    # Non-existent namespace should fail
    with pytest.raises(ValbridgeError) as exc_info:
        valbridge("nonexistent:Schema")
    assert "Unknown schema: nonexistent:Schema" in str(exc_info.value)


def test_client_is_callable():
    """Test that returned client is callable."""
    schemas = {"user:Profile": MockValidator("Profile")}
    valbridge = create_valbridge(schemas)

    # Should be callable
    assert callable(valbridge)

    # Should work like a function
    result = valbridge("user:Profile")
    assert result.name == "Profile"


def test_multiple_clients():
    """Test creating multiple independent clients."""
    schemas1 = {"user:Profile": MockValidator("Profile1")}
    schemas2 = {"user:Profile": MockValidator("Profile2")}

    valbridge1 = create_valbridge(schemas1)
    valbridge2 = create_valbridge(schemas2)

    # Should be independent
    assert valbridge1("user:Profile").name == "Profile1"
    assert valbridge2("user:Profile").name == "Profile2"
