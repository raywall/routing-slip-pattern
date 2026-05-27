function activateTab(tab) {
  els.sideTabs.classList.remove("collapsed");
  document.querySelectorAll("[data-tab]").forEach((button) => {
    button.classList.toggle("active", button.dataset.tab === tab);
  });
  document.querySelectorAll("[data-panel]").forEach((panel) => {
    panel.classList.toggle("active", panel.dataset.panel === tab);
  });
  localStorage.setItem("routing-slip-studio:active-tab", tab);
  localStorage.setItem("routing-slip-studio:tabs-collapsed", "0");
  updateTabsToggle();
}

function toggleTabsCollapsed() {
  const collapsed = !els.sideTabs.classList.contains("collapsed");
  els.sideTabs.classList.toggle("collapsed", collapsed);
  localStorage.setItem("routing-slip-studio:tabs-collapsed", collapsed ? "1" : "0");
  updateTabsToggle();
}

function restorePanelState() {
  activateTab(localStorage.getItem("routing-slip-studio:active-tab") || "workspace");
  const tabHeight = localStorage.getItem("routing-slip-studio:tabs-height");
  if (tabHeight) els.sideTabs.style.height = `${tabHeight}px`;
  if (localStorage.getItem("routing-slip-studio:tabs-collapsed") === "1") {
    els.sideTabs.classList.add("collapsed");
  }
  updateTabsToggle();
  const width = localStorage.getItem("routing-slip-studio:sidebar-width");
  if (width) els.sidebar.style.width = `${width}px`;
}

function updateTabsToggle() {
  const collapsed = els.sideTabs.classList.contains("collapsed");
  const label = collapsed ? "Maximizar painel" : "Minimizar painel";
  const icon = collapsed ? "square" : "minus";
  els.collapseTabs.title = label;
  els.collapseTabs.setAttribute("aria-label", label);
  els.collapseTabs.innerHTML = `<i data-lucide="${icon}"></i>`;
  renderIcons();
}

function bindSidebarResize() {
  restorePanelState();
  bindTabsResize();
  let startX = 0;
  let startWidth = 0;

  els.resizer.addEventListener("mousedown", (event) => {
    startX = event.clientX;
    startWidth = els.sidebar.getBoundingClientRect().width;
    document.body.classList.add("resizing");
    document.addEventListener("mousemove", onMouseMove);
    document.addEventListener("mouseup", onMouseUp);
  });

  function onMouseMove(event) {
    const max = Math.max(420, window.innerWidth * 0.72);
    const width = Math.min(max, Math.max(340, startWidth + event.clientX - startX));
    els.sidebar.style.width = `${width}px`;
  }

  function onMouseUp() {
    document.body.classList.remove("resizing");
    localStorage.setItem("routing-slip-studio:sidebar-width", String(Math.round(els.sidebar.getBoundingClientRect().width)));
    document.removeEventListener("mousemove", onMouseMove);
    document.removeEventListener("mouseup", onMouseUp);
  }
}

function bindTabsResize() {
  let startY = 0;
  let startHeight = 0;

  els.tabsResizer.addEventListener("mousedown", (event) => {
    if (els.sideTabs.classList.contains("collapsed")) return;
    startY = event.clientY;
    startHeight = els.sideTabs.getBoundingClientRect().height;
    document.body.classList.add("resizing-tabs");
    document.addEventListener("mousemove", onMouseMove);
    document.addEventListener("mouseup", onMouseUp);
  });

  function onMouseMove(event) {
    const sidebarHeight = els.sidebar.getBoundingClientRect().height;
    const min = 112;
    const max = Math.max(180, sidebarHeight - 260);
    const height = Math.min(max, Math.max(min, startHeight + event.clientY - startY));
    els.sideTabs.style.height = `${height}px`;
  }

  function onMouseUp() {
    document.body.classList.remove("resizing-tabs");
    localStorage.setItem("routing-slip-studio:tabs-height", String(Math.round(els.sideTabs.getBoundingClientRect().height)));
    document.removeEventListener("mousemove", onMouseMove);
    document.removeEventListener("mouseup", onMouseUp);
  }
}
