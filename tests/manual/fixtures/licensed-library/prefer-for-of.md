---
title: Prefer for-of for array iteration
whenToRead: Before implementing, changing, or reviewing iteration over arrays or typed arrays using forEach.
impact: LOW
impactDescription: Callback-based iteration can obscure loop control and introduce an unnecessary function boundary.
tags: [iteration, control-flow]
attribution:
  - url: https://github.com/sindresorhus/eslint-plugin-unicorn/blob/5d9d745c5365b6fdb824db1122ff982dd824b11a/docs/rules/no-for-each.md
    description: Adapted from Sindre Sorhus's MIT-licensed ESLint Unicorn rule; added task guidance and a sparse-array exception.
---

Prefer a for-of loop for array and typed-array iteration when it preserves the intended behavior. This is an adapted engineering preference, not proof that every forEach call is a defect.

### Implementation

Use loop control directly when the operation needs to skip an iteration or stop early. Use entries() when the index is needed. Preserve null checks and callback behavior when converting existing code.

### Validation

Check the receiver type and behavior before recommending a conversion. This example does not target Map, Set, or custom forEach methods. Array forEach skips empty slots in sparse arrays, whereas for-of visits them; preserve that distinction when it matters.
