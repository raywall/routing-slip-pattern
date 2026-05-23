const handlers = new Set([
  "validate",
  "condition",
  "assert",
  "compute",
  "enrich",
  "transform",
  "notify",
  "audit",
  "graphql_enrich",
  "jump_if",
  "rest_call",
]);

const STUDIO_DB = {
  name: "routing-slip-studio",
  store: "state",
  key: "current",
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
};

let lastWorkflow = null;
let lastLint = [];
let activeExecutionStepIndex = null;
let activeLogEntry = null;
let activeStepGroup = null;
let stepGroups = new Map();
let saveTimer = null;

async function boot() {
  const restored = await restoreStudioState();
  if (!restored) loadExample("payment", { persist: false });
  bindEvents();
  lintWorkflow();
}

function bindEvents() {
  els.workflow.addEventListener("input", () => {
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
  document.querySelectorAll("[data-toggle-panel]").forEach((button) => {
    button.addEventListener("click", () => togglePanel(button.dataset.togglePanel));
  });
  bindSidebarResize();
}

function loadExample(key, options = {}) {
  const example = examples[key] || examples.payment;
  els.example.value = key;
  els.workflow.value = example.workflow;
  els.payload.value = JSON.stringify(example.payload, null, 2);
  clearLogs();
  lintWorkflow();
  if (options.persist !== false) scheduleStudioSave();
}

function handleEditorKeydown(event) {
  if ((event.metaKey || event.ctrlKey) && event.key === "/") {
    event.preventDefault();
    toggleYamlComment();
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
  const editableLines = lines.filter((line) => line.trim() !== "");
  const shouldUncomment = editableLines.length > 0 && editableLines.every((line) => /^(\s*)# ?/.test(line));
  const changed = lines.map((line) => {
    if (line.trim() === "") return line;
    if (shouldUncomment) return line.replace(/^(\s*)# ?/, "$1");
    const indent = line.match(/^\s*/)[0];
    return `${indent}# ${line.slice(indent.length)}`;
  }).join("\n");

  editor.value = value.slice(0, lineStart) + changed + value.slice(blockEnd);
  editor.selectionStart = lineStart;
  editor.selectionEnd = lineStart + changed.length;
  lintWorkflow();
}

function togglePanel(id) {
  const panel = document.getElementById(id);
  if (!panel) return;
  panel.classList.toggle("collapsed");
  localStorage.setItem(`routing-slip-studio:${id}:collapsed`, panel.classList.contains("collapsed") ? "1" : "0");
}

function restorePanelState() {
  ["config-panel", "payload-panel"].forEach((id) => {
    const panel = document.getElementById(id);
    if (panel && localStorage.getItem(`routing-slip-studio:${id}:collapsed`) === "1") {
      panel.classList.add("collapsed");
    }
  });
  const width = localStorage.getItem("routing-slip-studio:sidebar-width");
  if (width) els.sidebar.style.width = `${width}px`;
}

function bindSidebarResize() {
  restorePanelState();
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

  const workflow = lastWorkflow;
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
    lintWorkflow();
  } catch (err) {
    addLog("error", "Nao foi possivel organizar YAML", err.message);
  }
}

function clearLogs() {
  els.timeline.innerHTML = "";
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
  return {
    cursor: state.cursor,
    remaining_steps: Math.max(0, (lastWorkflow?.steps?.length || 0) - state.cursor - 1),
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
  const steps = lastWorkflow?.steps || [];
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
