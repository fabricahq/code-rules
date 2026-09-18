---
title: Make error messages actionable
impact: MEDIUM
impactDescription: helps users recover without guessing
tags: errors, user-experience
whenToRead: When writing or reviewing validation errors shown to users.
---

## Make error messages actionable

When rejecting user input, explain what is wrong
and how to correct it. State the actual constraint
so users know what to change.

### Example

For an upload with a 10 MB size limit:

**Incorrect:** "Upload failed."

**Correct:** "This file is too large.
Choose a file no larger than 10 MB."

### Validation

Try a file above the limit. Check that the message
states the configured limit and how to proceed.
