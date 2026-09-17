# Interactive authoring

PR37 collects missing authoring inputs on a terminal before opening a writer lock.
Explicit flags are preserved. Both stdin and stderr must be terminals;
`--non-interactive` always refuses missing inputs. Prompts and input echo belong
to the terminal on stderr; successful JSON remains on stdout.

Source creation asks for the repository, ref or HashiCorp version constraint,
and comma-separated groups. Rule creation offers to create missing group metadata.
Declining, blank answers, or EOF leave files unchanged and return usage status 2.
Ctrl+C cancels through the existing command context and returns status 1.
All actual writes revalidate state under the existing authoring writer lock.

The reader uses Go's terminal editor, a 4096-byte input bound, and short cancellation polling.
It restores terminal attributes on every return and creates no blocked background reader.

The walkthrough runs the compiled CLI in a real pseudo-terminal. Editable answer
steps wait for a particular prompt; no shell is run. The PTY dependency belongs
to review/test code and is not part of the production CLI dependency graph.

Validation includes real subprocess prompts, source selection, missing-group
consent/refusal, blank answers, EOF, SIGINT, noninteractive mode, and unchanged
files after cancellation. Run `go test -race ./...`.
