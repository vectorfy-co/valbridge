from typing import Annotated, Literal

from pydantic import BaseModel, ConfigDict, Field, StringConstraints, UUID4
from pydantic import field_serializer, field_validator

from valbridge_pydantic_extractor import extract_model


def make_slug() -> str:
    return "guest"


class Cat(BaseModel):
    model_config = ConfigDict(extra="forbid")

    kind: Literal["cat"]
    name: Annotated[
        str,
        StringConstraints(strip_whitespace=True, to_lower=True, min_length=1),
    ] = Field(alias="cat_name")


class Dog(BaseModel):
    kind: Literal["dog"]
    bark: int


class Example(BaseModel):
    model_config = ConfigDict(extra="forbid")

    id: UUID4
    count: int = 3
    display_name: Annotated[
        str,
        StringConstraints(strip_whitespace=True, min_length=1),
    ] = Field(
        validation_alias="displayName",
        serialization_alias="displayName",
        validate_default=True,
        json_schema_extra={"x-ui": "display"},
    )
    slug: str = Field(default_factory=make_slug)
    chooser: Annotated[int | str, Field(union_mode="left_to_right")]
    pet: Annotated[Cat | Dog, Field(discriminator="kind")]

    @field_validator("display_name")
    @classmethod
    def validate_display_name(cls, value: str) -> str:
        return value

    @field_serializer("display_name")
    def serialize_display_name(self, value: str) -> str:
        return value


def test_extract_model_preserves_enriched_pydantic_semantics():
    result = extract_model(Example)

    assert result.diagnostics == ()
    assert result.schema["x-valbridge"]["extraMode"] == "forbid"

    count = result.schema["properties"]["count"]
    assert count["x-valbridge"]["defaultBehavior"] == {
        "kind": "default",
        "value": 3,
    }

    display_name = result.schema["properties"]["display_name"]
    assert display_name["x-valbridge"]["aliasInfo"] == {
        "validationAlias": "displayName",
        "serializationAlias": "displayName",
    }
    assert display_name["x-valbridge"]["transforms"] == [{"kind": "trim"}]
    assert display_name["x-valbridge"]["registryMeta"]["validateDefault"] is True
    assert display_name["x-valbridge"]["codeStubs"] == [
        {"kind": "validator", "name": "validate_display_name", "payload": {"mode": "after"}},
        {"kind": "serializer", "name": "serialize_display_name", "payload": {"mode": "plain"}},
    ]

    slug = result.schema["properties"]["slug"]
    assert slug["x-valbridge"]["defaultBehavior"] == {
        "kind": "factory",
        "factory": f"{make_slug.__module__}.{make_slug.__qualname__}",
    }

    chooser = result.schema["properties"]["chooser"]
    assert chooser["x-valbridge"]["resolution"] == "leftToRight"

    pet = result.schema["properties"]["pet"]
    assert pet["x-valbridge"]["discriminator"] == "kind"

    identifier = result.schema["properties"]["id"]
    assert identifier["x-valbridge"]["formatDetail"] == {
        "kind": "uuid",
        "version": "v4",
    }

    cat_def = result.schema["$defs"]["Cat"]
    assert cat_def["x-valbridge"]["extraMode"] == "forbid"
