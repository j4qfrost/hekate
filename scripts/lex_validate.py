#!/usr/bin/env python3
"""Strict structural validator for app.hekate.* lexicon JSON files.

Sits between the previous one-line sanity check and full upstream-meta-schema
validation. Goes well beyond "has the right top-level keys": checks NSID
shape, def-types, record requireds resolve to known properties, property
types are real ATProto lexicon types, format-tagged strings declare an
appropriate format, and (heuristically) that AT-URI cross-references point
at app.hekate.* namespaces.

When the AT Protocol Lexicon SDK redesign (Buf-inspired, see ADR 0001)
lands an authoritative meta-schema, replace this with that. Until then,
this is the project's frozen contract.
"""

from __future__ import annotations

import json
import re
import sys
from pathlib import Path

LEXICON_VERSION = 1

# https://atproto.com/specs/lexicon — primitive + container types recognised
# at v0.1. Object is implied (used by `record.type` and `params`).
PROP_TYPES = {
    "boolean", "integer", "string", "bytes", "cid-link",
    "blob", "array", "object", "ref", "union", "unknown",
    "null",
}

# Top-level def types we expect at v0.1. Most hekate lexicons are records;
# the validator accepts the others so future xrpc additions don't break it.
DEF_TYPES = {"record", "query", "procedure", "subscription", "token", "object"}

# Formats recognised by `string.format`. Values from ATProto specs.
STRING_FORMATS = {
    "at-uri", "at-identifier", "did", "handle", "nsid", "tid",
    "datetime", "uri", "language", "cid",
}

NSID = re.compile(r"^([a-z][a-z0-9-]*\.)+[a-zA-Z][a-zA-Z0-9-]*$")


class LexErr(Exception):
    pass


def vassert(cond: bool, msg: str) -> None:
    if not cond:
        raise LexErr(msg)


def validate_property(path: str, name: str, prop: dict) -> None:
    vassert(isinstance(prop, dict), f"{path}.{name}: property must be object")
    t = prop.get("type")
    vassert(t in PROP_TYPES, f"{path}.{name}: unknown property type {t!r} (allowed: {sorted(PROP_TYPES)})")

    if t == "string":
        fmt = prop.get("format")
        if fmt is not None:
            vassert(fmt in STRING_FORMATS, f"{path}.{name}: unknown string format {fmt!r}")
        ml = prop.get("maxLength")
        if ml is not None:
            vassert(isinstance(ml, int) and ml > 0, f"{path}.{name}: maxLength must be positive int")
        enum = prop.get("enum")
        if enum is not None:
            vassert(isinstance(enum, list) and all(isinstance(e, str) for e in enum),
                    f"{path}.{name}: enum must be list of strings")
            default = prop.get("default")
            if default is not None:
                vassert(default in enum, f"{path}.{name}: default {default!r} not in enum {enum}")
    elif t == "array":
        items = prop.get("items")
        vassert(isinstance(items, dict), f"{path}.{name}: array missing items schema")
        validate_property(f"{path}.{name}.items", "_item", items)
    elif t == "integer":
        for k in ("minimum", "maximum"):
            v = prop.get(k)
            if v is not None:
                vassert(isinstance(v, int), f"{path}.{name}.{k}: must be int")
    elif t == "ref":
        vassert("ref" in prop, f"{path}.{name}: ref-type missing `ref`")
    elif t == "union":
        refs = prop.get("refs")
        vassert(isinstance(refs, list) and refs, f"{path}.{name}: union must have non-empty `refs`")


def validate_record(path: str, rec: dict) -> None:
    vassert(rec.get("type") == "object", f"{path}.record.type must be 'object'")
    props = rec.get("properties")
    vassert(isinstance(props, dict) and props, f"{path}.record.properties must be a non-empty object")
    required = rec.get("required", [])
    vassert(isinstance(required, list), f"{path}.record.required must be a list")
    missing = [r for r in required if r not in props]
    vassert(not missing, f"{path}.record.required references undefined properties: {missing}")
    vassert("createdAt" in props, f"{path}.record: every hekate record MUST define a createdAt property")
    vassert("createdAt" in required, f"{path}.record: createdAt MUST be in required")
    for name, prop in props.items():
        validate_property(path, name, prop)


def validate_def(path: str, name: str, d: dict) -> None:
    t = d.get("type")
    vassert(t in DEF_TYPES, f"{path}.{name}.type {t!r} not in {sorted(DEF_TYPES)}")
    if t == "record":
        vassert(d.get("key") in ("tid", "literal", "any", "nsid"), f"{path}.{name}.key must be one of tid/literal/any/nsid")
        rec = d.get("record")
        vassert(isinstance(rec, dict), f"{path}.{name}.record missing or wrong type")
        validate_record(f"{path}.{name}", rec)


def validate_file(p: Path) -> None:
    raw = json.loads(p.read_text())
    vassert(raw.get("lexicon") == LEXICON_VERSION, f"{p}: lexicon must be {LEXICON_VERSION}")

    nsid = raw.get("id")
    vassert(isinstance(nsid, str), f"{p}: missing id")
    vassert(NSID.match(nsid), f"{p}: id {nsid!r} is not a valid NSID")
    expected_filename = nsid.split(".")[-1] + ".json"
    vassert(p.name == expected_filename, f"{p}: filename should be {expected_filename} (matches id last segment)")

    vassert(nsid.startswith("app.hekate."), f"{p}: id {nsid} must live under the app.hekate.* namespace")

    defs = raw.get("defs")
    vassert(isinstance(defs, dict) and defs, f"{p}: defs must be a non-empty object")
    vassert("main" in defs, f"{p}: defs.main is required")
    for name, d in defs.items():
        validate_def(str(p), name, d)


def main(argv: list[str]) -> int:
    if not argv:
        print("usage: lex_validate.py <file.json> [...]", file=sys.stderr)
        return 2
    failed = 0
    for arg in argv:
        p = Path(arg)
        try:
            validate_file(p)
        except (LexErr, json.JSONDecodeError, OSError) as e:
            print(f"FAIL {p}: {e}", file=sys.stderr)
            failed += 1
        else:
            print(f"OK   {p}")
    if failed:
        print(f"\n{failed} lexicon(s) failed validation", file=sys.stderr)
        return 1
    print(f"\nAll {len(argv)} lexicon(s) OK")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
