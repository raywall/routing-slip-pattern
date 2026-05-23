(function () {
  "use strict";

  const state = {
    openSections: new Set(),
    activeItem: null,
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

    if (docs[0]) state.openSections.add(docs[0].title);
    renderTree(tree, docs, { timeline, title, summary, eyebrow, panelTitle, panelMeta });
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
        renderTree(tree, docs, targets);
      });
    });

    tree.querySelectorAll("[data-doc-id]").forEach((button) => {
      button.addEventListener("click", () => {
        const item = findDocItem(docs, button.dataset.docId);
        if (!item) return;
        state.activeItem = item.id;
        renderTree(tree, docs, targets);
        renderDoc(item, targets);
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

  function renderDoc(item, targets) {
    targets.timeline.classList.add("timeline--docs");
    targets.timeline.innerHTML = `<article class="doc-view">${markdownToHtml(item.content)}</article>`;
    renderMermaid(targets.timeline);
    if (targets.eyebrow) targets.eyebrow.textContent = "Documentacao";
    if (targets.title) targets.title.textContent = item.title;
    if (targets.summary) targets.summary.textContent = "Documentacao";
    if (targets.panelTitle) targets.panelTitle.textContent = "Documentacao";
    if (targets.panelMeta) targets.panelMeta.textContent = "Markdown";
    targets.timeline.scrollTop = 0;
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
        window.mermaid.initialize({ startOnLoad: false, securityLevel: "strict", theme: "default" });
        window.mermaid.run({ nodes: root.querySelectorAll(".mermaid") });
      } catch (error) {
        console.warn("Nao foi possivel renderizar Mermaid:", error);
      }
    }
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
