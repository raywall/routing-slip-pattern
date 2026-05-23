async function runLocalSimulation(options = {}) {
  clearLogs();
  const isReprocess = Boolean(options.reprocess);
  if (isReprocess && !lastExecutionSnapshot) {
    addLog("error", "Reprocessamento indisponivel", "Execute o workflow antes de reprocessar.");
    return;
  }
  const issues = lintWorkflow();
  if (issues.some((item) => item.level === "error")) {
    addLog("error", "Lint bloqueou a execucao", "Corrija os erros do workflow antes de executar.");
    return;
  }

  let payload;
  try {
    payload = isReprocess ? structuredClone(lastExecutionSnapshot.payload) : JSON.parse(els.payload.value);
  } catch (err) {
    addLog("error", "Payload JSON invalido", err.message);
    return;
  }

  let workflow;
  try {
    workflow = isReprocess ? structuredClone(lastExecutionSnapshot.workflow) : await expandWorkflowRefsForStudio(lastWorkflow);
  } catch (err) {
    addLog("error", "Falha ao compor workflows", err.message);
    return;
  }
  activeRuntimeWorkflow = workflow;
  const startCursor = isReprocess ? Math.min(lastExecutionSnapshot.resumeCursor || 0, workflow.steps.length) : 0;
  const state = {
    payload: structuredClone(payload),
    history: [],
    cursor: startCursor,
    stopped: false,
    errors: [],
    reprocess: isReprocess ? {
      previousDurationMs: lastExecutionSnapshot.durationMs,
      previousCursor: lastExecutionSnapshot.resumeCursor,
      previousStatus: lastExecutionSnapshot.status,
    } : null,
    metrics: {
      startedAt: performance.now(),
      integrations: [],
    },
  };
  addLog("info", isReprocess ? "Reprocessamento iniciado" : "Workflow iniciado", `${workflow.name || "workflow"} com ${workflow.steps.length} etapa(s).`, {
    message_id: getPath(state.payload, workflow.message_id_path),
    correlation_id: getPath(state.payload, workflow.correlation_id_path),
    cursor_inicial: startCursor,
    execucao_anterior_ms: state.reprocess?.previousDurationMs,
  });

  if (isReprocess && startCursor > 0) {
    addLog("info", "Etapas anteriores preservadas", `${startCursor} etapa(s) ja processada(s) antes do reprocessamento.`, {
      cursor_retomada: startCursor,
      etapas_preservadas: workflow.steps.slice(0, startCursor).map((step, index) => ({ index, id: step.id, name: step.name })),
    });
  }

  for (let i = startCursor; i < workflow.steps.length; i += 1) {
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
  const totalDuration = Math.round(performance.now() - state.metrics.startedAt);
  state.metrics.total_duration_ms = totalDuration;
  const resumeCursor = resolveResumeCursor(state, workflow);
  els.summary.textContent = `${workflow.name || "Workflow"} ${status}: ${state.history.length} etapa(s), ${state.errors.length} erro(s), ${formatDuration(totalDuration)}`;
  addLog(state.errors.length ? "error" : "ok", `Workflow ${status}`, "Payload final da simulacao.", snapshot(state));
  renderExecutionSummary(workflow, state, status);
  lastExecutionSnapshot = {
    workflow: structuredClone(workflow),
    payload: structuredClone(state.payload),
    resumeCursor,
    durationMs: totalDuration,
    status,
    executedAt: new Date().toISOString(),
  };
  els.reprocess.disabled = false;
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
    case "filter_array":
      return runFilterArray(params, state);
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

function runFilterArray(params, state) {
  const source = params.source || params.field;
  if (!source) throw new Error("filter_array: source is required");
  const target = params.target || source;
  const items = getPath(state.payload, source);
  if (!Array.isArray(items)) throw new Error(`filter_array: source "${source}" must be an array`);

  const filtered = items.filter((item, index) => evaluateFilterArrayItem(params, state, item, index));
  setPath(state.payload, target, filtered);
  state.payload[`${target}_filtered_count`] = filtered.length;
  state.payload[`${target}_removed_count`] = items.length - filtered.length;
  addLog("info", "Array filtrado", `${items.length - filtered.length} item(ns) removido(s).`, {
    source,
    target,
    original_count: items.length,
    filtered_count: filtered.length,
  });
  return true;
}

function evaluateFilterArrayItem(params, state, item, index) {
  const payload = { ...state.payload, item, index };
  if (params.expr || params.expression) {
    return evaluateCELExpression(params.expr || params.expression, payload);
  }
  if (!params.where) throw new Error("filter_array: where or expr is required");
  const itemState = { ...state, payload };
  if (Array.isArray(params.where)) {
    return params.where.every((condition) => evaluateConditionConfig(condition, itemState));
  }
  return evaluateAssertConfig(params.where, itemState).matched;
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
  recordIntegration(state, {
    type: "notify",
    mode: "simulated",
    target: params.channel || "log",
    status: "ok",
  });
  state.payload.notification_sent = true;
  state.payload.notification_channel = params.channel || "log";
  addLog("info", `Notify ${state.payload.notification_channel}`, body, {
    recipient: params.recipient,
  });
  return true;
}

async function runGraphQLEnrich(params, state) {
  const endpoint = params.endpoint || els.graphqlEndpoint.value;
  const integration = recordIntegration(state, {
    type: "graphql",
    mode: els.integrations.checked ? "real" : "simulated",
    target: params.target || "external_data",
    endpoint,
    status: "attempted",
  });
  if (!els.integrations.checked) {
    const target = params.target || "external_data";
    state.payload[target] = mockGraphQLValue(params);
    state.payload[`${target}_enriched_at`] = new Date().toISOString();
    integration.status = "ok";
    addLog("warn", "GraphQL simulado", "Ative integracoes reais para chamar o endpoint.", state.payload[target]);
    return true;
  }

  let response;
  try {
    response = await fetch(endpoint, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        query: params.query,
        variables: interpolateAny(params.variables || {}, state.payload),
      }),
    });
  } catch (err) {
    integration.status = "error";
    throw err;
  }
  integration.http_status = response.status;
  if (!response.ok) {
    integration.status = "error";
    throw new Error(`graphql_enrich: HTTP ${response.status}`);
  }
  const result = await response.json();
  if (result.errors && result.errors.length && params.required !== false) {
    integration.status = "error";
    throw new Error(`graphql_enrich: ${JSON.stringify(result.errors)}`);
  }
  const target = params.target || "external_data";
  const resultRoot = result && typeof result === "object" && "data" in result ? result.data : result;
  const selected = params.result_path ? getPath(resultRoot || {}, params.result_path) : resultRoot;
  if (selected === undefined && params.required !== false) {
    integration.status = "error";
    throw new Error(`graphql_enrich: result_path ${params.result_path} not found`);
  }
  state.payload[target] = selected;
  state.payload[`${target}_enriched_at`] = new Date().toISOString();
  integration.status = "ok";
  return true;
}

