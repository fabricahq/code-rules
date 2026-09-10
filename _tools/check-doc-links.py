"""Check local destinations and fragments in the built documentation."""
from html.parser import HTMLParser
from pathlib import Path
from urllib.parse import unquote, urlsplit

root = Path(__file__).resolve().parents[1] / "docs" / "dist"

class Links(HTMLParser):
    def __init__(self, source):
        super().__init__()
        self.ids = set()
        self.links = []
        self.feed(source)

    def handle_starttag(self, tag, attrs):
        attrs = dict(attrs)
        if "id" in attrs:
            self.ids.add(attrs["id"])
        if tag == "a" and "href" in attrs:
            self.links.append(attrs["href"])

pages = {path: Links(path.read_text()) for path in root.rglob("*.html")}
if not pages:
    raise SystemExit("No built pages found. Run bun run docs:build first.")
errors = []
for path, page in pages.items():
    for href in page.links:
        link = urlsplit(href)
        if link.scheme or link.netloc:
            continue
        dest = unquote(link.path)
        target = ((root / dest.lstrip("/")) if dest.startswith("/") else (path.parent / dest)) if dest else path
        if target.is_dir():
            target = target / "index.html"
        target = target.resolve()
        if not target.is_relative_to(root):
            errors.append(f"{path.relative_to(root)}: outside output: {href}")
        elif not target.exists():
            errors.append(f"{path.relative_to(root)}: missing destination: {href}")
        elif link.fragment and target in pages and unquote(link.fragment) not in pages[target].ids:
            errors.append(f"{path.relative_to(root)}: missing anchor: {href}")
if errors:
    raise SystemExit("\n".join(errors))
print(f"Checked local links and fragments in {len(pages)} pages.")
