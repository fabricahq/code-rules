/** @fileoverview Browse fixture files and configuration before the lab sends them to Go. */

/** Enhance a request textarea without changing file bytes until the user edits them. */
function mountFixtureFiles(input, onEdit) {
  // Build UI nodes using text, including untrusted filenames and source aliases.
  function element(tag, className, text) {
    const node = document.createElement(tag);
    node.className = className;
    if (text !== undefined) node.textContent = text;
    return node;
  }

  const section = element("section", "fixture-browser");
  section.setAttribute("aria-label", "Input filesystem");
  const heading = element("div", "fixture-heading");
  heading.append(
    element("h3", "", "Input files"),
    element(
      "p",
      "hint",
      "Explore the filesystem before Go loads it. Select a file to read or edit its contents.",
    ),
  );
  const tree = element("nav", "fixture-tree");
  tree.setAttribute("aria-label", "Fixture files");
  const pane = element("div", "fixture-pane");
  const filename = element("label", "fixture-filename", "Select a file");
  filename.htmlFor = "fixture-content";
  const editor = element("textarea", "fixture-content");
  editor.id = "fixture-content";
  editor.spellcheck = false;
  const info = element("p", "hint fixture-info");
  pane.append(filename, editor, info);
  section.append(heading, tree, pane);

  const advanced = element("section", "fixture-advanced");
  advanced.id = "fixture-request";
  section.id = "fixture-files";
  advanced.append(
    element(
      "p",
      "hint",
      "Edit groups, sources, and other settings here. Add, rename, or remove files in the files objects.",
    ),
  );
  const configuration = element("section", "fixture-configuration");
  configuration.id = "fixture-configuration";
  configuration.append(
    element("h3", "", ".code-rules/config.json"),
    element(
      "p",
      "hint",
      "Project configuration for this scenario. Edit it in Request JSON and settings.",
    ),
  );
  const configText = element("pre", "");
  configuration.append(configText);
  const switcher = element("div", "fixture-views");
  switcher.setAttribute("role", "group");
  switcher.setAttribute("aria-label", "Input view");
  const filesView = element("button", "", "Files");
  const requestView = element("button", "", "Request JSON and settings");
  const configView = element("button", "", "config.json");
  filesView.type = requestView.type = configView.type = "button";
  filesView.setAttribute("aria-controls", section.id);
  requestView.setAttribute("aria-controls", advanced.id);
  configView.setAttribute("aria-controls", configuration.id);
  configView.hidden = true;
  switcher.append(filesView, configView, requestView);
  let currentView = "files";

  // Switch presentation only; all views share the same request.
  function showView(view) {
    currentView = view;
    section.hidden = view !== "files";
    advanced.hidden = view !== "request";
    configuration.hidden = view !== "config";
    filesView.setAttribute("aria-pressed", String(view === "files"));
    requestView.setAttribute("aria-pressed", String(view === "request"));
    configView.setAttribute("aria-pressed", String(view === "config"));
  }
  // Selecting a view never invokes Go or clears an existing result.
  filesView.addEventListener("click", () => showView("files"));
  requestView.addEventListener("click", () => showView("request"));
  configView.addEventListener("click", () => showView("config"));
  showView("files");

  const label = input.previousElementSibling;
  input.before(switcher, section, configuration, advanced);
  if (label?.matches('label[for="input"]')) advanced.append(label);
  advanced.append(input);
  const workspace = input.closest(".workspace");
  workspace.classList.add("has-fixture-browser");
  // Keep invocation next to the files and leave explanatory notes available on demand.
  const toolbar = element("div", "fixture-toolbar");
  switcher.before(toolbar);
  toolbar.append(input.closest(".panel").querySelector(".actions"), switcher);
  for (const details of workspace.querySelectorAll(".result > details")) {
    details.open = false;
  }

  let request;
  let entries = [];
  let selected;

  // Recognize JSON objects while leaving invalid fixtures for Go to validate.
  function isObject(value) {
    return value !== null && typeof value === "object" && !Array.isArray(value);
  }

  // Show stored text verbatim; selecting a file does not rewrite the request.
  function select(entry) {
    selected = entry;
    filename.textContent = `${entry.root} / ${entry.path}`;
    const content = entry.files[entry.path];
    editor.readOnly = typeof content !== "string";
    editor.value =
      typeof content === "string" ? content : JSON.stringify(content, null, 2);
    info.textContent = editor.readOnly
      ? "This file value is not text. Correct it in Request JSON and settings."
      : `${new TextEncoder().encode(content).length} bytes · Edits update the input fixture. Invoke Go to see the result.`;
    for (const file of entries)
      file.button.setAttribute("aria-pressed", String(file === entry));
  }

  // Render one root with nested folders; Maps keep special property names literal.
  function addRoot(root, files) {
    const group = element("div", "fixture-root");
    group.append(element("p", "fixture-root-name", root));
    tree.append(group);
    if (!isObject(files)) {
      group.append(
        element(
          "p",
          "hint",
          "Invalid file map. Edit Request JSON and settings.",
        ),
      );
      return;
    }
    const list = element("ul", "");
    group.append(list);
    const folders = new Map([["", list]]);
    for (const path of Object.keys(files).sort()) {
      const parts = path.split("/");
      let parent = list;
      let prefix = "";
      for (const part of parts.slice(0, -1)) {
        prefix += `${part}/`;
        if (!folders.has(prefix)) {
          const item = element("li", "");
          const folder = element("details", "fixture-folder");
          folder.open = true;
          const children = element("ul", "");
          folder.append(
            element("summary", "", `${part || "(empty)"}/`),
            children,
          );
          item.append(folder);
          parent.append(item);
          folders.set(prefix, children);
        }
        parent = folders.get(prefix);
      }
      const button = element(
        "button",
        "fixture-file",
        parts.at(-1) || "(empty filename)",
      );
      button.type = "button";
      button.setAttribute("aria-label", `${root} / ${path}`);
      button.title = path;
      const entry = { root, path, files, button };
      // Navigation leaves both fixture bytes and the last result untouched.
      button.addEventListener("click", () => select(entry));
      entries.push(entry);
      const item = element("li", "");
      item.append(button);
      parent.append(item);
    }
    if (!Object.keys(files).length)
      group.append(element("p", "hint", "No files in this root."));
  }

  // Rebuild from presets or raw JSON, removing stale files when input is invalid.
  function refresh() {
    configView.hidden = true;
    configText.textContent = "";
    const previous = selected;
    selected = undefined;
    entries = [];
    tree.replaceChildren();
    editor.value = "";
    editor.readOnly = true;
    filename.textContent = "Select a file";
    info.textContent = "";
    try {
      request = JSON.parse(input.value);
    } catch {
      tree.append(
        element(
          "p",
          "hint",
          "Invalid request JSON. Fix it in Request JSON and settings to view files.",
        ),
      );
      showView("request");
      return;
    }
    const fixture =
      isObject(request) && Object.hasOwn(request, "fixture")
        ? request.fixture
        : request;
    if (isObject(fixture)) {
      if (Object.hasOwn(fixture, "configuration")) {
        configView.hidden = false;
        configText.textContent = JSON.stringify(fixture.configuration, null, 2);
      }
      if (Object.hasOwn(fixture, "files"))
        addRoot(
          `Library: ${typeof fixture.source === "string" && fixture.source ? fixture.source : "library"}`,
          fixture.files,
        );
      if (isObject(fixture.libraries)) {
        for (const [alias, library] of Object.entries(fixture.libraries))
          addRoot(`Library: ${alias}`, library?.files);
      }
      if (Object.hasOwn(fixture, "localFiles"))
        addRoot("Local rules", fixture.localFiles);
    }
    if (!tree.childElementCount) {
      tree.append(
        element(
          "p",
          "hint",
          "No input file maps. Edit Request JSON and settings.",
        ),
      );
      showView("request");
    }
    if (currentView === "config" && configView.hidden) showView("request");
    // Prefer the previous file, then a rule document, then the first available file.
    const initial =
      entries.find(
        (entry) =>
          entry.root === previous?.root && entry.path === previous?.path,
      ) ||
      entries.find(
        (entry) =>
          entry.path.endsWith(".md") && !entry.path.includes("assets/"),
      ) ||
      entries[0];
    if (initial) select(initial);
  }

  // Commit only the selected file, preserving other files and request settings.
  editor.addEventListener("input", () => {
    if (!selected || editor.readOnly) return;
    selected.files[selected.path] = editor.value;
    input.value = JSON.stringify(request, null, 2);
    info.textContent = `${new TextEncoder().encode(editor.value).length} bytes · Edits update the input fixture. Invoke Go to see the result.`;
    onEdit();
  });
  input.addEventListener("input", refresh);
  refresh();
}