async function runRestCall(params, state) {
  const baseUrl = expandEnv(params.base_url || els.externalApiUrl.value);
  const method = String(params.method || "GET").toUpperCase();
  const endpoint = interpolateString(params.endpoint || "", state.payload);
  const integration = recordIntegration(state, {
    type: "rest",
    mode: els.integrations.checked ? "real" : "simulated",
    target: params.target || "http_response",
    method,
    endpoint: `${baseUrl.replace(/\/$/, "")}${endpoint}`,
    status: "attempted",
  });
  if (!els.integrations.checked) {
    const target = params.target || "http_response";
    state.payload[target] = {
      simulated: true,
      method,
      endpoint,
      status: "SIMULATED",
    };
    state.payload[`${target}_called_at`] = new Date().toISOString();
    integration.status = "ok";
    addLog("warn", "REST simulado", "Ative integracoes reais para chamar a API.", state.payload[target]);
    return true;
  }

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
  let response;
  try {
    response = await fetch(`${baseUrl.replace(/\/$/, "")}${endpoint}`, options);
  } catch (err) {
    integration.status = "error";
    throw err;
  }
  integration.http_status = response.status;
  if (!response.ok) {
    integration.status = "error";
    throw new Error(`rest_call: HTTP ${response.status}`);
  }
  const result = await response.json();
  const target = params.target || "http_response";
  state.payload[target] = params.result_path ? getPath(result, params.result_path) : result;
  state.payload[`${target}_called_at`] = new Date().toISOString();
  integration.status = "ok";
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
