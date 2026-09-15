# Git repository source parsing

Researched 2026-09-14. Scope: Git repository addresses, duplicate detection, and source links. Imports now fetches repositories through Git; sync orchestration belongs to a dependent PR. No upstream implementation code was copied.

## Recommendation

Use native WHATWG `URL` with a small explicit scp-style parser and a strict repository contract. Accept HTTPS, `ssh://`, and `user@host:path`. Use explicit addresses in configuration and examples. Since the CLI format has not shipped, remove `owner/name` shorthand instead of retaining an implicit GitHub default. Keep the original locator for provenance and snapshot matching. Treat normalized identity and browser links as separate outputs.

This is a design recommendation, not a claim that these forms cover every valid Git repository. Git also supports local paths and other protocols; its manual warns that `git://` lacks authentication and discourages FTP. Arbitrary nested repository paths are valid, so a universal `owner/name` shape is inappropriate. [Git clone transport documentation](https://git-scm.com/docs/git-clone#_git_urls)

## Why not go-getter or a provider parser?

- OpenTofu allows generic Git repositories with `git::`, runs `git clone`, supports a separate `ref` argument, and defines special `//subdirectory` syntax. This demonstrates host independence, but its source grammar is deliberately broader than a repository identity field. Keep revision and selected paths in existing separate manifest fields instead of adopting getter query semantics. [OpenTofu module sources](https://opentofu.org/docs/v1.9/language/modules/sources/)
- Go-getter is a fetching framework supporting local files, Git, Mercurial, HTTP, S3 and GCS, with detection, redirects, archive extraction and query options. Its maintainers explicitly identify SSRF, traversal and denial-of-service risks for user-supplied URLs. It is unnecessary for an offline TypeScript metadata parser and would expand future fetching scope substantially. [Go-getter README and security guidance](https://github.com/hashicorp/go-getter)
- `hosted-git-info` generates convenient browser/raw URLs, but its documented host set is GitHub/Gists, Bitbucket, GitLab and Sourcehut. That does not provide generic Git host independence. [hosted-git-info README](https://github.com/npm/hosted-git-info)
- `git-url-parse` interprets provider URLs, including file URLs, refs, credentials and transport conversions. Its API accepts a list of known refs to disambiguate branches containing slashes. These conveniences do not define the narrow manifest contract, and adding it would still require separate validation. [git-url-parse README](https://github.com/IonicaBizau/git-url-parse)

## Validation and normalization

The WHATWG parser is a good component parser, not a strict input validator. The standard explicitly allows parsing to continue after validation errors. Reject raw control characters, whitespace, backslashes, missing authority, unsupported schemes, queries, fragments, malformed percent escapes, and literal or encoded dot segments before parsing can repair them. Reject HTTPS user information and SSH passwords. Permit SSH usernames and explicit ports under a documented grammar. A deliberately narrow path alphabet is reasonable if its limits are documented; otherwise validate encoded segments and reject encoded separators. [WHATWG URL Standard](https://url.spec.whatwg.org/#writing)

Normalize DNS host case, preserve arbitrary path and username case, and preserve nondefault ports. RFC 3986 treats scheme and host as case-insensitive; other components are case-sensitive unless their scheme says otherwise. Do not apply GitHub-specific path case folding to other hosts. [RFC 3986 section 6.2.2.1](https://www.rfc-editor.org/rfc/rfc3986#section-6.2.2.1)

Node's WHATWG implementation exposes components and handles HTTPS host normalization, but non-special schemes behave differently. A local Node probe on the research date confirmed:

| Input | Parsed result |
| --- | --- |
| `https://Example.COM/A/B.git` | Host becomes `example.com`; path case stays intact |
| `ssh://git@Example.COM/A/B.git` | Host remains `Example.COM` |
| `https://example.com/a/../b` | Path becomes `/b` |
| `https://example.com/a/%2e%2e/b` | Path becomes `/b` |
| HTTPS path containing a backslash | Backslash becomes slash |
| HTTPS host containing a newline | Newline disappears |
| `https://example.com/a/%ZZ` | Invalid escape remains accepted |

Explicitly normalize SSH hostnames; do not rely on constructor success as proof of valid manifest input. [Node URL documentation](https://nodejs.org/api/url.html#new-urlinput-base)

Preserve scp versus URL form for generic repositories. In particular, `user@host:repo.git` and `ssh://user@host/repo.git` must not be assumed equivalent: relative scp paths and absolute URL paths have different semantics. Git's parser distinguishes colon-separated scp form from slash-separated URL form. Likewise, do not generally remove `.git` or equate HTTPS and SSH addresses without provider knowledge. [Git transport parser](https://github.com/git/git/blob/master/connect.c)

## Browser and raw links

For explicitly recognized public provider endpoints, provider-specific rendering is appropriate:

- GitHub browser files use `/blob/<ref>/<path>`. A commit ID gives a stable permalink; branches can change. [GitHub permanent file links](https://docs.github.com/en/repositories/working-with-files/using-files/getting-permanent-links-to-files)
- GitLab browser files use `/-/blob/<ref>/<path>`, as demonstrated by its own repository. Its raw-file API accepts separately encoded project and file paths and a ref. [GitLab repository example](https://gitlab.com/gitlab-org/gitlab-foss/-/blob/v17.7.4/doc/api/repository_files.md), [GitLab raw-file API](https://docs.gitlab.com/api/repository_files/#retrieve-a-raw-file-from-a-repository)

Recommendation: match exact known hosts, default provider ports and supported SSH users before generating provider URLs. A hostname containing `github` or `gitlab` does not establish its software. An arbitrary SSH host/port does not establish any HTTPS browser address. Preserve nested namespace paths. Encode ref and file segments intentionally and prefer resolved commit IDs when future imports record them.

For unknown hosts, use a vendored source link and show repository, ref and path as provenance. Do not invent `/blob` or raw URLs. When a relative upstream document or asset is not vendored, report the missing snapshot content and require preserving it or supplying an explicit source URL. This is a product recommendation based on the distinction between Git transport addresses and provider browser routes.

## Importer boundary

Parser acceptance does not prove that a remote exists, is reachable, or is safe to fetch. The research identified these constraints:

1. Invoke Git with an argument array, never a shell-interpolated command. Supply a controlled destination and explicit option terminators where supported.
2. Restrict Git transport policy to HTTPS and SSH, with other protocols disabled. Apply restrictions to actual Git execution: `url.*.insteadOf` can rewrite an apparently allowed URL, and Git applies protocol policies after rewriting. Decide explicitly whether local Git/SSH configuration is trusted. [Git protocol and URL rewrite settings](https://git-scm.com/docs/git-config)
3. Avoid recursive submodules and working-tree execution; use a controlled temporary checkout, bounded resources and timeouts, and verify every consumed file stays inside the snapshot root. Disable hooks and account for configured checkout filters before materializing untrusted trees. Git exposes hook disabling and smudge-filter commands in configuration. [Git configuration documentation](https://git-scm.com/docs/git-config)
4. If fetching runs as a service, add a network egress policy and redirect/DNS handling appropriate to its trust boundary. HTTPS and SSH alone do not prevent SSRF. [Go-getter security guidance](https://github.com/hashicorp/go-getter#security)

The Imports implementation now uses the configured address directly, separate Git arguments, bare repositories, disabled hooks and submodules, and bounded execution. HTTPS and SSH are allowed by default. The caller's local Git and SSH configuration is trusted: another rewritten protocol requires an explicit protocol-specific allowance; `ext` remains disabled. Service-level egress policy remains outside this local tool.
