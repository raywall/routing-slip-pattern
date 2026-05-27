const handlers = new Set([
  "validate",
  "condition",
  "assert",
  "compute",
  "cel",
  "filter_array",
  "enrich",
  "transform",
  "notify",
  "audit",
  "graphql_enrich",
  "jump_if",
  "rest_call",
  "workflow_ref",
]);

const els = {
  workflow: document.querySelector("#workflow-editor"),
  highlight: document.querySelector("#workflow-highlight"),
  lineNumbers: document.querySelector("#editor-lines"),
  payload: document.querySelector("#payload-editor"),
  lint: document.querySelector("#lint-result"),
  lintBadge: document.querySelector("#lint-badge"),
  title: document.querySelector("#workflow-title"),
  lineCount: document.querySelector("#line-count"),
  timeline: document.querySelector("#timeline"),
  summary: document.querySelector("#run-summary"),
  example: document.querySelector("#example-select"),
  graphqlEndpoint: document.querySelector("#graphql-endpoint"),
  workflowEndpoint: document.querySelector("#workflow-endpoint"),
  mcpEndpoint: document.querySelector("#mcp-endpoint"),
  mcpApiKey: document.querySelector("#mcp-api-key"),
  externalApiUrl: document.querySelector("#external-api-url"),
  integrations: document.querySelector("#execute-integrations"),
  sidebar: document.querySelector("#sidebar"),
  resizer: document.querySelector("#sidebar-resizer"),
  sideTabs: document.querySelector("#side-tabs"),
  tabsResizer: document.querySelector("#tabs-resizer"),
  workspaceName: document.querySelector("#workspace-name"),
  workspaceTree: document.querySelector("#workspace-tree"),
  workspaceCurrent: document.querySelector("#workspace-current"),
  openWorkspace: document.querySelector("#open-workspace"),
  newService: document.querySelector("#new-service"),
  newWorkflow: document.querySelector("#new-workflow"),
  saveWorkflowFile: document.querySelector("#save-workflow-file"),
  exportWorkflowFile: document.querySelector("#export-workflow-file"),
  refreshWorkspace: document.querySelector("#refresh-workspace"),
  collapseTabs: document.querySelector("#collapse-tabs"),
  reprocess: document.querySelector("#reprocess-test"),
  themeToggle: document.querySelector("#theme-toggle"),
  mobileDocsToggle: document.querySelector("#mobile-docs-toggle"),
  mobileDocsBackdrop: document.querySelector("#mobile-docs-backdrop"),
};

let lastWorkflow = null;
let lastLint = [];
let activeExecutionStepIndex = null;
let activeLogEntry = null;
let activeStepGroup = null;
let stepGroups = new Map();
let saveTimer = null;
let workspaceState = {
  rootHandle: null,
  name: "",
  services: [],
};
let currentWorkspaceFile = {
  handle: null,
  serviceName: "",
  fileName: "",
};
let workflowDirty = false;
let activeRuntimeWorkflow = null;
let lastExecutionSnapshot = null;

async function boot() {
  initEnvSwitcher();

  const restored = await restoreStudioState();
  if (!restored) loadExample("payment", { persist: false });

  initTheme();
  renderIcons();
  bindEvents();
  initDocumentation();
  await restoreWorkspace();
  lintWorkflow();
}

