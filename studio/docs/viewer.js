(function () {
  "use strict";

  const STORAGE_KEYS = {
    activeItem: "routing-slip-studio:docs-active-item",
    openSections: "routing-slip-studio:docs-open-sections",
  };

  const state = {
    openSections: new Set(),
    activeItem: null,
    docs: [],
    targets: null,
  };

  function init(options) {
    const docs = window.RoutingSlipDocs || [];
    const tree = document.querySelector(options.treeSelector);
    const timeline = document.querySelector(options.timelineSelector);
    const title = document.querySelector(options.titleSelector);
    const summary = document.querySelector(options.summarySelector);
    const eyebrow = document.querySelector(options.eyebrowSelector);
    const panelTitle = document.querySelector(options.panelTitleSelector);
    const panelMeta = document.querySelector(options.panelMetaSelector);
    if (!tree || !timeline || !docs.length) return;

    state.docs = docs;
    state.targets = { tree, timeline, title, summary, eyebrow, panelTitle, panelMeta };
    restoreDocsState(docs);
    renderTree(tree, docs, state.targets);
    const activeItem = findDocItem(docs, state.activeItem);
    if (activeItem) {
      renderDoc(activeItem, state.targets);
    } else if (isDocsOnlyViewport()) {
      renderFirstDoc(docs, state.targets);
    }
    bindDocsOnlyViewport(docs, state.targets);
    window.addEventListener("routing-slip-theme-change", () => {
      const item = findDocItem(state.docs, state.activeItem);
      if (item && state.targets) renderDoc(item, state.targets);
    });
  }

  function renderTree(tree, docs, targets) {
    tree.innerHTML = docs.map((section, sectionIndex) => {
      const open = state.openSections.has(section.title);
      const items = open ? section.items.map((item) => `
        <button class="docs-item ${state.activeItem === item.id ? "active" : ""}" type="button" data-doc-id="${escapeHtml(item.id)}">
          <span class="docs-title">${escapeHtml(item.title)}</span>
        </button>
      `).join("") : "";
      return `
        <section class="docs-section ${open ? "open" : ""}">
          <button class="docs-section-head" type="button" data-section-index="${sectionIndex}">
            <span>${open ? "▾" : "▸"}</span>
            <span class="docs-title">${escapeHtml(section.title)}</span>
          </button>
          <div>${items}</div>
        </section>
      `;
    }).join("");

    tree.querySelectorAll("[data-section-index]").forEach((button) => {
      button.addEventListener("click", () => {
        const section = docs[Number(button.dataset.sectionIndex)];
        if (!section) return;
        if (state.openSections.has(section.title)) state.openSections.delete(section.title);
        else state.openSections.add(section.title);
        persistOpenSections();
        renderTree(tree, docs, targets);
      });
    });

    tree.querySelectorAll("[data-doc-id]").forEach((button) => {
      button.addEventListener("click", async () => {
        const item = findDocItem(docs, button.dataset.docId);
        if (!item) return;
        state.activeItem = item.id;
        persistActiveItem();
        renderTree(tree, docs, targets);
        await renderDoc(item, targets);
        closeMobileDocs();
      });
    });
  }

  function findDocItem(docs, id) {
    for (const section of docs) {
      const item = section.items.find((entry) => entry.id === id);
      if (item) return item;
    }
    return null;
  }

  async function renderFirstDoc(docs, targets) {
    const item = docs.flatMap((section) => section.items || [])[0];
    if (!item) return;
    state.activeItem = item.id;
    persistActiveItem();
    renderTree(targets.tree, docs, targets);
    await renderDoc(item, targets);
  }

  function isDocsOnlyViewport() {
    return window.matchMedia?.("(max-width: 760px)")?.matches;
  }

  function bindDocsOnlyViewport(docs, targets) {
    const query = window.matchMedia?.("(max-width: 760px)");
    if (!query?.addEventListener) return;
    query.addEventListener("change", (event) => {
      if (event.matches && !state.activeItem) renderFirstDoc(docs, targets);
    });
  }

  function restoreDocsState(docs) {
    const storedSections = parseStoredSections();
    state.openSections = storedSections.size > 0 ? storedSections : new Set();

    const storedItem = localStorage.getItem(STORAGE_KEYS.activeItem);
    const activeItem = findDocItem(docs, storedItem);
    state.activeItem = activeItem ? activeItem.id : null;

    if (activeItem) {
      const section = docs.find((entry) => (entry.items || []).some((item) => item.id === activeItem.id));
      if (section) state.openSections.add(section.title);
    }
    if (state.openSections.size === 0 && docs[0]) state.openSections.add(docs[0].title);
  }

  function parseStoredSections() {
    try {
      const sections = JSON.parse(localStorage.getItem(STORAGE_KEYS.openSections) || "[]");
      return new Set(Array.isArray(sections) ? sections : []);
    } catch {
      return new Set();
    }
  }

  function persistActiveItem() {
    if (state.activeItem) localStorage.setItem(STORAGE_KEYS.activeItem, state.activeItem);
  }

  function persistOpenSections() {
    localStorage.setItem(STORAGE_KEYS.openSections, JSON.stringify([...state.openSections]));
  }

  async function renderDoc(item, targets) {
    targets.timeline.classList.add("timeline--docs");
    let content = "";
    try {
      content = await loadDocContent(item);
    } catch (error) {
      content = `# Documentacao indisponivel\n\nNao foi possivel carregar o arquivo \`${item.file || item.id}\`.\n\n${error.message}`;
    }
    targets.timeline.innerHTML = `<article class="doc-view">${markdownToHtml(content)}</article>`;
    renderMermaid(targets.timeline);
    if (targets.eyebrow) targets.eyebrow.textContent = "Documentacao";
    if (targets.title) targets.title.textContent = item.title;
    if (targets.summary) targets.summary.textContent = "Documentacao";
    if (targets.panelTitle) targets.panelTitle.textContent = "Documentacao";
    if (targets.panelMeta) targets.panelMeta.textContent = "Markdown";
    targets.timeline.scrollTop = 0;
  }

  async function loadDocContent(item) {
    if (item.content) return item.content;
    if (!item.file) return "";
    if (item.cachedContent) return item.cachedContent;
    const response = await fetch(resolveDocUrl(item.file), { cache: "no-cache" });
    if (!response.ok) throw new Error(`Nao foi possivel carregar ${item.file}`);
    item.cachedContent = await response.text();
    return item.cachedContent;
  }

  function resolveDocUrl(file) {
    return new URL(file, new URL("docs/", window.location.href)).href;
  }

  function closeMobileDocs() {
    if (isDocsOnlyViewport()) document.body.classList.remove("mobile-docs-open");
  }

  function markdownToHtml(markdown) {
    if (window.marked?.parse) {
      return sanitizeHtml(window.marked.parse(String(markdown || "")));
    }
    const lines = String(markdown || "").trim().split("\n");
    const html = [];
    let inCode = false;
    let code = [];
    let list = null;

    function closeList() {
      if (!list) return;
      html.push(`</${list}>`);
      list = null;
    }

    function closeCode() {
      if (!inCode) return;
      html.push(`<pre><code>${escapeHtml(code.join("\n"))}</code></pre>`);
      code = [];
      inCode = false;
    }

    for (const line of lines) {
      if (line.startsWith("```")) {
        if (inCode) closeCode();
        else {
          closeList();
          inCode = true;
          code = [];
        }
        continue;
      }
      if (inCode) {
        code.push(line);
        continue;
      }
      if (/^# /.test(line)) {
        closeList();
        html.push(`<h1>${inlineMarkdown(line.replace(/^# /, ""))}</h1>`);
        continue;
      }
      if (/^## /.test(line)) {
        closeList();
        html.push(`<h2>${inlineMarkdown(line.replace(/^## /, ""))}</h2>`);
        continue;
      }
      if (/^- /.test(line)) {
        if (list !== "ul") {
          closeList();
          list = "ul";
          html.push("<ul>");
        }
        html.push(`<li>${inlineMarkdown(line.replace(/^- /, ""))}</li>`);
        continue;
      }
      if (/^\d+\. /.test(line)) {
        if (list !== "ol") {
          closeList();
          list = "ol";
          html.push("<ol>");
        }
        html.push(`<li>${inlineMarkdown(line.replace(/^\d+\. /, ""))}</li>`);
        continue;
      }
      if (line.trim() === "") {
        closeList();
        continue;
      }
      closeList();
      html.push(`<p>${inlineMarkdown(line)}</p>`);
    }

    closeCode();
    closeList();
    return html.join("");
  }

  function inlineMarkdown(value) {
    return escapeHtml(value)
      .replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>")
      .replace(/\*([^*]+)\*/g, "<em>$1</em>")
      .replace(/`([^`]+)`/g, "<code>$1</code>");
  }

  function sanitizeHtml(html) {
    return String(html)
      .replace(/<script[\s\S]*?>[\s\S]*?<\/script>/gi, "")
      .replace(/\son\w+="[^"]*"/gi, "")
      .replace(/\son\w+='[^']*'/gi, "");
  }

  function renderMermaid(root) {
    const blocks = root.querySelectorAll("pre code.language-mermaid");
    blocks.forEach((block) => {
      const wrapper = document.createElement("div");
      wrapper.className = "mermaid";
      wrapper.textContent = block.textContent;
      block.closest("pre").replaceWith(wrapper);
    });
    if (window.mermaid && blocks.length) {
      try {
        window.mermaid.initialize(mermaidOptions());
        window.mermaid.run({ nodes: root.querySelectorAll(".mermaid") });
      } catch (error) {
        console.warn("Nao foi possivel renderizar Mermaid:", error);
      }
    }
  }

  function mermaidOptions() {
    const dark = document.body.dataset.theme === "dark";
    return {
      startOnLoad: false,
      securityLevel: "strict",
      theme: dark ? "base" : "default",
      themeVariables: dark ? {
        background: "#111827",
        mainBkg: "#1f2937",
        secondBkg: "#0f172a",
        primaryColor: "#1f2937",
        primaryTextColor: "#f8fafc",
        primaryBorderColor: "#2dd4bf",
        secondaryColor: "#0f172a",
        secondaryTextColor: "#e5e7eb",
        secondaryBorderColor: "#5eead4",
        tertiaryColor: "#111827",
        tertiaryTextColor: "#f8fafc",
        tertiaryBorderColor: "#334155",
        nodeTextColor: "#f8fafc",
        lineColor: "#94a3b8",
        textColor: "#f8fafc",
        edgeLabelBackground: "#111827",
        clusterBkg: "#0f172a",
        clusterBorder: "#334155",
        defaultLinkColor: "#94a3b8",
        titleColor: "#f8fafc"
      } : {}
    };
  }

  function escapeHtml(value) {
    return String(value)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;");
  }

  window.RoutingSlipDocsViewer = { init };
})();
