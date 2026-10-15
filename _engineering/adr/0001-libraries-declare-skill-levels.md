# Libraries declare skill levels

Organizations need to make some skills mandatory for their projects and suggest others. A library declares each of its skills, including those it borrows from its own sources, as required, recommended, or offered, and any repo that takes from the library installs its required skills. The policy then lives with the content it governs, in the org's own library, and a project reasons about one declaration per source.

## Considered Options

- **Presets that projects extend:** an early draft of the design put the default selection in a separate preset file that a project named with `extends`. Rejected because it added a second kind of file and a second layering mechanism beside sources, which made it harder to tell which layer brought a skill in. Presets never shipped.
- **Levels that chain through borrowed libraries:** rejected. A project sees only the levels of the libraries it takes from directly, so an upstream library can't change what a downstream library asks of its projects.

## Consequences

- Taking anything from a library means taking what it requires. A project can't exclude or fork a required skill; to diverge, it stops using the source.
- A library repo installs the required skills of the sources it borrows, as any project would.
