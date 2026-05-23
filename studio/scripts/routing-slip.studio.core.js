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

const STUDIO_DB = {
  name: "routing-slip-studio",
  store: "state",
  key: "current",
  workspaceHandleKey: "workspace-handle",
  currentFileKey: "workspace-current-file",
};


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
  const restored = await restoreStudioState();
  if (!restored) loadExample("payment", { persist: false });
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
  [els.example, els.graphqlEndpoint, els.workflowEndpoint, els.externalApiUrl, els.integrations].forEach((input) => {
    input.addEventListener("change", () => {
      invalidateExecutionSnapshot();
      scheduleStudioSave();
    });
  });
  document.querySelector("#load-example").addEventListener("click", () => loadExample(els.example.value));
  document.querySelector("#lint-now").addEventListener("click", lintWorkflow);
  document.querySelector("#run-test").addEventListener("click", () => runLocalSimulation());
  els.reprocess.addEventListener("click", () => runLocalSimulation({ reprocess: true }));
  document.querySelector("#clear-logs").addEventListener("click", clearLogs);
  document.querySelector("#format-yaml").addEventListener("click", formatWorkflow);
  document.querySelector("#toggle-comment").addEventListener("click", () => toggleYamlComment());
  document.querySelector("#send-rest").addEventListener("click", sendToRestEndpoint);
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
