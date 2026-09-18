---
title: Verify retry limits
impact: HIGH
impactDescription: prevents unbounded requests
tags: testing, retries
whenToRead: When planning, implementing, reviewing, or diagnosing bounded retry behavior.
---

## Verify retry limits

When adding or changing bounded retries,
test that requests stop at the limit.

### Example

For a limit of three attempts:
- Make every attempt fail.
- Assert exactly three calls.
- Assert the documented failure is returned.

### Validation

Remove the stop condition temporarily.
The test should fail.
