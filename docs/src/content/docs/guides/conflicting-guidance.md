---
title: "Conflicting guidance"
description: "Find contradictory rules and make the project's intended policy explicit."
---

Two rules can have distinct IDs and still give incompatible instructions for the same situation.
Namespacing identifies each rule; it does not reconcile what the rules say.

## Recognize a conflict

Consider two illustrative rules:

| Rule ID | Instruction |
| --- | --- |
| `fabrica:techs/typescript/prefer-type-aliases` | Use type aliases for object types. |
| `acme:techs/typescript/prefer-interfaces` | Use interfaces for object types. |

If both rules apply to the same object type, an agent cannot satisfy both.
An organization source does not automatically take precedence over another library.

Different wording alone is not a conflict.
One rule could require interfaces only for public extension points, while another requires type aliases only for internal types.
Those rules can coexist when their scopes are explicit and do not overlap.
Repeated guidance may be redundant without being contradictory.

## Generate a review prompt

