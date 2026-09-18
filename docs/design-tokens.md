# Design tokens

A token represents a design decision that should change consistently across its consumers.
Choose it by purpose, not by its current hex value. Equal values alone do not justify sharing a token.

## Shared site tokens

`src/styles/custom.css` owns light/dark values. `src/styles/tailwind.css` exposes reusable roles as utilities.

| Token / utility suffix | Purpose |
| --- | --- |
| `paper` | Page background and surfaces that match the page |
| `surface` | Subtle background for grouped content and hover states |
| `ink` | Primary foreground, including text and icons |
| `muted` | Secondary foreground, including captions and metadata |
| `border` | Normal component boundaries |
| `border-subtle` | Low-emphasis section separators |
| `border-strong` | Stronger boundaries, such as outlined actions |
| `primary-action` / `on-primary-action` | Filled primary action and its contrasting foreground; use together |
| `rounded-small` | Small-radius geometry for compact controls and badges (6px) |
| `rounded-card` | Content-card geometry (12px) |
| `rounded-panel` | Large-panel geometry (20px) |

Examples: `bg-paper`, `text-muted`, `border-border-subtle`, and `bg-primary-action text-on-primary-action`.
Radius names describe geometry or surface scale, not a mandatory component type. A badge and a compact button may share `rounded-small`; a pill-shaped action uses `rounded-full`.

The global fallback focus ring (`--cr-focus-ring`) and current-page marker (`--cr-current-page-indicator`) have independent roles even though their colors currently match. Component focus treatments may use a stronger foreground outline.
Starlight's `--sl-*` names are a framework API: retain those names at the integration boundary.
Use Tailwind's spacing and type scale for new component layout; avoid introducing a parallel global scale for one-off values.
The retained `--cr-text-sm`, `--cr-text-body`, and `--cr-line-height-body` values style generated documentation markup. `--cr-font-mono` styles examples; Starlight's inline-code font remains separate deliberately.
Responsive tokens and their Starlight alignment are documented in [the styling guide](README.md#styling).

## Component-private tokens

These variables are implementation details of their owning components. They are not exported in the global Tailwind color theme.

- **AgentPreview and its AgentSession children:** `--session-text`, `--session-text-muted`, `--session-border`, and `--session-surface*` describe the session window; `--session-surface-subtle` groups low-emphasis tool output and control states. `--session-action` marks interaction, `--session-user` identifies the task prompt, and `--session-agent` identifies the agent. `--session-success` / `--session-error` describe command results. `--session-diff-added` / `--session-diff-removed` describe edits independently of command success. Consume them only inside this subtree, for example `text-(--session-action)`.
- **AgentSetup:** `--setup-agent` identifies the agent and `--setup-rule-path` emphasizes the path in its instructions. Its palette is intentionally independent of the interactive session.
- **CodeSample:** `--sample-token-*` names follow Shiki's syntax roles. Each tone owns its complete palette, so the terminal tone works without an AgentPreview parent. Syntax colors must not borrow command-success, error, or button colors. Callers may set `--sample-font-size` and `--sample-padding` on their wrapper; syntax variables stay inside the renderer.

## Adding or reusing a token

1. Reuse an existing token when both consumers represent the same role and should change together in both themes.
2. Keep a component-specific decision local. Use a component prefix and a role name, such as `--setup-rule-path`, rather than a hue such as `--setup-blue`.
3. Promote a local role only when multiple independent components need the same design contract. Define both theme values, document its consumers, then expose a Tailwind alias.
4. Keep equal-valued roles separate when they need to evolve independently. A green string literal, passing test, and added diff line are three different meanings.
5. Remove unused tokens rather than leaving speculative APIs. Do not add a global token for every literal spacing, color mix, or illustration detail.
6. Verify changes in both themes and at the shared breakpoints. A naming-only refactor should preserve computed styles and layout.

