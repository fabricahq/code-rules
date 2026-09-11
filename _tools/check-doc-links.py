"""Check local destinations and fragments in the built documentation."""
from html.parser import HTMLParser
from pathlib import Path
from urllib.parse import unquote, urlsplit


class Links(HTMLParser):
    """Collect element IDs and anchor destinations from one HTML page."""

    def __init__(self, source):
        """Parse page text into ID and link collections."""
        super().__init__()
        self.ids = set()
        self.links = []
        self.feed(source)

    def handle_starttag(self, tag, attrs):
        """Record IDs on any element and href values on anchors."""
        attrs = dict(attrs)
        if "id" in attrs:
            self.ids.add(attrs["id"])
        if tag == "a" and "href" in attrs:
            self.links.append(attrs["href"])


def local_target(root, source, url_path):
    """Resolve a decoded URL path and directory index relative to its page or site root."""
    destination = unquote(url_path)
    if not destination:
        target = source
    elif destination.startswith("/"):
        target = root / destination.lstrip("/")
    else:
        target = source.parent / destination
    if target.is_dir():
        target = target / "index.html"
    return target.resolve()


def check_link(root, source, href, pages):
    """Return the first local-link diagnostic, or None for a valid or external link."""
    link = urlsplit(href)
    if link.scheme or link.netloc:
        return None
    target = local_target(root, source, link.path)
    location = source.relative_to(root)
    if not target.is_relative_to(root):
        return f"{location}: outside output: {href}"
    if not target.exists():
        return f"{location}: missing destination: {href}"
    if link.fragment and target in pages and unquote(link.fragment) not in pages[target].ids:
        return f"{location}: missing anchor: {href}"
    return None


def check_site(root):
    """Scan built pages and fail with collected link diagnostics, or report the page count."""
    pages = {path: Links(path.read_text()) for path in root.rglob("*.html")}
    if not pages:
        raise SystemExit("No built pages found. Run bun run docs:build first.")
    errors = []
    for source, page in pages.items():
        for href in page.links:
            error = check_link(root, source, href, pages)
            if error is not None:
                errors.append(error)
    if errors:
        raise SystemExit("\n".join(errors))
    print(f"Checked local links and fragments in {len(pages)} pages.")


if __name__ == "__main__":
    check_site(Path(__file__).resolve().parents[1] / "docs" / "dist")
