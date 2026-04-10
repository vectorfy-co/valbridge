from typing import Annotated, Generic, Literal, TypeVar

from pydantic import BaseModel, ConfigDict, Field, RootModel, StringConstraints, UUID4
from pydantic import AliasChoices, AliasPath, field_serializer, field_validator, model_validator

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


class AliasDriven(BaseModel):
    observations: str = Field(
        validation_alias=AliasChoices(
            "observations",
            "reasoning",
            AliasPath("payload", "observations"),
        ),
        serialization_alias="observations",
    )


class ReviewEnvelope(RootModel[list[int]]):
    @model_validator(mode="after")
    def validate_payload(self):
        return self


T = TypeVar("T")


class EvidenceBox(BaseModel, Generic[T]):
    value: T


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


def test_extract_model_preserves_alias_choices_and_alias_paths():
    result = extract_model(AliasDriven)

    observations = result.schema["properties"]["observations"]
    assert observations["x-valbridge"]["aliasInfo"] == {
        "validationAlias": "observations",
        "serializationAlias": "observations",
    }
    assert observations["x-valbridge"]["registryMeta"]["validationAliasChoices"] == [
        "observations",
        "reasoning",
        ["payload", "observations"],
    ]


def test_extract_model_surfaces_root_and_model_validator_metadata():
    result = extract_model(ReviewEnvelope)

    assert result.schema["x-valbridge"]["registryMeta"] == {"rootModel": True}
    assert result.schema["x-valbridge"]["codeStubs"] == [
        {
            "kind": "modelValidator",
            "name": "validate_payload",
            "payload": {"mode": "after"},
        }
    ]


def test_extract_model_surfaces_generic_specialization_metadata():
    result = extract_model(EvidenceBox[int])

    assert result.schema["x-valbridge"]["registryMeta"] == {
        "genericOrigin": f"{EvidenceBox.__module__}.{EvidenceBox.__qualname__}",
        "genericArgs": ["builtins.int"],
    }