function bindEvents() {
  els.workflow.addEventListener("input", () => {
    invalidateExecutionSnapshot();
    markWorkflowDirty();
    lintWorkflow();
    scheduleStudioSave();
  });
  els.workflow.addEventListener("scroll", syncLineNumbers);
  els.workflow.addEventListener("keydown", handleEditorKeydown);
  els.payload.addEventListener("input", () => {
    invalidateExecutionSnapshot();
    validatePayload();
    scheduleStudioSave();
  });
  els.payload.addEventListener("keydown", handlePayloadKeydown);
  [els.example, els.graphqlEndpoint, els.workflowEndpoint, els.mcpEndpoint, els.mcpApiKey, els.externalApiUrl, els.integrations].forEach((input) => {
    input.addEventListener("change", () => {
      invalidateExecutionSnapshot();
      scheduleStudioSave();
    });
  });
  document.querySelector("#load-example").addEventListener("click", () => loadExample(els.example.value));
  document.querySelector("#lint-now").addEventListener("click", lintWorkflow);
  document.querySelector("#run-test").addEventListener("click", () => runLocalSimulation());
  els.reprocess.addEventListener("click", () => runLocalSimulation({ reprocess: true }));
  els.themeToggle.addEventListener("click", toggleTheme);
  els.mobileDocsToggle.addEventListener("click", toggleMobileDocs);
  els.mobileDocsBackdrop.addEventListener("click", closeMobileDocs);
  document.querySelector("#clear-logs").addEventListener("click", clearLogs);
  document.querySelector("#format-yaml").addEventListener("click", formatWorkflow);
  document.querySelector("#toggle-comment").addEventListener("click", () => toggleYamlComment());
  document.querySelector("#send-rest").addEventListener("click", sendToRestEndpoint);
  document.querySelector("#mcp-validate").addEventListener("click", validateWorkflowViaMCP);
  document.querySelector("#mcp-explain").addEventListener("click", explainWorkflowViaMCP);
  document.querySelector("#mcp-diagnostics").addEventListener("click", showConnectorDiagnostics);
  document.querySelectorAll("[data-tab]").forEach((button) => {
    button.addEventListener("click", () => activateTab(button.dataset.tab));
  });
  els.collapseTabs.addEventListener("click", toggleTabsCollapsed);
  els.openWorkspace.addEventListener("click", openWorkspace);
  els.newService.addEventListener("click", createService);
  els.newWorkflow.addEventListener("click", createWorkflowInActiveService);
  els.saveWorkflowFile.addEventListener("click", saveCurrentWorkflowFile);
  els.exportWorkflowFile.addEventListener("click", exportComposedWorkflow);
  els.refreshWorkspace.addEventListener("click", refreshWorkspace);
  document.addEventListener("click", dismissContextMenu);
  document.addEventListener("keydown", (event) => {
    if (event.defaultPrevented) return;
    if (isSaveShortcut(event)) {
      event.preventDefault();
      event.stopPropagation();
      saveCurrentWorkflowFile();
    }
  });
  bindSidebarResize();
}

function renderIcons() {
  if (window.lucide && typeof window.lucide.createIcons === "function") {
    window.lucide.createIcons();
  }
}

function initTheme() {
  const stored = localStorage.getItem("routing-slip-studio:theme");
  const prefersDark = window.matchMedia?.("(prefers-color-scheme: dark)")?.matches;
  applyTheme(stored || (prefersDark ? "dark" : "light"));
}

function toggleTheme() {
  const current = document.body.dataset.theme === "dark" ? "dark" : "light";
  applyTheme(current === "dark" ? "light" : "dark");
}

function applyTheme(theme) {
  const normalized = theme === "dark" ? "dark" : "light";
  document.body.dataset.theme = normalized;
  localStorage.setItem("routing-slip-studio:theme", normalized);
  if (!els.themeToggle) return;
  const icon = normalized === "dark" ? "fa-sun" : "fa-moon";
  const label = normalized === "dark" ? "Alternar para tema claro" : "Alternar para tema escuro";
  els.themeToggle.title = label;
  els.themeToggle.setAttribute("aria-label", label);
  els.themeToggle.innerHTML = `<i class="fa-solid ${icon}" aria-hidden="true"></i>`;
  window.dispatchEvent(new CustomEvent("routing-slip-theme-change", { detail: { theme: normalized } }));
}

function toggleMobileDocs() {
  document.body.classList.toggle("mobile-docs-open");
}

function closeMobileDocs() {
  document.body.classList.remove("mobile-docs-open");
}

function initDocumentation() {
  if (!window.RoutingSlipDocsViewer) return;
  window.RoutingSlipDocsViewer.init({
    treeSelector: "#docs-tree",
    timelineSelector: "#timeline",
    titleSelector: "#workflow-title",
    summarySelector: "#run-summary",
    eyebrowSelector: "#workspace-mode-label",
    panelTitleSelector: "#result-panel-title",
    panelMetaSelector: "#result-panel-meta",
  });
}

function initEnvSwitcher() {
  const meta = document.querySelector('meta[name="studio-env"]');
  if (!meta) return;

  const currentEnv = meta.getAttribute('content');
  if (!["production", "development"].includes(currentEnv)) return;
  if (document.querySelector("#studio-env-switcher")) return;

  const repoName = window.location.pathname.split('/')[1] || "routing-slip-pattern";
  const ENVS = {
    production: { label: "Producao", path: `/${repoName}/` },
    development: { label: "Desenvolvimento", path: `/${repoName}/dev/` },
  };

  const select = document.createElement('select');
  select.id = 'studio-env-switcher';
  select.className = "env-switcher";
  select.title = "Alternar ambiente da documentacao";
  select.setAttribute("aria-label", "Alternar ambiente da documentacao");

  for (const [key, { label }] of Object.entries(ENVS)) {
    const opt = document.createElement('option');
    opt.value = key;
    opt.textContent = label;
    if (key === currentEnv) opt.selected = true;
    select.appendChild(opt);
  }

  select.addEventListener('change', () => {
    window.location.href = ENVS[select.value].path;
  });

  const toolbar = document.querySelector(".topbar-actions");
  if (!toolbar) return;
  toolbar.prepend(select);
}
