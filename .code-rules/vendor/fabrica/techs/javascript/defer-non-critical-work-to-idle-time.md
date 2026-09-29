---
title: "Defer non-critical browser work to idle time"
whenToRead: "Before planning, writing, changing, or reviewing browser code that does secondary work in response to user input or page load, such as analytics, persisting drafts, prefetching, or processing large data."
impact: "MEDIUM"
impactDescription: "Secondary work done immediately after user input competes with rendering the response, making the interface feel slow."
tags: "javascript, browser, scheduling, requestIdleCallback, performance"
attribution:
  - url: https://github.com/vercel-labs/agent-skills/blob/4ec6f84b61cd3c931046c3e6e398f3ae7de372f7/skills/react-best-practices/rules/js-request-idle-callback.md
    description: "Adapted from the Vercel Agent Skills rule js-request-idle-callback: moved from the React group, restructured to the rule template, and generalized the browser support guidance."
---

## Defer non-critical browser work to idle time

Schedule work that the user is not waiting for with `requestIdleCallback`, so it runs when the browser is idle instead of delaying the response to input.

### Implementation

- Defer analytics, saving non-urgent state to storage, prefetching likely next resources, and non-urgent data processing.
- Pass a `timeout` when the work must eventually run even if the browser stays busy, such as analytics.
- Split large jobs into chunks that check `deadline.timeRemaining()` and reschedule themselves.
- `requestIdleCallback` is not available in every browser; check support for the project's target browsers and fall back to `setTimeout`.
- Do not defer work the user is waiting for, such as rendering the result of their action.
- Idle callbacks may never run if the user leaves the page; flush work that must not be lost, such as saving a draft, when the page is hidden.

### Rationale

JavaScript runs on the same thread that handles input and rendering.
Doing secondary work right after an interaction delays the frame that shows the result.
Idle callbacks run in gaps when the browser has nothing more urgent to do.

### Examples

**Incorrect (counterexample):**

```ts
function handleSearch(query: string) {
  setResults(searchItems(query));
  analytics.track('search', { query });
  saveToRecentSearches(query);
}
```

Tracking and saving run before the browser can paint the results.

**Correct:**

```ts
const scheduleIdle: (callback: () => void) => void =
  typeof requestIdleCallback === 'function'
    ? (callback) => requestIdleCallback(callback, { timeout: 2000 })
    : (callback) => setTimeout(callback, 1);

function handleSearch(query: string) {
  setResults(searchItems(query));
  scheduleIdle(() => {
    analytics.track('search', { query });
    saveToRecentSearches(query);
  });
}
```

### Validation

Record a performance profile of the interaction and check that deferred work runs after the result is painted.
Check that a fallback exists for browsers without `requestIdleCallback`.

Running work immediately is not a violation when the user is waiting for its result.
