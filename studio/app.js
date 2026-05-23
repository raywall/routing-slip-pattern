const handlers = new Set([
  "validate",
  "condition",
  "assert",
  "compute",
  "cel",
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

const examples = {
  payment: {
    workflow: `name: payment-event-fulfillment
description: Evento de pagamento aprovado consulta pedido, emite nota fiscal, aciona expedicao e baixa estoque.
error_policy: stop
message_id_path: payload.pagamento_id
correlation_id_path: correlation_id
steps:
  - name: validate
    params:
      required:
        - evento
        - payload.pagamento_id
        - payload.pedido_id
        - payload.valor_pago

  - name: condition
    params:
      field: evento
      equals: PAGAMENTO_APROVADO

  - name: graphql_enrich
    params:
      query: "query ($pedidoID: String!) { dataSources(pedidoID: $pedidoID) { order { pedido_id cliente_id status valor_total endereco_entrega itens { produto_id quantidade } } } }"
      variables:
        pedidoID: "{payload.pedido_id}"
      target: pedido
      result_path: dataSources.order
      timeout_ms: 1500
      required: true

  - name: rest_call
    params:
      base_url: "\${EXTERNAL_API_URL:-https://mock.raysouz.studio}"
      method: POST
      endpoint: /lambda/notas-fiscais
      target: nota_fiscal
      body:
        pedido_id: "{pedido.pedido_id}"
        cliente_id: "{pedido.cliente_id}"
        valor_total: "{pedido.valor_total}"
        itens: "{pedido.itens}"
        pagamento_id: "{payload.pagamento_id}"
      required: true

  - name: audit
    params:
      event: payment.fulfillment.completed
      fields:
        - correlation_id
        - payload.pedido_id
        - pedido.status
        - nota_fiscal.status
`,
    payload: {
      evento: "PAGAMENTO_APROVADO",
      payload: {
        pagamento_id: "PAG-5544",
        pedido_id: "PED-9988",
        valor_pago: 150,
      },
      correlation_id: "corr-payment-fulfillment-001",
      received_at: "2026-05-21T00:00:00Z",
    },
  },
  baixa: {
    workflow: `name: Processamento de desconto em folha - baixa de parcelas
description: Preparacao da baixa de parcelas ate antes da execucao do Step Functions.
error_policy: stop
message_id_path: data.codigo_identificador_evento
correlation_id_path: correlation_id
steps:
  - name: validate
    params:
      required:
        - data.codigo_identificador_evento
        - data.event_name
        - data.codigo_identificacao_pessoa
        - data.codigo_identificacao_operacao_credito
        - data.valor_desconto

  - name: condition
    params:
      field: data.event_name
      equals: DESCONTO_FOLHA_REALIZADO

  - name: graphql_enrich
    params:
      query: "query ($codigoCliente: String!, $identificadorOperacaoCredito: String!, $dataPosicaoCalculo: String!) { dataSources(codigoCliente: $codigoCliente, identificadorOperacaoCredito: $identificadorOperacaoCredito, dataPosicaoCalculo: $dataPosicaoCalculo) { custodias { operacaoId situacaoOperacao siglaCustodia saldoDevedor } saldos { saldo { saldo_liquido_operacao } } } }"
      variables:
        codigoCliente: "{data.codigo_identificacao_pessoa}"
        identificadorOperacaoCredito: "{data.codigo_identificacao_operacao_credito}"
        dataPosicaoCalculo: "2025-05-13"
      target: baixa_contexto
      result_path: dataSources
      timeout_ms: 3000
      required: true

  - name: enrich
    params:
      data:
        workflow_input_preparado:
          status: PRONTO_PARA_STEP_FUNCTIONS
          codigo_origem_desconto: DESCONTO_EM_FOLHA

  - name: audit
    params:
      event: baixa_parcelas.preparacao.completed
      fields:
        - correlation_id
        - data.codigo_identificador_evento
        - workflow_input_preparado.status
`,
    payload: {
      correlation_id: "corr-baixa-parcelas-001",
      received_at: "2026-05-21T00:00:00Z",
      data: {
        event_name: "DESCONTO_FOLHA_REALIZADO",
        codigo_identificador_evento: "evt-abc12345-def6-7890-ghij-klmn12345678",
        codigo_identificacao_pessoa: "12345678901",
        codigo_identificacao_operacao_credito: "2699999999",
        valor_desconto: 82.6,
      },
    },
  },
  blank: {
    workflow: `name: novo-workflow
description: Workflow em construcao.
error_policy: stop
message_id_path: id
correlation_id_path: correlation_id
steps:
  - name: validate
    params:
      required:
        - id
        - correlation_id

  - name: audit
    params:
      event: novo_workflow.completed
      fields:
        - id
        - correlation_id
`,
    payload: {
      id: "MSG-001",
      correlation_id: "corr-001",
    },
  },
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
    markWorkflowDirty();
    lintWorkflow();
    scheduleStudioSave();
  });
  els.workflow.addEventListener("scroll", syncLineNumbers);
  els.workflow.addEventListener("keydown", handleEditorKeydown);
  els.payload.addEventListener("input", () => {
    validatePayload();
    scheduleStudioSave();
  });
  [els.example, els.graphqlEndpoint, els.workflowEndpoint, els.externalApiUrl, els.integrations].forEach((input) => {
    input.addEventListener("change", scheduleStudioSave);
  });
  document.querySelector("#load-example").addEventListener("click", () => loadExample(els.example.value));
  document.querySelector("#lint-now").addEventListener("click", lintWorkflow);
  document.querySelector("#run-test").addEventListener("click", runLocalSimulation);
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

function loadExample(key, options = {}) {
  if (workflowDirty && !confirm("Descartar alteracoes nao salvas no workflow atual?")) return;
  const example = examples[key] || examples.payment;
  els.example.value = key;
  els.workflow.value = example.workflow;
  els.payload.value = JSON.stringify(example.payload, null, 2);
  currentWorkspaceFile = { handle: null, serviceName: "", fileName: "" };
  workflowDirty = false;
  renderWorkspace();
  clearLogs();
  lintWorkflow();
  if (options.persist !== false) scheduleStudioSave();
}

function handleEditorKeydown(event) {
  if (isRunShortcut(event)) {
    event.preventDefault();
    runLocalSimulation();
    return;
  }
  if (isSaveShortcut(event)) {
    event.preventDefault();
    event.stopPropagation();
    saveCurrentWorkflowFile();
    return;
  }
  if ((event.metaKey || event.ctrlKey) && event.key === "/") {
    event.preventDefault();
    toggleYamlComment();
    return;
  }
  if (event.key === "Enter") {
    event.preventDefault();
    insertIndentedNewline(event.currentTarget);
    return;
  }
  if (event.key !== "Tab") return;
  event.preventDefault();
  const editor = event.currentTarget;
  const start = editor.selectionStart;
  const end = editor.selectionEnd;
  const value = editor.value;
  const indent = "  ";

  if (start !== end && value.slice(start, end).includes("\n")) {
    const lineStart = value.lastIndexOf("\n", start - 1) + 1;
    const selected = value.slice(lineStart, end);
    const replacement = selected.split("\n").map((line) => {
      if (!event.shiftKey) return indent + line;
      if (line.startsWith(indent)) return line.slice(indent.length);
      if (line.startsWith(" ")) return line.slice(1);
      return line;
    }).join("\n");
    editor.value = value.slice(0, lineStart) + replacement + value.slice(end);
    editor.selectionStart = lineStart;
    editor.selectionEnd = lineStart + replacement.length;
  } else if (event.shiftKey) {
    const lineStart = value.lastIndexOf("\n", start - 1) + 1;
    const beforeCursor = value.slice(lineStart, start);
    if (beforeCursor.endsWith(indent)) {
      editor.value = value.slice(0, start - indent.length) + value.slice(end);
      editor.selectionStart = editor.selectionEnd = start - indent.length;
    }
  } else {
    editor.value = value.slice(0, start) + indent + value.slice(end);
    editor.selectionStart = editor.selectionEnd = start + indent.length;
  }
  lintWorkflow();
  markWorkflowDirty();
  scheduleStudioSave();
}

function insertIndentedNewline(editor) {
  const start = editor.selectionStart;
  const end = editor.selectionEnd;
  const value = editor.value;
  const lineStart = value.lastIndexOf("\n", start - 1) + 1;
  const line = value.slice(lineStart, start);
  const indent = nextLineIndent(line);
  editor.value = value.slice(0, start) + "\n" + indent + value.slice(end);
  editor.selectionStart = editor.selectionEnd = start + 1 + indent.length;
  lintWorkflow();
  markWorkflowDirty();
  scheduleStudioSave();
}

function nextLineIndent(line) {
  const stepStart = line.match(/^(\s*-\s+)(id|name)\s*:/);
  if (stepStart) return " ".repeat(stepStart[1].length);
  return line.match(/^\s*/)[0];
}

function isRunShortcut(event) {
  if (event.key !== "Enter") return false;
  const isApple = /Mac|iPhone|iPad|iPod/.test(navigator.platform || "");
  return isApple ? event.metaKey && !event.ctrlKey : event.ctrlKey && !event.metaKey;
}

function isSaveShortcut(event) {
  return event.key.toLowerCase() === "s" && (event.ctrlKey || event.metaKey);
}

function toggleYamlComment() {
  const editor = els.workflow;
  const value = editor.value;
  const start = editor.selectionStart;
  const end = editor.selectionEnd;
  const lineStart = value.lastIndexOf("\n", start - 1) + 1;
  const lineEnd = end > start && value[end - 1] === "\n" ? end - 1 : end;
  const selectionEnd = value.indexOf("\n", lineEnd);
  const blockEnd = selectionEnd === -1 ? value.length : selectionEnd;
  const block = value.slice(lineStart, blockEnd);
  const lines = block.split("\n");
  const firstEditable = lines.find((line) => line.trim() !== "");
  const commentColumn = firstEditable ? firstEditable.match(/^\s*/)[0].length : 0;
  const editableLines = lines.filter((line) => line.trim() !== "");
  const shouldUncomment = editableLines.length > 0 && editableLines.every((line) => {
    const prefix = line.slice(0, commentColumn);
    const marker = line.slice(commentColumn);
    return /^\s*$/.test(prefix) && /^# ?/.test(marker);
  });
  const changed = lines.map((line) => {
    if (line.trim() === "") return line;
    if (shouldUncomment) {
      return line.slice(0, commentColumn) + line.slice(commentColumn).replace(/^# ?/, "");
    }
    const padding = " ".repeat(Math.max(0, commentColumn - line.match(/^\s*/)[0].length));
    return `${line.slice(0, commentColumn)}# ${padding}${line.slice(commentColumn)}`;
  }).join("\n");

  editor.value = value.slice(0, lineStart) + changed + value.slice(blockEnd);
  editor.selectionStart = lineStart;
  editor.selectionEnd = lineStart + changed.length;
  lintWorkflow();
  markWorkflowDirty();
  scheduleStudioSave();
}

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

function currentStudioState() {
  return {
    workflow: els.workflow.value,
    payload: els.payload.value,
    example: els.example.value,
    graphqlEndpoint: els.graphqlEndpoint.value,
    workflowEndpoint: els.workflowEndpoint.value,
    externalApiUrl: els.externalApiUrl.value,
    executeIntegrations: els.integrations.checked,
    updatedAt: new Date().toISOString(),
  };
}

async function restoreStudioState() {
  const state = await StudioDB.get(STUDIO_DB.key);
  if (!state) return false;
  els.workflow.value = typeof state.workflow === "string" ? state.workflow : "";
  els.payload.value = typeof state.payload === "string" ? state.payload : "{}";
  if (state.example && examples[state.example]) els.example.value = state.example;
  if (state.graphqlEndpoint) els.graphqlEndpoint.value = state.graphqlEndpoint;
  if (state.workflowEndpoint) els.workflowEndpoint.value = state.workflowEndpoint;
  if (state.externalApiUrl) els.externalApiUrl.value = state.externalApiUrl;
  els.integrations.checked = Boolean(state.executeIntegrations);
  return els.workflow.value.trim() !== "";
}

function scheduleStudioSave() {
  clearTimeout(saveTimer);
  saveTimer = setTimeout(() => {
    StudioDB.set(STUDIO_DB.key, currentStudioState());
  }, 250);
}

const StudioDB = {
  open() {
    return new Promise((resolve, reject) => {
      if (!window.indexedDB) {
        reject(new Error("IndexedDB indisponivel"));
        return;
      }
      const request = indexedDB.open(STUDIO_DB.name, 1);
      request.onupgradeneeded = () => {
        if (!request.result.objectStoreNames.contains(STUDIO_DB.store)) {
          request.result.createObjectStore(STUDIO_DB.store);
        }
      };
      request.onsuccess = () => resolve(request.result);
      request.onerror = () => reject(request.error);
    });
  },

  async get(key) {
    try {
      const db = await this.open();
      const value = await new Promise((resolve, reject) => {
        const tx = db.transaction(STUDIO_DB.store, "readonly");
        const request = tx.objectStore(STUDIO_DB.store).get(key);
        request.onsuccess = () => resolve(request.result || null);
        request.onerror = () => reject(request.error);
      });
      db.close();
      return value;
    } catch (_) {
      try {
        return JSON.parse(localStorage.getItem(`routing-slip-studio:${key}`) || "null");
      } catch {
        return null;
      }
    }
  },

  async set(key, value) {
    try {
      const db = await this.open();
      await new Promise((resolve, reject) => {
        const tx = db.transaction(STUDIO_DB.store, "readwrite");
        tx.objectStore(STUDIO_DB.store).put(value, key);
        tx.oncomplete = resolve;
        tx.onerror = () => reject(tx.error);
      });
      db.close();
    } catch (_) {
      localStorage.setItem(`routing-slip-studio:${key}`, JSON.stringify(value));
    }
  },
};

const WorkspaceFS = {
  supported: typeof window.showDirectoryPicker === "function",

  async ensurePermission(handle) {
    if (!handle) return false;
    const options = { mode: "readwrite" };
    if (typeof handle.queryPermission === "function") {
      const current = await handle.queryPermission(options);
      if (current === "granted") return true;
    }
    if (typeof handle.requestPermission === "function") {
      try {
        return await handle.requestPermission(options) === "granted";
      } catch {
        return false;
      }
    }
    return true;
  },

  async buildTree(rootHandle) {
    const expanded = new Set(loadExpandedServices());
    const services = [];
    for await (const [name, handle] of rootHandle.entries()) {
      if (name.startsWith(".") || name === "node_modules") continue;
      if (handle.kind !== "directory") continue;
      const files = [];
      for await (const [fileName, fileHandle] of handle.entries()) {
        if (fileHandle.kind === "file" && /\.(ya?ml)$/i.test(fileName)) {
          files.push({ name: fileName, handle: fileHandle });
        }
      }
      files.sort((a, b) => a.name.localeCompare(b.name));
      services.push({ name, handle, expanded: expanded.has(name), files });
    }
    services.sort((a, b) => a.name.localeCompare(b.name));
    return services;
  },

  async readFile(fileHandle) {
    const file = await fileHandle.getFile();
    return file.text();
  },

  async writeFile(fileHandle, content) {
    const writable = await fileHandle.createWritable();
    await writable.write(content);
    await writable.close();
  },

  async createFile(dirHandle, fileName, content) {
    const fileHandle = await dirHandle.getFileHandle(fileName, { create: true });
    await this.writeFile(fileHandle, content);
    return fileHandle;
  },
};

async function openWorkspace() {
  if (!WorkspaceFS.supported) {
    alert("Workspace local requer Chrome ou Edge com suporte a File System Access API.");
    return;
  }
  try {
    const handle = await window.showDirectoryPicker({ mode: "readwrite" });
    workspaceState.rootHandle = handle;
    workspaceState.name = handle.name;
    workspaceState.services = await WorkspaceFS.buildTree(handle);
    await StudioDB.set(STUDIO_DB.workspaceHandleKey, handle);
    renderWorkspace();
  } catch (err) {
    if (err.name !== "AbortError") alert(`Erro ao abrir workspace: ${err.message}`);
  }
}

async function restoreWorkspace() {
  if (!WorkspaceFS.supported) {
    renderWorkspace();
    return;
  }
  const handle = await StudioDB.get(STUDIO_DB.workspaceHandleKey);
  if (!handle) {
    renderWorkspace();
    return;
  }
  const allowed = await WorkspaceFS.ensurePermission(handle);
  if (!allowed) {
    renderWorkspace();
    return;
  }
  try {
    workspaceState.rootHandle = handle;
    workspaceState.name = handle.name;
    workspaceState.services = await WorkspaceFS.buildTree(handle);
    renderWorkspace();
    const current = await StudioDB.get(STUDIO_DB.currentFileKey);
    if (current?.serviceName && current?.fileName) {
      await openWorkflowFile(current.serviceName, current.fileName, { skipDirtyCheck: true });
    }
  } catch (err) {
    console.warn("Nao foi possivel restaurar workspace:", err);
    renderWorkspace();
  }
}

async function refreshWorkspace() {
  if (!workspaceState.rootHandle) return;
  saveExpandedServices();
  workspaceState.services = await WorkspaceFS.buildTree(workspaceState.rootHandle);
  if (currentWorkspaceFile.serviceName && currentWorkspaceFile.fileName) {
    const service = findService(currentWorkspaceFile.serviceName);
    const file = service?.files.find((item) => item.name === currentWorkspaceFile.fileName);
    currentWorkspaceFile.handle = file?.handle || null;
  }
  renderWorkspace();
}

function renderWorkspace() {
  els.workspaceName.textContent = workspaceState.name || "Nenhum diretorio";
  const enabled = Boolean(workspaceState.rootHandle);
  [els.newService, els.refreshWorkspace].forEach((button) => {
    button.disabled = !enabled;
  });
  els.newWorkflow.disabled = !enabled || workspaceState.services.length === 0;
  els.saveWorkflowFile.disabled = !currentWorkspaceFile.handle || !workflowDirty;
  els.exportWorkflowFile.disabled = !currentWorkspaceFile.handle;

  if (!enabled) {
    els.workspaceTree.innerHTML = `
      <div class="workspace-empty">
        <div>Nenhum workspace aberto</div>
        <button type="button" data-open-empty>Abrir pasta</button>
      </div>`;
    els.workspaceTree.querySelector("[data-open-empty]")?.addEventListener("click", openWorkspace);
    els.workspaceCurrent.textContent = WorkspaceFS.supported
      ? "Abra um diretorio para organizar microservicos e workflows YAML."
      : "Seu navegador nao suporta workspace local.";
    return;
  }

  if (!workspaceState.services.length) {
    els.workspaceTree.innerHTML = `
      <div class="workspace-empty">
        <div>Workspace vazio</div>
        <small>Crie um microservico para comecar.</small>
      </div>`;
  } else {
    els.workspaceTree.innerHTML = workspaceState.services.map(renderServiceNode).join("");
    bindWorkspaceTree();
    renderIcons();
  }

  const path = currentWorkspaceFile.serviceName && currentWorkspaceFile.fileName
    ? `${workspaceState.name}/${currentWorkspaceFile.serviceName}/${currentWorkspaceFile.fileName}${workflowDirty ? " *" : ""}`
    : "Nenhum workflow aberto.";
  els.workspaceCurrent.textContent = path;
}

function renderServiceNode(service) {
  const active = currentWorkspaceFile.serviceName === service.name;
  const files = service.expanded ? service.files.map((file) => renderWorkflowFile(service.name, file)).join("") : "";
  return `
    <div class="workspace-service ${active ? "active" : ""}">
      <button class="workspace-service-head" type="button" data-service="${escapeHtml(service.name)}">
        <span>${service.expanded ? "▾" : "▸"}</span>
        <i class="workspace-icon" data-lucide="folder"></i>
        <span class="workspace-name">${escapeHtml(service.name)}</span>
        <span class="workspace-count">${service.files.length}</span>
      </button>
      <div>${files}</div>
    </div>`;
}

function renderWorkflowFile(serviceName, file) {
  const active = currentWorkspaceFile.serviceName === serviceName && currentWorkspaceFile.fileName === file.name;
  const label = file.name.replace(/\.(ya?ml)$/i, "");
  return `
    <button class="workspace-file ${active ? "active" : ""}" type="button" data-service="${escapeHtml(serviceName)}" data-file="${escapeHtml(file.name)}">
      <i class="workspace-icon" data-lucide="file"></i>
      <span class="workspace-file-name">${escapeHtml(label)}</span>
      <span class="workspace-dirty">${active && workflowDirty ? "●" : ""}</span>
    </button>`;
}

function bindWorkspaceTree() {
  els.workspaceTree.querySelectorAll(".workspace-service-head").forEach((button) => {
    button.addEventListener("click", () => toggleService(button.dataset.service));
    button.addEventListener("contextmenu", (event) => {
      event.preventDefault();
      showContextMenu(event.clientX, event.clientY, [
        { label: "Novo workflow", action: () => createWorkflow(button.dataset.service) },
        { label: "Renomear microservico", action: () => renameService(button.dataset.service) },
        { separator: true },
        { label: "Excluir microservico", action: () => deleteService(button.dataset.service), danger: true },
      ]);
    });
  });
  els.workspaceTree.querySelectorAll(".workspace-file").forEach((button) => {
    button.addEventListener("click", () => openWorkflowFile(button.dataset.service, button.dataset.file));
    button.addEventListener("contextmenu", (event) => {
      event.preventDefault();
      showContextMenu(event.clientX, event.clientY, [
        { label: "Abrir", action: () => openWorkflowFile(button.dataset.service, button.dataset.file) },
        { label: "Salvar", action: () => saveCurrentWorkflowFile() },
        { label: "Renomear workflow", action: () => renameWorkflow(button.dataset.service, button.dataset.file) },
        { separator: true },
        { label: "Excluir workflow", action: () => deleteWorkflow(button.dataset.service, button.dataset.file), danger: true },
      ]);
    });
  });
}

function toggleService(serviceName) {
  const service = findService(serviceName);
  if (!service) return;
  service.expanded = !service.expanded;
  saveExpandedServices();
  renderWorkspace();
}

async function openWorkflowFile(serviceName, fileName, options = {}) {
  if (workflowDirty && !options.skipDirtyCheck) {
    const save = confirm(`"${currentWorkspaceFile.fileName}" tem alteracoes nao salvas. Salvar antes de continuar?`);
    if (save) await saveCurrentWorkflowFile();
  }
  const service = findService(serviceName);
  const file = service?.files.find((item) => item.name === fileName);
  if (!file) return;
  try {
    els.workflow.value = await WorkspaceFS.readFile(file.handle);
    currentWorkspaceFile = { handle: file.handle, serviceName, fileName };
    workflowDirty = false;
    service.expanded = true;
    await StudioDB.set(STUDIO_DB.currentFileKey, { serviceName, fileName });
    renderWorkspace();
    clearLogs();
    lintWorkflow();
    scheduleStudioSave();
  } catch (err) {
    alert(`Erro ao abrir workflow: ${err.message}`);
  }
}

async function saveCurrentWorkflowFile() {
  if (!currentWorkspaceFile.handle) {
    alert("Abra ou crie um workflow no workspace antes de salvar.");
    return;
  }
  try {
    await WorkspaceFS.writeFile(currentWorkspaceFile.handle, els.workflow.value);
    workflowDirty = false;
    renderWorkspace();
    scheduleStudioSave();
  } catch (err) {
    alert(`Erro ao salvar workflow: ${err.message}`);
  }
}

async function exportComposedWorkflow() {
  if (!currentWorkspaceFile.handle) {
    alert("Abra um workflow no workspace antes de exportar.");
    return;
  }
  const issues = lintWorkflow();
  if (issues.some((item) => item.level === "error")) {
    alert("Corrija os erros do workflow antes de exportar.");
    return;
  }
  try {
    const expanded = await expandWorkflowRefsForStudio(lastWorkflow);
    const exportable = stripStudioRuntimeFields(expanded);
    const yaml = window.jsyaml.dump(exportable, {
      lineWidth: 140,
      noRefs: true,
      sortKeys: false,
    });
    const baseName = (currentWorkspaceFile.fileName || "workflow.yaml").replace(/\.(ya?ml)$/i, "");
    downloadTextFile(`${baseName}-bundle.yaml`, yaml);
  } catch (err) {
    alert(`Erro ao exportar workflow composto: ${err.message}`);
  }
}

function stripStudioRuntimeFields(workflow) {
  const clean = structuredClone(workflow);
  clean.steps = (clean.steps || []).map((step) => {
    const next = { ...step };
    delete next.__sourceWorkflow;
    if (next.params && typeof next.params === "object") {
      next.params = stripPrivateFields(next.params);
    }
    return next;
  });
  return clean;
}

function stripPrivateFields(value) {
  if (Array.isArray(value)) return value.map(stripPrivateFields);
  if (!value || typeof value !== "object") return value;
  return Object.fromEntries(
    Object.entries(value)
      .filter(([key]) => !key.startsWith("__"))
      .map(([key, item]) => [key, stripPrivateFields(item)])
  );
}

function downloadTextFile(fileName, content) {
  const blob = new Blob([content], { type: "text/yaml;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = fileName;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

async function createService() {
  if (!workspaceState.rootHandle) return;
  const raw = prompt("Nome do microservico:");
  if (!raw?.trim()) return;
  const name = normalizeName(raw);
  if (findService(name)) {
    alert(`O microservico "${name}" ja existe.`);
    return;
  }
  try {
    const handle = await workspaceState.rootHandle.getDirectoryHandle(name, { create: true });
    workspaceState.services.push({ name, handle, expanded: true, files: [] });
    workspaceState.services.sort((a, b) => a.name.localeCompare(b.name));
    saveExpandedServices();
    renderWorkspace();
  } catch (err) {
    alert(`Erro ao criar microservico: ${err.message}`);
  }
}

async function createWorkflowInActiveService() {
  const target = currentWorkspaceFile.serviceName || workspaceState.services[0]?.name;
  if (!target) {
    alert("Crie um microservico primeiro.");
    return;
  }
  await createWorkflow(target);
}

async function createWorkflow(serviceName) {
  const service = findService(serviceName);
  if (!service) return;
  const raw = prompt(`Nome do workflow em "${serviceName}":`);
  if (!raw?.trim()) return;
  const fileName = `${normalizeName(raw).replace(/\.(ya?ml)$/i, "")}.yaml`;
  if (service.files.some((file) => file.name === fileName)) {
    alert(`O workflow "${fileName}" ja existe.`);
    return;
  }
  try {
    const content = workflowTemplate(fileName);
    const handle = await WorkspaceFS.createFile(service.handle, fileName, content);
    service.files.push({ name: fileName, handle });
    service.files.sort((a, b) => a.name.localeCompare(b.name));
    service.expanded = true;
    saveExpandedServices();
    renderWorkspace();
    await openWorkflowFile(serviceName, fileName, { skipDirtyCheck: true });
  } catch (err) {
    alert(`Erro ao criar workflow: ${err.message}`);
  }
}

async function renameService(serviceName) {
  const service = findService(serviceName);
  if (!service) return;
  const raw = prompt("Novo nome do microservico:", serviceName);
  if (!raw?.trim()) return;
  const newName = normalizeName(raw);
  if (newName === serviceName) return;
  if (findService(newName)) {
    alert(`O microservico "${newName}" ja existe.`);
    return;
  }
  if (!confirm(`Renomear "${serviceName}" para "${newName}"?`)) return;
  try {
    const newDir = await workspaceState.rootHandle.getDirectoryHandle(newName, { create: true });
    for (const file of service.files) {
      await WorkspaceFS.createFile(newDir, file.name, await WorkspaceFS.readFile(file.handle));
    }
    await workspaceState.rootHandle.removeEntry(serviceName, { recursive: true });
    if (currentWorkspaceFile.serviceName === serviceName) {
      currentWorkspaceFile.serviceName = newName;
      await StudioDB.set(STUDIO_DB.currentFileKey, {
        serviceName: newName,
        fileName: currentWorkspaceFile.fileName,
      });
    }
    await refreshWorkspace();
  } catch (err) {
    alert(`Erro ao renomear microservico: ${err.message}`);
  }
}

async function renameWorkflow(serviceName, fileName) {
  const service = findService(serviceName);
  const file = service?.files.find((item) => item.name === fileName);
  if (!service || !file) return;
  const raw = prompt("Novo nome do workflow:", fileName);
  if (!raw?.trim()) return;
  const newName = `${normalizeName(raw).replace(/\.(ya?ml)$/i, "")}.yaml`;
  if (newName === fileName) return;
  if (service.files.some((item) => item.name === newName)) {
    alert(`O workflow "${newName}" ja existe.`);
    return;
  }
  try {
    const content = currentWorkspaceFile.serviceName === serviceName && currentWorkspaceFile.fileName === fileName
      ? els.workflow.value
      : await WorkspaceFS.readFile(file.handle);
    const newHandle = await WorkspaceFS.createFile(service.handle, newName, content);
    await service.handle.removeEntry(fileName);
    file.name = newName;
    file.handle = newHandle;
    if (currentWorkspaceFile.serviceName === serviceName && currentWorkspaceFile.fileName === fileName) {
      currentWorkspaceFile = { handle: newHandle, serviceName, fileName: newName };
      workflowDirty = false;
      await StudioDB.set(STUDIO_DB.currentFileKey, { serviceName, fileName: newName });
    }
    service.files.sort((a, b) => a.name.localeCompare(b.name));
    renderWorkspace();
  } catch (err) {
    alert(`Erro ao renomear workflow: ${err.message}`);
  }
}

async function deleteService(serviceName) {
  if (!workspaceState.rootHandle) return;
  if (!confirm(`Excluir a pasta "${serviceName}" e todos os workflows dentro dela?`)) return;
  try {
    await workspaceState.rootHandle.removeEntry(serviceName, { recursive: true });
    if (currentWorkspaceFile.serviceName === serviceName) {
      currentWorkspaceFile = { handle: null, serviceName: "", fileName: "" };
      workflowDirty = false;
      await StudioDB.set(STUDIO_DB.currentFileKey, null);
    }
    await refreshWorkspace();
  } catch (err) {
    alert(`Erro ao excluir microservico: ${err.message}`);
  }
}

async function deleteWorkflow(serviceName, fileName) {
  const service = findService(serviceName);
  if (!service) return;
  if (!confirm(`Excluir o workflow "${fileName}"?`)) return;
  try {
    await service.handle.removeEntry(fileName);
    service.files = service.files.filter((file) => file.name !== fileName);
    if (currentWorkspaceFile.serviceName === serviceName && currentWorkspaceFile.fileName === fileName) {
      currentWorkspaceFile = { handle: null, serviceName: "", fileName: "" };
      workflowDirty = false;
      await StudioDB.set(STUDIO_DB.currentFileKey, null);
    }
    renderWorkspace();
  } catch (err) {
    alert(`Erro ao excluir workflow: ${err.message}`);
  }
}

function markWorkflowDirty() {
  if (!currentWorkspaceFile.handle) return;
  workflowDirty = true;
  els.saveWorkflowFile.disabled = false;
  renderWorkspace();
}

function findService(serviceName) {
  return workspaceState.services.find((service) => service.name === serviceName);
}

function normalizeName(value) {
  return String(value)
    .trim()
    .toLowerCase()
    .replace(/\.(ya?ml)$/i, "")
    .replace(/[^a-z0-9._-]+/g, "-")
    .replace(/^-+|-+$/g, "") || "workflow";
}

function workflowTemplate(fileName) {
  const name = fileName.replace(/\.(ya?ml)$/i, "");
  return `name: ${name}
description: Workflow em construcao.
error_policy: stop
message_id_path: id
correlation_id_path: correlation_id
steps:
  - name: validate
    params:
      required:
        - id
        - correlation_id

  - name: audit
    params:
      event: ${name}.completed
      fields:
        - id
        - correlation_id
`;
}

function saveExpandedServices() {
  const expanded = workspaceState.services.filter((service) => service.expanded).map((service) => service.name);
  localStorage.setItem("routing-slip-studio:workspace-expanded", JSON.stringify(expanded));
}

function loadExpandedServices() {
  try {
    const expanded = JSON.parse(localStorage.getItem("routing-slip-studio:workspace-expanded") || "[]");
    return Array.isArray(expanded) ? expanded : [];
  } catch {
    return [];
  }
}

function showContextMenu(x, y, items) {
  dismissContextMenu();
  const menu = document.createElement("div");
  menu.className = "context-menu";
  items.forEach((item) => {
    if (item.separator) {
      menu.appendChild(Object.assign(document.createElement("div"), { className: "context-sep" }));
      return;
    }
    const button = document.createElement("button");
    button.type = "button";
    button.textContent = item.label;
    if (item.danger) button.classList.add("danger");
    button.addEventListener("click", () => {
      dismissContextMenu();
      item.action();
    });
    menu.appendChild(button);
  });
  document.body.appendChild(menu);
  const rect = menu.getBoundingClientRect();
  menu.style.left = `${Math.min(x, window.innerWidth - rect.width - 8)}px`;
  menu.style.top = `${Math.min(y, window.innerHeight - rect.height - 8)}px`;
}

function dismissContextMenu() {
  document.querySelector(".context-menu")?.remove();
}

async function expandWorkflowRefsForStudio(workflow, seen = new Set(), source = currentWorkspaceFile) {
  if (!workflow || !Array.isArray(workflow.steps)) return workflow;
  const expanded = { ...workflow, steps: [] };
  for (const step of workflow.steps) {
    if (step.enabled === false) continue;
    if (step.name !== "workflow_ref") {
      expanded.steps.push(step);
      continue;
    }
    const ref = workflowRefFile(step);
    const resolved = resolveWorkspaceWorkflow(ref, source);
    const key = `${resolved.serviceName}/${resolved.fileName}`;
    if (seen.has(key)) throw new Error(`workflow_ref ciclo detectado em ${key}`);
    seen.add(key);
    const text = await WorkspaceFS.readFile(resolved.handle);
    const child = window.jsyaml.load(text);
    if (!child || !Array.isArray(child.steps)) {
      throw new Error(`workflow_ref ${key} nao possui steps validos.`);
    }
    const childSource = { handle: resolved.handle, serviceName: resolved.serviceName, fileName: resolved.fileName };
    const childExpanded = await expandWorkflowRefsForStudio(child, seen, childSource);
    const prefix = cleanWorkflowRefPrefix(step.params?.prefix || step.id || child.name || resolved.fileName);
    const childIDs = new Set(childExpanded.steps.map((childStep) => childStep.id).filter(Boolean));
    childExpanded.steps.forEach((childStep, index) => {
      const cloned = structuredClone(childStep);
      cloned.id = prefixedWorkflowStepID(prefix, cloned.id, cloned.name, index);
      cloned.params = rewriteWorkflowRefTargetsForStudio(cloned.params || {}, prefix, childIDs);
      cloned.__sourceWorkflow = key;
      expanded.steps.push(cloned);
    });
    seen.delete(key);
  }
  return expanded;
}

function workflowRefFile(step) {
  const params = step.params || {};
  const ref = params.file || params.path || params.workflow;
  if (!ref || typeof ref !== "string") throw new Error("workflow_ref precisa de params.file, params.path ou params.workflow.");
  return ref.trim();
}

function resolveWorkspaceWorkflow(ref, source) {
  if (!workspaceState.rootHandle) throw new Error("Abra um workspace para resolver workflow_ref.");
  const parts = ref.replace(/^\/+/, "").split("/").filter(Boolean);
  if (!parts.length) throw new Error("workflow_ref vazio.");
  let serviceName = source?.serviceName || currentWorkspaceFile.serviceName;
  let fileName = "";
  if (ref.startsWith("/") || parts.length > 1) {
    const rootRelative = ref.startsWith("/") || (parts[0] !== "." && parts[0] !== "..");
    const stack = rootRelative ? [] : [serviceName];
    parts.forEach((part) => {
      if (part === ".") return;
      if (part === "..") stack.pop();
      else stack.push(part);
    });
    fileName = stack.pop();
    serviceName = stack.pop();
    if (stack.length) throw new Error(`workflow_ref ${ref} deve apontar para um arquivo YAML dentro de um microservico.`);
  } else {
    fileName = parts[0];
  }
  if (!/\.(ya?ml)$/i.test(fileName)) fileName = `${fileName}.yaml`;
  const service = findService(serviceName);
  const file = service?.files.find((item) => item.name === fileName);
  if (!file) throw new Error(`workflow_ref nao encontrado: ${serviceName}/${fileName}`);
  return { serviceName, fileName, handle: file.handle };
}

function cleanWorkflowRefPrefix(value) {
  return String(value || "workflow")
    .trim()
    .toLowerCase()
    .replace(/\.(ya?ml)$/i, "")
    .replace(/[^a-z0-9._-]+/g, "-")
    .replace(/^-+|-+$/g, "") || "workflow";
}

function prefixedWorkflowStepID(prefix, id, name, index) {
  const cleaned = cleanWorkflowRefPrefix(id || "");
  if (cleaned) return `${prefix}.${cleaned}`;
  return `${prefix}.${String(index + 1).padStart(3, "0")}.${cleanWorkflowRefPrefix(name || "step")}`;
}

function rewriteWorkflowRefTargetsForStudio(params, prefix, childIDs) {
  if (!params || typeof params !== "object") return params;
  Object.entries(params).forEach(([key, value]) => {
    if (key === "to" && typeof value === "string" && childIDs.has(value)) {
      params[key] = `${prefix}.${value}`;
      return;
    }
    if (Array.isArray(value)) {
      value.forEach((item) => rewriteWorkflowRefTargetsForStudio(item, prefix, childIDs));
      return;
    }
    if (value && typeof value === "object") {
      rewriteWorkflowRefTargetsForStudio(value, prefix, childIDs);
    }
  });
  return params;
}

function parseWorkflow() {
  if (!window.jsyaml) {
    throw new Error("js-yaml nao foi carregado. Verifique acesso ao CDN.");
  }
  const value = window.jsyaml.load(els.workflow.value);
  if (!value || typeof value !== "object") {
    throw new Error("YAML deve declarar um objeto de workflow.");
  }
  return value;
}

function lintWorkflow() {
  const issues = [];
  let workflow = null;
  try {
    workflow = parseWorkflow();
    validateWorkflowShape(workflow, issues);
  } catch (err) {
    issues.push({ level: "error", message: err.message });
  }

  lastWorkflow = workflow;
  lastLint = issues;
  renderLint(issues);
  updateHeader(workflow);
  updateLineCount();
  syncLineNumbers();
  return issues;
}

function validateWorkflowShape(workflow, issues) {
  if (!stringValue(workflow.name)) issues.push(error("Campo obrigatorio ausente: name."));
  if (!Array.isArray(workflow.steps) || workflow.steps.length === 0) {
    issues.push(error("Campo steps deve ser uma lista com pelo menos uma etapa."));
    return;
  }
  if (!["", "stop", "continue", "skip"].includes(String(workflow.error_policy || "").toLowerCase())) {
    issues.push(error("error_policy deve ser stop, continue ou skip."));
  }
  if (!stringValue(workflow.message_id_path)) {
    issues.push(warn("message_id_path nao definido. O reprocessamento fica menos rastreavel."));
  }
  if (!stringValue(workflow.correlation_id_path)) {
    issues.push(warn("correlation_id_path nao definido. Observabilidade ponta a ponta fica limitada."));
  }

  workflow.steps.forEach((step, index) => validateStep(step, index, issues));
}

function validateStep(step, index, issues) {
  const label = `steps[${index}]`;
  if (!step || typeof step !== "object") {
    issues.push(error(`${label} deve ser objeto.`));
    return;
  }
  if (!handlers.has(step.name)) {
    issues.push(error(`${label}.name "${step.name || ""}" nao e um handler registrado.`));
  }
  const params = step.params || {};
  if (step.name === "validate" && !Array.isArray(params.required)) {
    issues.push(error(`${label} validate precisa de params.required como lista.`));
  }
  if (step.name === "condition" && !stringValue(params.field)) {
    issues.push(error(`${label} condition precisa de params.field.`));
  }
  if (step.name === "condition" && !("equals" in params) && !("not_equals" in params)) {
    issues.push(error(`${label} condition precisa de equals ou not_equals.`));
  }
  if (step.name === "assert") {
    const hasGroup = Array.isArray(params.all) || Array.isArray(params.any);
    const hasSingle = stringValue(params.field) || stringValue(params.exists);
    if (!hasGroup && !hasSingle) {
      issues.push(error(`${label} assert precisa de all, any ou uma condicao simples.`));
    }
  }
  if (step.name === "compute") {
    if (!stringValue(params.target)) issues.push(error(`${label} compute precisa de target.`));
    if (!params.value || typeof params.value !== "object") issues.push(error(`${label} compute precisa de value como objeto.`));
  }
  if (step.name === "cel") {
    if (!stringValue(params.expr) && !stringValue(params.expression)) {
      issues.push(error(`${label} cel precisa de params.expr.`));
    }
    const onFalse = String(params.on_false || "").toLowerCase();
    if (onFalse && !["error", "fail", "jump", "continue", "stop"].includes(onFalse)) {
      issues.push(error(`${label} cel possui on_false invalido. Use error, jump, continue ou stop.`));
    }
    if ((onFalse === "jump" || (!onFalse && stringValue(params.to))) && !stringValue(params.to)) {
      issues.push(error(`${label} cel precisa de to quando on_false for jump.`));
    }
  }
  if (step.name === "jump_if") {
    if (!stringValue(params.field)) issues.push(error(`${label} jump_if precisa de field.`));
    if (!stringValue(params.to)) issues.push(error(`${label} jump_if precisa de to com id ou name do step destino.`));
    if (!("equals" in params) && !("not_equals" in params)) {
      issues.push(error(`${label} jump_if precisa de equals ou not_equals.`));
    }
  }
  if (step.name === "graphql_enrich") {
    if (!stringValue(params.query)) issues.push(error(`${label} graphql_enrich precisa de query.`));
    if (!stringValue(params.target)) issues.push(error(`${label} graphql_enrich precisa de target.`));
    if (params.result_path && String(params.result_path).startsWith("data.")) {
      issues.push(warn(`${label} result_path parte de dentro de data; remova o prefixo "data.".`));
    }
  }
  if (step.name === "rest_call") {
    if (!stringValue(params.base_url)) issues.push(error(`${label} rest_call precisa de base_url.`));
    if (!stringValue(params.endpoint)) issues.push(error(`${label} rest_call precisa de endpoint.`));
    if (!stringValue(params.target)) issues.push(warn(`${label} rest_call sem target usa http_response.`));
  }
  if (step.name === "workflow_ref") {
    if (!stringValue(params.file) && !stringValue(params.path) && !stringValue(params.workflow)) {
      issues.push(error(`${label} workflow_ref precisa de params.file, params.path ou params.workflow.`));
    }
    if (!currentWorkspaceFile.handle) {
      issues.push(warn(`${label} workflow_ref e melhor testado com um arquivo aberto no workspace.`));
    }
  }
}

function renderLint(issues) {
  const errors = issues.filter((item) => item.level === "error");
  const warnings = issues.filter((item) => item.level === "warn");
  els.lintBadge.className = `badge ${errors.length ? "badge-error" : warnings.length ? "badge-warn" : "badge-ok"}`;
  els.lintBadge.textContent = errors.length ? `${errors.length} erro(s)` : warnings.length ? `${warnings.length} aviso(s)` : "Valido";

  if (!issues.length) {
    els.lint.innerHTML = `<span class="badge badge-ok">Workflow valido</span>`;
    return;
  }
  els.lint.innerHTML = issues.map((issue) => {
    const cls = issue.level === "error" ? "badge-error" : "badge-warn";
    return `<div><span class="badge ${cls}">${issue.level}</span> ${escapeHtml(issue.message)}</div>`;
  }).join("");
}

function updateHeader(workflow) {
  els.title.textContent = workflow && workflow.name ? workflow.name : "Workflow";
}

function updateLineCount() {
  const lines = els.workflow.value.split("\n").length;
  els.lineCount.textContent = `${lines} linhas`;
}

function syncLineNumbers() {
  const count = els.workflow.value.split("\n").length;
  if (Number(els.lineNumbers.dataset.count || 0) !== count) {
    els.lineNumbers.dataset.count = String(count);
    els.lineNumbers.textContent = Array.from({ length: count }, (_, index) => index + 1).join("\n");
  }
  els.lineNumbers.scrollTop = els.workflow.scrollTop;
  renderWorkflowHighlight();
}

function renderWorkflowHighlight() {
  if (!els.highlight) return;
  els.highlight.scrollTop = els.workflow.scrollTop;
  els.highlight.scrollLeft = els.workflow.scrollLeft;
  els.highlight.innerHTML = highlightWorkflowYaml(els.workflow.value);
}

function highlightWorkflowYaml(value) {
  let inHeader = true;
  let inSteps = false;
  let stepIndex = -1;
  let currentStepHighlighted = false;
  return String(value || "").split("\n").map((line) => {
    const lineClasses = ["yaml-line"];
    const isComment = /^\s*#/.test(line);
    const isBlank = line.trim() === "";

    if (inSteps && !isComment && !isBlank) {
      const stepStart = line.match(/^(\s*-\s*)(id|name)(\s*:\s*)/);
      if (stepStart) {
        stepIndex += 1;
        currentStepHighlighted = stepIndex % 2 === 0;
      }
      if (currentStepHighlighted) lineClasses.push("yaml-step-band");
    }

    if (isComment) return `<span class="${lineClasses.join(" ")}"><span class="yaml-comment">${escapeHtml(line)}</span></span>`;
    if (/^\s*steps\s*:/.test(line)) {
      inHeader = false;
      inSteps = true;
      return `<span class="${lineClasses.join(" ")}"><span class="yaml-step-divider">${escapeHtml(line)}</span></span>`;
    }
    if (inHeader) {
      const match = line.match(/^(\s*)([A-Za-z0-9_.-]+)(\s*:\s*)(.*)$/);
      if (match) {
        return `<span class="${lineClasses.join(" ")}">${escapeHtml(match[1])}<span class="yaml-header-key">${escapeHtml(match[2])}</span>${escapeHtml(match[3])}<span class="yaml-text">${escapeHtml(match[4])}</span></span>`;
      }
    }
    const stepMatch = line.match(/^(\s*-\s*)(id|name)(\s*:\s*)(.*)$/);
    if (stepMatch) {
      return `<span class="${lineClasses.join(" ")}">${escapeHtml(stepMatch[1])}<span class="yaml-step-key">${escapeHtml(stepMatch[2])}</span>${escapeHtml(stepMatch[3])}<span class="yaml-text">${escapeHtml(stepMatch[4])}</span></span>`;
    }
    return `<span class="${lineClasses.join(" ")}"><span class="yaml-text">${escapeHtml(line)}</span></span>`;
  }).join("");
}

function validatePayload() {
  try {
    JSON.parse(els.payload.value);
    return true;
  } catch (err) {
    return false;
  }
}

async function runLocalSimulation() {
  clearLogs();
  const issues = lintWorkflow();
  if (issues.some((item) => item.level === "error")) {
    addLog("error", "Lint bloqueou a execucao", "Corrija os erros do workflow antes de executar.");
    return;
  }

  let payload;
  try {
    payload = JSON.parse(els.payload.value);
  } catch (err) {
    addLog("error", "Payload JSON invalido", err.message);
    return;
  }

  let workflow;
  try {
    workflow = await expandWorkflowRefsForStudio(lastWorkflow);
  } catch (err) {
    addLog("error", "Falha ao compor workflows", err.message);
    return;
  }
  activeRuntimeWorkflow = workflow;
  const state = {
    payload: structuredClone(payload),
    history: [],
    cursor: 0,
    stopped: false,
    errors: [],
  };
  addLog("info", "Workflow iniciado", `${workflow.name || "workflow"} com ${workflow.steps.length} etapa(s).`, {
    message_id: getPath(state.payload, workflow.message_id_path),
    correlation_id: getPath(state.payload, workflow.correlation_id_path),
  });

  for (let i = 0; i < workflow.steps.length; i += 1) {
    state.cursor = i;
    activeExecutionStepIndex = i;
    const step = workflow.steps[i];
    startStepGroup(i, step, workflow.steps.length);
    const started = performance.now();
    try {
      addLog("info", `Executando ${step.name}`, `Etapa ${i + 1} de ${workflow.steps.length}.`, step.params || {});
      const proceed = await executeStep(step, state);
      const duration = Math.round(performance.now() - started);
      state.history.push({ step: step.name, duration_ms: duration, skipped: !proceed });
      addLog("ok", `Etapa ${step.name} concluida`, `${duration} ms`, snapshot(state));
      if (Number.isInteger(state.__jumpToIndex)) {
        const jumpTo = state.__jumpToIndex;
        delete state.__jumpToIndex;
        addLog("warn", "Salto aplicado", `Proxima etapa: ${jumpTo + 1}.`, { cursor: jumpTo });
        i = jumpTo - 1;
        continue;
      }
      if (!proceed) {
        state.stopped = true;
        addLog("warn", "Workflow interrompido por gate", `Cursor parado apos a etapa ${i}.`);
        break;
      }
    } catch (err) {
      state.errors.push({ step: step.name, error: err.message, cursor: i });
      addLog("error", `Falha em ${step.name}`, err.message, snapshot(state));
      if (String(workflow.error_policy || "stop").toLowerCase() === "stop") {
        state.cursor = i;
        addLog("warn", "Snapshot pronto para reprocessamento", `Reprocessamento deve retomar do cursor ${i}.`);
        break;
      }
    } finally {
      activeExecutionStepIndex = null;
    }
  }

  const status = state.errors.length ? "com falha" : state.stopped ? "interrompido" : "concluido";
  els.summary.textContent = `${workflow.name || "Workflow"} ${status}: ${state.history.length} etapa(s), ${state.errors.length} erro(s)`;
  addLog(state.errors.length ? "error" : "ok", `Workflow ${status}`, "Payload final da simulacao.", snapshot(state));
  activeRuntimeWorkflow = null;
}

async function executeStep(step, state) {
  const params = step.params || {};
  switch (step.name) {
    case "validate":
      return runValidate(params, state);
    case "condition":
      return runCondition(params, state);
    case "assert":
      return runAssert(params, state);
    case "compute":
      return runCompute(params, state);
    case "cel":
      return runCEL(params, state);
    case "enrich":
      return runEnrich(params, state);
    case "transform":
      return runTransform(params, state);
    case "audit":
      return runAudit(params, state);
    case "notify":
      return runNotify(params, state);
    case "graphql_enrich":
      return runGraphQLEnrich(params, state);
    case "jump_if":
      return runJumpIf(params, state);
    case "rest_call":
      return runRestCall(params, state);
    default:
      throw new Error(`Handler nao registrado: ${step.name}`);
  }
}

function runValidate(params, state) {
  const missing = (params.required || []).filter((path) => {
    const value = getPath(state.payload, path);
    return value === undefined || value === null || value === "";
  });
  if (missing.length) {
    state.payload.validation_error = `missing fields: ${missing.join(", ")}`;
    if (params.stop_on_failure !== false) throw new Error(state.payload.validation_error);
  } else {
    state.payload.validation_passed = true;
  }
  return true;
}

function runCondition(params, state) {
  const current = getPath(state.payload, params.field);
  if ("equals" in params && String(current) !== String(params.equals)) {
    state.payload.gate_stopped = true;
    return false;
  }
  if ("not_equals" in params && String(current) === String(params.not_equals)) {
    state.payload.gate_stopped = true;
    return false;
  }
  return true;
}

function runAssert(params, state) {
  const result = evaluateAssertConfig(params, state);
  state.payload.assert_passed = result.matched;
  if (!result.matched) {
    state.payload.assert_failures = result.failures;
    throw new Error(`${params.message || "assertion failed"}${result.failures.length ? `: ${result.failures.join("; ")}` : ""}`);
  }
  return true;
}

function runCompute(params, state) {
  if (!params.target) throw new Error("compute: target is required");
  const value = evaluateValueConfig(params.value || {}, state);
  state.payload[params.target] = value;
  return true;
}

function runCEL(params, state) {
  const expression = params.expr || params.expression;
  if (!expression) throw new Error("cel: expr is required");
  const matched = evaluateCELExpression(expression, state.payload);
  const target = params.target || "cel_passed";
  setPath(state.payload, target, matched);
  state.payload.cel_passed = matched;
  addLog(matched ? "ok" : "warn", "CEL avaliado", expression, { matched, target });
  if (matched) return true;

  const onFalse = String(params.on_false || (params.to ? "jump" : "error")).toLowerCase();
  if (onFalse === "jump") {
    if (!params.to) throw new Error("cel: to is required when on_false is jump");
    const index = findWorkflowStepIndex(params.to);
    if (index < 0) throw new Error(`cel: target step "${params.to}" not found`);
    if (index <= state.cursor) throw new Error(`cel: target step "${params.to}" must be after current step`);
    state.payload.jumped_to = params.to;
    state.payload.jumped_to_cursor = index;
    state.__jumpToIndex = index;
    return true;
  }
  if (onFalse === "continue") return true;
  if (onFalse === "stop") {
    state.payload.cel_stopped = true;
    return false;
  }
  if (onFalse === "error" || onFalse === "fail") {
    throw new Error(params.message || `cel: expression evaluated to false: ${expression}`);
  }
  throw new Error(`cel: unsupported on_false value "${onFalse}"`);
}

function runJumpIf(params, state) {
  const matched = evaluateConditionConfig(params, state);
  state.payload.jump_if_matched = matched;
  if (!matched) return true;
  const index = findWorkflowStepIndex(params.to);
  if (index < 0) throw new Error(`jump_if: target step "${params.to}" not found`);
  if (index <= state.cursor) throw new Error(`jump_if: target step "${params.to}" must be after current step`);
  state.payload.jumped_to = params.to;
  state.payload.jumped_to_cursor = index;
  state.__jumpToIndex = index;
  return true;
}

function evaluateCELExpression(expression, payload) {
  const translated = String(expression)
    .replace(/\bhas\s*\(([^)]+)\)/g, (_, path) => `celHas(payload, ${JSON.stringify(path.trim())})`)
    .replace(/\bsize\s*\(/g, "celSize(");
  try {
    return Boolean(Function("payload", "celSize", "celHas", `with (payload) { return (${translated}); }`)(payload, celSize, celHas));
  } catch (err) {
    throw new Error(`cel: nao foi possivel avaliar a expressao no simulador local: ${err.message}`);
  }
}

function celSize(value) {
  if (Array.isArray(value) || typeof value === "string") return value.length;
  if (value && typeof value === "object") return Object.keys(value).length;
  return 0;
}

function celHas(payload, path) {
  const normalized = String(path || "").replace(/^payload\./, "");
  return getPath(payload, normalized) !== undefined;
}

function runEnrich(params, state) {
  const data = interpolateAny(params.data || {}, state.payload);
  const prefix = params.prefix || "";
  Object.entries(data).forEach(([key, value]) => {
    state.payload[`${prefix}${key}`] = value;
  });
  state.payload[`${prefix}enriched_at`] = new Date().toISOString();
  return true;
}

function runTransform(params, state) {
  const field = params.field;
  const target = params.target || field;
  const value = getPath(state.payload, field);
  if (value === undefined) throw new Error(`transform: field "${field}" not found`);
  let text = String(value);
  const operation = params.operation || "";
  if (operation === "uppercase") text = text.toUpperCase();
  else if (operation === "lowercase") text = text.toLowerCase();
  else if (operation === "trim") text = text.trim();
  else if (operation.startsWith("prefix:")) text = operation.slice("prefix:".length) + text;
  else if (operation.startsWith("suffix:")) text += operation.slice("suffix:".length);
  else throw new Error(`transform: unknown operation "${operation}"`);
  setPath(state.payload, target, text);
  return true;
}

function runAudit(params, state) {
  const fields = {};
  (params.fields || []).forEach((field) => {
    fields[field] = getPath(state.payload, field);
  });
  addLog("info", `Audit ${params.event || "audit"}`, "Campos auditados.", fields);
  return true;
}

function runNotify(params, state) {
  const body = interpolateString(params.template || "", state.payload);
  state.payload.notification_sent = true;
  state.payload.notification_channel = params.channel || "log";
  addLog("info", `Notify ${state.payload.notification_channel}`, body, {
    recipient: params.recipient,
  });
  return true;
}

async function runGraphQLEnrich(params, state) {
  if (!els.integrations.checked) {
    const target = params.target || "external_data";
    state.payload[target] = mockGraphQLValue(params);
    state.payload[`${target}_enriched_at`] = new Date().toISOString();
    addLog("warn", "GraphQL simulado", "Ative integracoes reais para chamar o endpoint.", state.payload[target]);
    return true;
  }

  const response = await fetch(params.endpoint || els.graphqlEndpoint.value, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      query: params.query,
      variables: interpolateAny(params.variables || {}, state.payload),
    }),
  });
  if (!response.ok) throw new Error(`graphql_enrich: HTTP ${response.status}`);
  const result = await response.json();
  if (result.errors && result.errors.length && params.required !== false) {
    throw new Error(`graphql_enrich: ${JSON.stringify(result.errors)}`);
  }
  const target = params.target || "external_data";
  const resultRoot = result && typeof result === "object" && "data" in result ? result.data : result;
  const selected = params.result_path ? getPath(resultRoot || {}, params.result_path) : resultRoot;
  if (selected === undefined && params.required !== false) throw new Error(`graphql_enrich: result_path ${params.result_path} not found`);
  state.payload[target] = selected;
  state.payload[`${target}_enriched_at`] = new Date().toISOString();
  return true;
}

async function runRestCall(params, state) {
  if (!els.integrations.checked) {
    const target = params.target || "http_response";
    state.payload[target] = {
      simulated: true,
      method: params.method || "GET",
      endpoint: interpolateString(params.endpoint || "", state.payload),
      status: "SIMULATED",
    };
    state.payload[`${target}_called_at`] = new Date().toISOString();
    addLog("warn", "REST simulado", "Ative integracoes reais para chamar a API.", state.payload[target]);
    return true;
  }

  const baseUrl = expandEnv(params.base_url || els.externalApiUrl.value);
  const method = String(params.method || "GET").toUpperCase();
  const endpoint = interpolateString(params.endpoint || "", state.payload);
  const options = {
    method,
    headers: { Accept: "application/json" },
  };
  if (!["GET", "HEAD"].includes(method)) {
    options.headers["Content-Type"] = "application/json";
    options.body = JSON.stringify(interpolateAny(params.body || {}, state.payload));
  }
  Object.entries(params.headers || {}).forEach(([key, value]) => {
    options.headers[key] = interpolateString(String(value), state.payload);
  });
  const response = await fetch(`${baseUrl.replace(/\/$/, "")}${endpoint}`, options);
  if (!response.ok) throw new Error(`rest_call: HTTP ${response.status}`);
  const result = await response.json();
  const target = params.target || "http_response";
  state.payload[target] = params.result_path ? getPath(result, params.result_path) : result;
  state.payload[`${target}_called_at`] = new Date().toISOString();
  return true;
}

async function sendToRestEndpoint() {
  clearLogs();
  try {
    const payload = JSON.parse(els.payload.value);
    addLog("info", "Enviando para REST", els.workflowEndpoint.value, payload);
    const response = await fetch(els.workflowEndpoint.value, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    const body = await response.json().catch(() => ({}));
    addLog(response.ok ? "ok" : "error", `REST retornou ${response.status}`, response.statusText, body);
  } catch (err) {
    addLog("error", "Falha ao enviar REST", err.message);
  }
}

function mockGraphQLValue(params) {
  const path = params.result_path || "";
  if (path.includes("order")) {
    return {
      pedido_id: "PED-9988",
      cliente_id: "CLI-1001",
      status: "PAGO",
      valor_total: 150,
      endereco_entrega: "Rua Exemplo, 123",
      itens: [{ produto_id: "SKU-1", quantidade: 1 }],
    };
  }
  if (path.includes("dataSources")) {
    return {
      custodias: [{ operacaoId: "2699999999", situacaoOperacao: "ATIVA", siglaCustodia: "SF", saldoDevedor: 48000 }],
      saldos: [{ saldo: { saldo_liquido_operacao: 47500 } }],
    };
  }
  return { simulated: true };
}

function formatWorkflow() {
  try {
    const workflow = parseWorkflow();
    els.workflow.value = window.jsyaml.dump(workflow, { lineWidth: 140, noRefs: true });
    markWorkflowDirty();
    lintWorkflow();
    scheduleStudioSave();
  } catch (err) {
    addLog("error", "Nao foi possivel organizar YAML", err.message);
  }
}

function clearLogs() {
  els.timeline.classList.remove("timeline--docs");
  els.timeline.innerHTML = "";
  document.querySelector("#workspace-mode-label").textContent = "Execucao";
  document.querySelector("#result-panel-title").textContent = "Logs da execucao";
  document.querySelector("#result-panel-meta").textContent = "Timeline";
  els.summary.textContent = "Nenhuma execucao";
  activeLogEntry = null;
  activeStepGroup = null;
  stepGroups = new Map();
}

function addLog(level, title, copy, data, options = {}) {
  const entry = document.createElement("article");
  entry.className = "log-entry";
  const stepIndex = options.stepIndex ?? activeExecutionStepIndex;
  if (Number.isInteger(stepIndex)) {
    entry.classList.add("log-entry--step");
    entry.dataset.stepIndex = String(stepIndex);
    entry.title = "Clique para focar esta etapa no YAML";
    entry.addEventListener("click", () => focusStepFromLog(entry, stepIndex));
  }
  const cls = level === "error" ? "level-error" : level === "warn" ? "level-warn" : level === "ok" ? "level-ok" : "level-info";
  entry.innerHTML = `
    <div class="log-meta">
      <span class="log-time">${new Date().toLocaleTimeString()}</span>
      <span class="log-level ${cls}">${level}</span>
    </div>
    <div class="log-body">
      <p class="log-title">${escapeHtml(title)}</p>
      <p class="log-copy">${escapeHtml(copy || "")}</p>
      ${data === undefined ? "" : `<pre class="log-json">${escapeHtml(JSON.stringify(data, null, 2))}</pre>`}
    </div>
  `;
  const container = Number.isInteger(stepIndex) ? ensureStepGroup(stepIndex).querySelector(".phase-logs") : els.timeline;
  container.appendChild(entry);
  els.timeline.scrollTop = els.timeline.scrollHeight;
}

function startStepGroup(stepIndex, step, totalSteps) {
  const group = ensureStepGroup(stepIndex, step, totalSteps);
  setActiveStepGroup(group);
}

function ensureStepGroup(stepIndex, step = null, totalSteps = null) {
  if (stepGroups.has(stepIndex)) return stepGroups.get(stepIndex);
  const group = document.createElement("section");
  group.className = "phase-group";
  group.dataset.stepIndex = String(stepIndex);
  const title = step?.name || `step ${stepIndex + 1}`;
  group.innerHTML = `
    <button class="phase-head" type="button" title="Clique para focar esta fase no YAML">
      <span class="phase-index">${stepIndex + 1}</span>
      <span class="phase-title">${escapeHtml(title)}</span>
      <span class="phase-copy">${totalSteps ? `Etapa ${stepIndex + 1} de ${totalSteps}` : "Etapa"}</span>
    </button>
    <div class="phase-logs"></div>
  `;
  group.querySelector(".phase-head").addEventListener("click", () => {
    setActiveStepGroup(group);
    focusWorkflowStep(stepIndex);
  });
  stepGroups.set(stepIndex, group);
  els.timeline.appendChild(group);
  return group;
}

function setActiveStepGroup(group) {
  if (activeStepGroup) activeStepGroup.classList.remove("phase-group--active");
  activeStepGroup = group;
  activeStepGroup.classList.add("phase-group--active");
}

function focusStepFromLog(entry, stepIndex) {
  if (activeLogEntry) activeLogEntry.classList.remove("log-entry--active");
  activeLogEntry = entry;
  activeLogEntry.classList.add("log-entry--active");
  const group = stepGroups.get(stepIndex);
  if (group) {
    if (activeStepGroup) activeStepGroup.classList.remove("phase-group--active");
    activeStepGroup = group;
    activeStepGroup.classList.add("phase-group--active");
  }
  focusWorkflowStep(stepIndex);
}

function focusWorkflowStep(stepIndex) {
  const range = findStepRange(stepIndex);
  if (!range) {
    addLog("warn", "Nao encontrei a etapa no YAML", `step index ${stepIndex}`);
    return;
  }
  els.workflow.focus();
  els.workflow.selectionStart = range.start;
  els.workflow.selectionEnd = range.end;
  const lineHeight = getEditorLineHeight();
  const targetTop = Math.max(0, (range.startLine - 2) * lineHeight);
  els.workflow.scrollTop = targetTop;
  syncLineNumbers();
}

function findStepRange(stepIndex) {
  const text = els.workflow.value;
  const lines = text.split("\n");
  const stepsLine = lines.findIndex((line) => /^steps\s*:\s*(?:#.*)?$/.test(line.trim()));
  if (stepsLine < 0) return null;

  const positions = lineStartPositions(lines);
  const blocks = [];
  let inSteps = false;
  let stepsIndent = 0;

  for (let i = stepsLine; i < lines.length; i += 1) {
    const raw = lines[i];
    const trimmed = raw.trim();
    if (!trimmed || trimmed.startsWith("#")) continue;
    const indent = raw.match(/^\s*/)[0].length;
    if (i === stepsLine) {
      inSteps = true;
      stepsIndent = indent;
      continue;
    }
    if (inSteps && indent <= stepsIndent && !trimmed.startsWith("- ")) break;
    if (inSteps && /^-\s+(?:id|name)\s*:/.test(trimmed)) {
      blocks.push({ line: i, start: positions[i] });
    }
  }

  const block = blocks[stepIndex];
  if (!block) return null;
  const next = blocks[stepIndex + 1];
  const end = next ? Math.max(block.start, next.start - 1) : text.length;
  return {
    start: block.start,
    end,
    startLine: block.line,
  };
}

function lineStartPositions(lines) {
  const positions = [];
  let cursor = 0;
  lines.forEach((line) => {
    positions.push(cursor);
    cursor += line.length + 1;
  });
  return positions;
}

function getEditorLineHeight() {
  const value = Number.parseFloat(getComputedStyle(els.workflow).lineHeight);
  return Number.isFinite(value) ? value : 19;
}

function snapshot(state) {
  const workflow = activeRuntimeWorkflow || lastWorkflow;
  return {
    cursor: state.cursor,
    remaining_steps: Math.max(0, (workflow?.steps?.length || 0) - state.cursor - 1),
    payload: state.payload,
    history: state.history,
    errors: state.errors,
  };
}

function getPath(obj, path) {
  if (!path) return undefined;
  return String(path).split(".").reduce((acc, key) => {
    if (Array.isArray(acc) && /^\d+$/.test(key)) return acc[Number(key)];
    if (acc && typeof acc === "object" && key in acc) return acc[key];
    return undefined;
  }, obj);
}

function findWorkflowStepIndex(ref) {
  const steps = (activeRuntimeWorkflow || lastWorkflow)?.steps || [];
  let index = steps.findIndex((step) => step.id === ref);
  if (index >= 0) return index;
  return steps.findIndex((step) => step.name === ref);
}

function evaluateValueConfig(config, state) {
  if ("literal" in config) return config.literal;
  if (typeof config.count === "string") {
    const value = getPath(state.payload, config.count);
    if (!isCountable(value)) throw new Error(`compute: field "${config.count}" is not countable`);
    return value.length ?? Object.keys(value).length;
  }
  if ("exists" in config) return getPath(state.payload, config.exists) !== undefined;
  if (config.field && Object.keys(config).length === 1) {
    const value = getPath(state.payload, config.field);
    if (value === undefined) throw new Error(`compute: field "${config.field}" not found`);
    return value;
  }
  return evaluateConditionConfig(config, state);
}

function evaluateAssertConfig(config, state) {
  if (Array.isArray(config.all)) {
    const failures = [];
    config.all.forEach((condition, index) => {
      try {
        if (!evaluateConditionConfig(condition, state)) failures.push(`all[${index}]: condition not satisfied`);
      } catch (err) {
        failures.push(`all[${index}]: ${err.message}`);
      }
    });
    return { matched: failures.length === 0, failures };
  }
  if (Array.isArray(config.any)) {
    const failures = [];
    for (let index = 0; index < config.any.length; index += 1) {
      try {
        if (evaluateConditionConfig(config.any[index], state)) return { matched: true, failures: [] };
        failures.push(`any[${index}]: condition not satisfied`);
      } catch (err) {
        failures.push(`any[${index}]: ${err.message}`);
      }
    }
    return { matched: false, failures };
  }
  return { matched: evaluateConditionConfig(config, state), failures: ["condition not satisfied"] };
}

function evaluateConditionConfig(config, state) {
  if (typeof config.exists === "string") return getPath(state.payload, config.exists) !== undefined;
  const value = getPath(state.payload, config.field);
  if (value === undefined) throw new Error(`field "${config.field}" not found`);
  if ("equals" in config) return String(value) === String(config.equals);
  if ("not_equals" in config) return String(value) !== String(config.not_equals);
  if ("less_than" in config) return Number(value) < Number(config.less_than);
  if ("less_than_or_equal" in config) return Number(value) <= Number(config.less_than_or_equal);
  if ("greater_than" in config) return Number(value) > Number(config.greater_than);
  if ("greater_than_or_equal" in config) return Number(value) >= Number(config.greater_than_or_equal);
  if ("min_items" in config) {
    if (!isCountable(value)) throw new Error(`field "${config.field}" is not countable`);
    return value.length >= Number(config.min_items);
  }
  if ("max_items" in config) {
    if (!isCountable(value)) throw new Error(`field "${config.field}" is not countable`);
    return value.length <= Number(config.max_items);
  }
  throw new Error("no supported comparison configured");
}

function isCountable(value) {
  return Array.isArray(value) || typeof value === "string" || Boolean(value && typeof value === "object");
}

function setPath(obj, path, value) {
  const parts = String(path).split(".");
  let current = obj;
  parts.slice(0, -1).forEach((part) => {
    if (!current[part] || typeof current[part] !== "object") current[part] = {};
    current = current[part];
  });
  current[parts[parts.length - 1]] = value;
}

function interpolateAny(value, payload) {
  if (typeof value === "string") return interpolateString(value, payload);
  if (Array.isArray(value)) return value.map((item) => interpolateAny(item, payload));
  if (value && typeof value === "object") {
    return Object.fromEntries(Object.entries(value).map(([key, item]) => [key, interpolateAny(item, payload)]));
  }
  return value;
}

function interpolateString(text, payload) {
  const exact = text.match(/^\{([^{}]+)\}$/);
  if (exact) {
    const value = getPath(payload, exact[1]);
    return value === undefined ? text : value;
  }
  return text.replace(/\{([^{}]+)\}/g, (_, path) => {
    const value = getPath(payload, path);
    return value === undefined ? "" : String(value);
  });
}

function expandEnv(value) {
  return String(value).replace(/\$\{([^}:]+):-([^}]+)\}/g, (_, key, fallback) => {
    return localStorage.getItem(key) || fallback;
  });
}

function error(message) {
  return { level: "error", message };
}

function warn(message) {
  return { level: "warn", message };
}

function stringValue(value) {
  return typeof value === "string" && value.trim() !== "";
}

function escapeHtml(value) {
  return String(value)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}

boot();
