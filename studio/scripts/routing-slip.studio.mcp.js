async function callMCPTool(name, argumentsValue = {}) {
  const endpoint = (els.mcpEndpoint?.value || "").trim();
  if (!endpoint) throw new Error("Configure o MCP endpoint antes de chamar uma tool.");
  const headers = { "Content-Type": "application/json" };
  const apiKey = (els.mcpApiKey?.value || "").trim();
  if (apiKey) {
    headers.Authorization = `Bearer ${apiKey}`;
    headers["X-API-Key"] = apiKey;
  }
  const response = await fetch(endpoint, {
    method: "POST",
    headers,
    body: JSON.stringify({
      jsonrpc: "2.0",
      id: Date.now(),
      method: "tools/call",
      params: {
        name,
        arguments: argumentsValue,
      },
    }),
  });
  const body = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(`MCP HTTP ${response.status}`);
  if (body.error) throw new Error(body.error.message || "MCP retornou erro.");
  return body.result?.structuredContent || body.result || body;
}

async function validateWorkflowViaMCP() {
  clearLogs();
  try {
    const result = await callMCPTool("validate_workflow", { yaml: els.workflow.value });
    const issues = Array.isArray(result.issues) ? result.issues : [];
    addLog(result.valid ? "ok" : "warn", "Validacao MCP concluida", result.valid ? "Workflow valido." : `${issues.length} ocorrencia(s) encontrada(s).`, result);
    renderMCPResult("Validacao MCP", result, mcpValidationToMarkdown(result));
  } catch (err) {
    addLog("error", "Falha na validacao MCP", err.message);
  }
}

async function explainWorkflowViaMCP() {
  clearLogs();
  try {
    const result = await callMCPTool("explain_workflow", { yaml: els.workflow.value });
    addLog("ok", "Explicacao MCP concluida", `${(result.steps || []).length} etapa(s) analisada(s).`, result);
    renderMCPResult("Explicacao MCP", result, mcpExplanationToMarkdown(result));
  } catch (err) {
    addLog("error", "Falha ao explicar workflow via MCP", err.message);
  }
}

function showConnectorDiagnostics() {
  clearLogs();
  let workflow;
  try {
    workflow = jsyaml.load(els.workflow.value);
  } catch (err) {
    addLog("error", "YAML invalido", err.message);
    return;
  }
  const steps = Array.isArray(workflow?.steps) ? workflow.steps : [];
  const connectors = steps
    .map((step, index) => connectorDiagnosticFromStep(step, index))
    .filter(Boolean);
  const circuitBreakers = connectors.filter((item) => item.circuit_breaker?.enabled);
  const diagnostics = {
    workflow: workflow?.name || "workflow",
    connectors,
    summary: {
      total: connectors.length,
      graphql: connectors.filter((item) => item.type === "graphql").length,
      rest: connectors.filter((item) => item.type === "rest").length,
      notify: connectors.filter((item) => item.type === "notify").length,
      circuit_breakers: circuitBreakers.length,
    },
  };
  addLog("info", "Diagnostico de conectores", `${connectors.length} integracao(oes) encontrada(s).`, diagnostics);
  renderMCPResult("Diagnostico de conectores", diagnostics, connectorDiagnosticsToMarkdown(diagnostics));
}

function connectorDiagnosticFromStep(step, index) {
  const params = step?.params || {};
  if (step?.name === "graphql_enrich") {
    return {
      step_index: index,
      step_id: step.id || "",
      type: "graphql",
      target: params.target || "external_data",
      endpoint: params.endpoint || els.graphqlEndpoint.value,
      timeout_ms: params.timeout_ms || "",
      required: params.required !== false,
      retries: step.resilience?.retry?.attempts || 1,
      circuit_breaker: step.resilience?.circuit_breaker || step.resilience?.circuitBreaker || {},
    };
  }
  if (step?.name === "rest_call") {
    return {
      step_index: index,
      step_id: step.id || "",
      type: "rest",
      method: params.method || "GET",
      target: params.target || "http_response",
      endpoint: `${params.base_url || els.externalApiUrl.value}${params.endpoint || ""}`,
      timeout_ms: params.timeout_ms || "",
      required: params.required !== false,
      retries: step.resilience?.retry?.attempts || 1,
      circuit_breaker: step.resilience?.circuit_breaker || step.resilience?.circuitBreaker || {},
    };
  }
  if (step?.name === "notify") {
    return {
      step_index: index,
      step_id: step.id || "",
      type: "notify",
      target: params.channel || "log",
      required: false,
      retries: step.resilience?.retry?.attempts || 1,
      circuit_breaker: {},
    };
  }
  return null;
}

function renderMCPResult(title, data, markdown) {
  document.querySelector("#workspace-mode-label").textContent = "MCP";
  document.querySelector("#result-panel-title").textContent = title;
  document.querySelector("#result-panel-meta").textContent = "Diagnostico";
  const article = document.createElement("article");
  article.className = "mcp-result";
  article.innerHTML = `
    <div class="mcp-result-content">${window.marked ? marked.parse(markdown) : `<pre>${escapeHtml(markdown)}</pre>`}</div>
    <details class="mcp-result-raw">
      <summary>Resposta estruturada</summary>
      <pre class="log-json">${escapeHtml(JSON.stringify(data, null, 2))}</pre>
    </details>
  `;
  els.timeline.appendChild(article);
  els.timeline.scrollTop = els.timeline.scrollHeight;
}

function mcpValidationToMarkdown(result) {
  const issues = Array.isArray(result.issues) ? result.issues : [];
  if (!issues.length) return "## Workflow valido\n\nNenhuma ocorrencia foi reportada pelo MCP.";
  return [
    "## Ocorrencias encontradas",
    "",
    "| Nivel | Campo | Mensagem |",
    "|---|---|---|",
    ...issues.map((issue) => `| ${issue.level || "info"} | ${issue.field || "-"} | ${String(issue.message || "").replace(/\|/g, "\\|")} |`),
  ].join("\n");
}

function mcpExplanationToMarkdown(result) {
  const steps = Array.isArray(result.steps) ? result.steps : [];
  return [
    `## ${result.name || "Workflow"}`,
    "",
    result.description || "Workflow analisado pelo MCP.",
    "",
    "| # | Handler | Papel | Destino |",
    "|---:|---|---|---|",
    ...steps.map((step) => {
      const role = step.integration ? "Integracao" : step.control ? "Controle" : "Processamento";
      return `| ${Number(step.index) + 1} | ${step.handler || "-"} | ${role} | ${step.target || "-"} |`;
    }),
  ].join("\n");
}

function connectorDiagnosticsToMarkdown(diagnostics) {
  const rows = diagnostics.connectors.map((item) => {
    const circuit = item.circuit_breaker?.enabled ? "habilitado" : "desabilitado";
    return `| ${item.step_index + 1} | ${item.type} | ${item.target || "-"} | ${item.endpoint || "-"} | ${item.retries} | ${circuit} |`;
  });
  return [
    `## ${diagnostics.workflow}`,
    "",
    "Diagnostico local das etapas que acionam servicos externos.",
    "",
    "| Step | Tipo | Target | Endpoint | Tentativas | Circuit breaker |",
    "|---:|---|---|---|---:|---|",
    ...(rows.length ? rows : ["| - | - | - | - | - | - |"]),
  ].join("\n");
}
