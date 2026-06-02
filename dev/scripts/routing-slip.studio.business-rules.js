async function readWorkspaceProjectSummary(fileHandle) {
  try {
    const text = await WorkspaceFS.readFile(fileHandle);
    const project = parseWorkflowProject(text, "", fileHandle.name || "workflow.yaml");
    return {
      rules: (project.business_rules || []).map((rule) => ({
        rule_id: rule.rule_id,
        name: rule.human_context?.name || rule.rule_id,
        status: rule.status || "DRAFT",
        execution_order: rule.execution_order || 0,
      })),
    };
  } catch {
    return { rules: [] };
  }
}

function parseWorkflowProject(text, serviceName = "", fileName = "workflow.yaml") {
  let parsed = null;
  try {
    parsed = window.jsyaml.load(text) || {};
  } catch {
    parsed = {};
  }
  const isProject = parsed && typeof parsed === "object" && ("workflow_script" in parsed || "project_settings" in parsed || "business_rules" in parsed);
  if (isProject) {
    const workflowScriptRaw = extractYamlBlock(text, "workflow_script");
    return {
      service: parsed.service || serviceName,
      usecase: parsed.usecase || fileName.replace(/\.(ya?ml)$/i, ""),
      project_settings: parsed.project_settings || defaultProjectSettings(),
      payload_data: typeof parsed.payload_data === "string" ? parsed.payload_data : JSON.stringify(parsed.payload_data || {}, null, 2),
      workflow_script: parsed.workflow_script || {},
      workflow_script_raw: workflowScriptRaw || "",
      business_rules: Array.isArray(parsed.business_rules) ? parsed.business_rules : [],
    };
  }
  return {
    service: serviceName,
    usecase: fileName.replace(/\.(ya?ml)$/i, ""),
    project_settings: defaultProjectSettings(),
    payload_data: els.payload?.value || "{}",
    workflow_script: parsed || {},
    workflow_script_raw: text || "",
    business_rules: [],
  };
}

function workflowScriptFromProjectText(text, serviceName = "", fileName = "workflow.yaml") {
  return parseWorkflowProject(text, serviceName, fileName).workflow_script || {};
}

function applyWorkflowProject(project) {
  currentProjectEnvelope = project;
  currentBusinessRules = normalizeBusinessRules(project.business_rules || []);
  els.workflow.value = formatWorkflowYamlForEditor(project.workflow_script_raw || window.jsyaml.dump(project.workflow_script || {}, {
    lineWidth: 140,
    noRefs: true,
    sortKeys: false,
  }));
  applyProjectSettings(project.project_settings || {});
  els.payload.value = normalizePayloadData(project.payload_data);
}

function defaultProjectSettings() {
  return {
    use_real_integrations: Boolean(els.integrations?.checked),
    integrations: {
      graphql_endpoint: els.graphqlEndpoint?.value || "http://localhost:8090/graphql",
      rest_workflow_endpoint: els.workflowEndpoint?.value || "http://localhost:8088/process",
      external_api_url: els.externalApiUrl?.value || "https://mock.raysouz.studio",
    },
    mcp_server: {
      mcp_endpoint: els.mcpEndpoint?.value || "http://localhost:9091/mcp",
      mcp_api_key: els.mcpApiKey?.value || "",
    },
  };
}

function applyProjectSettings(settings) {
  if (!settings || typeof settings !== "object") return;
  els.integrations.checked = Boolean(settings.use_real_integrations);
  if (settings.integrations?.graphql_endpoint) els.graphqlEndpoint.value = settings.integrations.graphql_endpoint;
  if (settings.integrations?.rest_workflow_endpoint) els.workflowEndpoint.value = settings.integrations.rest_workflow_endpoint;
  if (settings.integrations?.external_api_url) els.externalApiUrl.value = settings.integrations.external_api_url;
  if (settings.mcp_server?.mcp_endpoint) els.mcpEndpoint.value = settings.mcp_server.mcp_endpoint;
  if (settings.mcp_server?.mcp_api_key) els.mcpApiKey.value = settings.mcp_server.mcp_api_key;
}

function normalizePayloadData(payloadData) {
  if (typeof payloadData !== "string") return JSON.stringify(payloadData || {}, null, 2);
  try {
    return JSON.stringify(JSON.parse(payloadData), null, 2);
  } catch {
    return payloadData || "{}";
  }
}

function currentProjectSettings() {
  return {
    use_real_integrations: Boolean(els.integrations.checked),
    integrations: {
      graphql_endpoint: els.graphqlEndpoint.value,
      rest_workflow_endpoint: els.workflowEndpoint.value,
      external_api_url: els.externalApiUrl.value,
    },
    mcp_server: {
      mcp_endpoint: els.mcpEndpoint.value,
      mcp_api_key: els.mcpApiKey.value,
    },
  };
}

function serializeCurrentWorkflowProject() {
  const workflowScript = window.jsyaml.load(els.workflow.value) || {};
  const project = {
    service: currentWorkspaceFile.serviceName || currentProjectEnvelope?.service || "",
    usecase: (currentWorkspaceFile.fileName || currentProjectEnvelope?.usecase || "workflow").replace(/\.(ya?ml)$/i, ""),
    project_settings: currentProjectSettings(),
    payload_data: els.payload.value,
    workflow_script: workflowScript,
    business_rules: normalizeBusinessRules(currentBusinessRules),
  };
  currentProjectEnvelope = project;
  return serializeWorkflowProjectWithRawScript(project, els.workflow.value);
}

function serializeWorkflowProjectWithRawScript(project, workflowYaml) {
  const head = {
    service: project.service,
    usecase: project.usecase,
    project_settings: project.project_settings,
  };
  const payload = String(project.payload_data || "{}").replace(/\s+$/g, "");
  const rules = { business_rules: project.business_rules || [] };
  return [
    window.jsyaml.dump(head, { lineWidth: 140, noRefs: true, sortKeys: false }).trimEnd(),
    "payload_data: |-",
    indentYamlBlock(payload, 2),
    "workflow_script:",
    indentYamlBlock(String(workflowYaml || "").replace(/\s+$/g, ""), 2),
    window.jsyaml.dump(rules, { lineWidth: 140, noRefs: true, sortKeys: false }).trimEnd(),
    "",
  ].join("\n");
}

function extractYamlBlock(text, key) {
  const lines = String(text || "").split("\n");
  const keyPattern = new RegExp(`^${escapeRegExp(key)}\\s*:\\s*(?:#.*)?$`);
  const startIndex = lines.findIndex((line) => keyPattern.test(line));
  if (startIndex < 0) return "";
  const block = [];
  for (let index = startIndex + 1; index < lines.length; index += 1) {
    const line = lines[index];
    if (/^\S[^:]*\s*:/.test(line)) break;
    block.push(line);
  }
  return dedentYamlBlock(block.join("\n")).replace(/\s+$/g, "") + "\n";
}

function dedentYamlBlock(text) {
  const lines = String(text || "").split("\n");
  const indents = lines
    .filter((line) => line.trim() !== "")
    .map((line) => line.match(/^\s*/)[0].length);
  const indent = indents.length ? Math.min(...indents) : 0;
  return lines.map((line) => line.trim() === "" ? "" : line.slice(indent)).join("\n");
}

function indentYamlBlock(text, size) {
  const prefix = " ".repeat(size);
  return String(text || "").split("\n").map((line) => `${prefix}${line}`).join("\n");
}

function escapeRegExp(value) {
  return String(value).replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function summarizeCurrentProject() {
  return {
    rules: normalizeBusinessRules(currentBusinessRules).map((rule) => ({
      rule_id: rule.rule_id,
      name: rule.human_context?.name || rule.rule_id,
      status: rule.status || "DRAFT",
      execution_order: rule.execution_order || 0,
    })),
  };
}

function normalizeBusinessRules(rules) {
  return (Array.isArray(rules) ? rules : [])
    .filter((rule) => rule && typeof rule === "object")
    .map((rule) => {
      const ruleID = rule.rule_id || rule.Rule_id || rule.id || `rule_${generateUUID().slice(0, 8)}`;
      return {
        rule_id: ruleID,
        domain: rule.domain || "",
        context: rule.context || "",
        execution_order: Number(rule.execution_order || 0),
        status: rule.status || "DRAFT",
        human_context: rule.human_context || { name: ruleID || "Nova regra", description: "", business_owner: "" },
        engineering_context: rule.engineering_context || {},
        ai_logic: rule.ai_logic || "",
        technical_metadata: normalizeBusinessRuleTechnicalMetadata(rule.technical_metadata || {}),
      };
    })
    .sort((a, b) => Number(a.execution_order || 0) - Number(b.execution_order || 0) || String(a.rule_id).localeCompare(String(b.rule_id)));
}

function normalizeBusinessRuleTechnicalMetadata(metadata) {
  const observability = metadata.observability || {};
  return {
    ...metadata,
    dependencies: normalizeBusinessRuleDependencies(metadata.dependencies || []),
    observability: {
      ...observability,
      datadog_monitor_ids: businessRuleList(observability.datadog_monitor_ids || observability.datadog_monitor_id),
      custom_metrics: normalizeBusinessRuleCustomMetrics(observability.custom_metrics || observability.custom_metric),
      log_markers: businessRuleList(observability.log_markers || observability.logs),
    },
  };
}

function currentBusinessRuleIndex(ruleID) {
  return currentBusinessRules.findIndex((rule) => rule.rule_id === ruleID);
}

function businessRuleField(path, value, options = {}) {
  const tag = options.multiline ? "textarea" : "input";
  const readonly = options.editing && !options.locked ? "" : "readonly";
  const type = options.type || "text";
  const rows = options.rows ? ` rows="${options.rows}"` : "";
  const classes = options.full ? " form-field--full" : "";
  const escaped = escapeHtml(value ?? "");
  if (tag === "textarea") {
    return `
      <label class="business-rule-field${classes}">
        <span>${escapeHtml(options.label)}</span>
        <textarea data-rule-field="${escapeHtml(path)}" ${readonly}${rows} spellcheck="false">${escaped}</textarea>
      </label>`;
  }
  return `
    <label class="business-rule-field${classes}">
      <span>${escapeHtml(options.label)}</span>
      <input data-rule-field="${escapeHtml(path)}" type="${escapeHtml(type)}" value="${escaped}" ${readonly}>
    </label>`;
}

function businessRuleStatusField(rule, editing) {
  if (!editing) {
    return `<span class="business-rule-status">${escapeHtml(rule.status || "DRAFT")}</span>`;
  }
  const statuses = ["DRAFT", "ACTIVE", "INACTIVE", "DEPRECATED"];
  return `
    <select class="business-rule-status business-rule-status--edit" data-rule-field="status" aria-label="Status da regra">
      ${statuses.map((status) => `<option value="${status}" ${status === rule.status ? "selected" : ""}>${status}</option>`).join("")}
    </select>`;
}

function businessRuleArrayValue(values) {
  return Array.isArray(values) ? values.filter(Boolean).join(", ") : String(values || "");
}

function businessRuleDependenciesText(rule) {
  const dependencies = normalizeBusinessRuleDependencies(rule.technical_metadata?.dependencies || []);
  return dependencies.length ? dumpBusinessRuleYaml(dependencies) : "";
}

function parseBusinessRuleDependencies(value) {
  const text = String(value || "").trim();
  if (!text) return [];
  try {
    return normalizeBusinessRuleDependencies(window.jsyaml.load(text));
  } catch {
    return normalizeBusinessRuleDependencies(text.split(/\n|,/).map((item) => item.trim()).filter(Boolean));
  }
}

function parseCommaList(value) {
  return String(value || "")
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

function businessRuleList(value) {
  if (Array.isArray(value)) return value.map((item) => String(item || "").trim()).filter(Boolean);
  if (typeof value === "string") return parseCommaList(value);
  if (value === undefined || value === null) return [];
  return [String(value).trim()].filter(Boolean);
}

function normalizeBusinessRuleDependencies(value) {
  const items = Array.isArray(value) ? value : value ? [value] : [];
  return items.map((item) => {
    if (typeof item === "string") {
      const [first, second] = item.split("|").map((part) => part.trim());
      return { type: "business_rule", rule_id: first, relation: second || "depends_on" };
    }
    if (!item || typeof item !== "object") return null;
    const type = String(item.type || item.Type || item.kind || "").toLowerCase().replace("-", "_") || (item.rule_id || item.Rule_id ? "business_rule" : "system");
    if (type === "business_rule" || type === "business-rule") {
      return {
        type: "business_rule",
        rule_id: item.rule_id || item.Rule_id || item.id || "",
        relation: item.relation || item.Relation || item.action || item.Action || "depends_on",
      };
    }
    return {
      type: "system",
      name: item.name || item.Name || "",
      component: item.component || item.Component || "",
      action: item.action || item.Action || "",
    };
  }).filter((item) => item && (item.rule_id || item.name || item.component));
}

function normalizeBusinessRuleCustomMetrics(value) {
  const items = Array.isArray(value) ? value : value ? [value] : [];
  return items.map((item) => {
    if (typeof item === "string") return { name: item, type: "count", tags: [] };
    if (!item || typeof item !== "object") return null;
    return {
      name: item.name || item.Name || item.metric || "",
      type: item.type || item.Type || "count",
      tags: businessRuleMetricTags(item.tags || item.Tags),
    };
  }).filter((item) => item && item.name);
}

function businessRuleMetricTags(value) {
  if (Array.isArray(value)) return value.map((item) => {
    if (typeof item === "string") return item;
    if (item && typeof item === "object") {
      return Object.entries(item).map(([key, val]) => `${key}:${val}`).join(",");
    }
    return String(item || "");
  }).filter(Boolean);
  if (typeof value === "string") return parseCommaList(value);
  if (value && typeof value === "object") return Object.entries(value).map(([key, val]) => `${key}:${val}`);
  return [];
}

function parseBusinessRuleCustomMetrics(value) {
  const text = String(value || "").trim();
  if (!text) return [];
  try {
    return normalizeBusinessRuleCustomMetrics(window.jsyaml.load(text));
  } catch {
    return normalizeBusinessRuleCustomMetrics(parseCommaList(text));
  }
}

function businessRuleCustomMetricsText(metrics) {
  const normalized = normalizeBusinessRuleCustomMetrics(metrics);
  return normalized.length ? dumpBusinessRuleYaml(normalized) : "";
}

function dumpBusinessRuleYaml(value) {
  return window.jsyaml.dump(value, { lineWidth: 140, noRefs: true, sortKeys: false }).trimEnd();
}

function setNestedValue(target, path, value) {
  const parts = path.split(".");
  let cursor = target;
  parts.slice(0, -1).forEach((part) => {
    if (!cursor[part] || typeof cursor[part] !== "object") cursor[part] = {};
    cursor = cursor[part];
  });
  cursor[parts[parts.length - 1]] = value;
}

function collectBusinessRuleForm(ruleID) {
  const original = currentBusinessRules.find((rule) => rule.rule_id === ruleID) || {};
  const updated = structuredClone(original);
  document.querySelectorAll("[data-rule-field]").forEach((field) => {
    const path = field.dataset.ruleField;
    if (path === "rule_id") return;
    let value = field.value;
    if (path === "execution_order") value = Number(value || 0);
    if (path === "technical_metadata.dependencies") value = parseBusinessRuleDependencies(value);
    if (path === "technical_metadata.observability.datadog_monitor_ids") value = parseCommaList(value);
    if (path === "technical_metadata.observability.custom_metrics") value = parseBusinessRuleCustomMetrics(value);
    if (path === "technical_metadata.observability.log_markers") value = parseCommaList(value);
    setNestedValue(updated, path, value);
  });
  updated.technical_metadata = normalizeBusinessRuleTechnicalMetadata(updated.technical_metadata || {});
  return updated;
}

function renderBusinessRuleDependencies(rule, editing) {
  if (editing) {
    return businessRuleField("technical_metadata.dependencies", businessRuleDependenciesText(rule), {
      label: "Dependencias (YAML)",
      multiline: true,
      rows: 7,
      full: true,
      editing,
    });
  }
  const dependencies = normalizeBusinessRuleDependencies(rule.technical_metadata?.dependencies || []);
  if (!dependencies.length) {
    return `<div class="business-rule-empty">Sem dependencias cadastradas.</div>`;
  }
  return `
    <div class="business-rule-dependencies">
      ${dependencies.map((dep) => {
        const type = dep.type || "system";
        const ruleID = dep.rule_id || "";
        const name = dep.name || dep.component || ruleID || "-";
        const relation = dep.relation || dep.action || type;
        const found = type === "business_rule" && ruleID && currentBusinessRules.some((item) => item.rule_id === ruleID);
        return found
          ? `<button class="business-rule-dependency" type="button" data-dependency-rule="${escapeHtml(ruleID)}">${escapeHtml(ruleID)} <em>${escapeHtml(relation)}</em></button>`
          : `<span class="business-rule-dependency disabled">${escapeHtml(name)} <em>${escapeHtml(type)}:${escapeHtml(relation)}</em></span>`;
      }).join("")}
    </div>`;
}

function navigateBusinessRule(delta) {
  const index = currentBusinessRuleIndex(activeBusinessRuleID);
  const next = currentBusinessRules[index + delta];
  if (!next) return;
  openBusinessRule(currentWorkspaceFile.serviceName, currentWorkspaceFile.fileName, next.rule_id, { preserveBackStack: true });
}

function openBusinessRuleDependency(ruleID) {
  if (activeBusinessRuleID && activeBusinessRuleID !== ruleID) businessRuleBackStack.push(activeBusinessRuleID);
  openBusinessRule(currentWorkspaceFile.serviceName, currentWorkspaceFile.fileName, ruleID, { preserveBackStack: true });
}

function backToBusinessRuleDependencyOrigin() {
  const previous = businessRuleBackStack.pop();
  if (!previous) return;
  openBusinessRule(currentWorkspaceFile.serviceName, currentWorkspaceFile.fileName, previous, { preserveBackStack: true });
}

function renderBusinessRulesForWorkflow(serviceName, file) {
  const rules = file.project?.rules || [];
  if (!rules.length) return "";
  return `
    <div class="workspace-rules">
      ${rules.map((rule) => `
        <button class="workspace-rule ${activeBusinessRuleID === rule.rule_id ? "active" : ""}" type="button"
          data-service="${escapeHtml(serviceName)}" data-file="${escapeHtml(file.name)}" data-rule-id="${escapeHtml(rule.rule_id)}">
          <i class="fa-regular fa-circle-dot" aria-hidden="true"></i>
          <span>${escapeHtml(rule.name || rule.rule_id)}</span>
          <em>${escapeHtml(rule.status || "DRAFT")}</em>
        </button>
      `).join("")}
    </div>`;
}

async function createBusinessRuleForCurrentWorkflow() {
  if (!currentWorkspaceFile.handle) {
    alert("Abra um workflow no workspace antes de criar uma regra.");
    return;
  }
  await createBusinessRule(currentWorkspaceFile.serviceName, currentWorkspaceFile.fileName);
}

async function createBusinessRule(serviceName, fileName) {
  if (!currentWorkspaceFile.handle || currentWorkspaceFile.serviceName !== serviceName || currentWorkspaceFile.fileName !== fileName) {
    await openWorkflowFile(serviceName, fileName);
  }
  const raw = prompt("ID unico da regra de negocio:");
  if (!raw?.trim()) return;
  const ruleID = normalizeName(raw);
  if (currentBusinessRules.some((rule) => rule.rule_id === ruleID)) {
    alert(`A regra "${ruleID}" ja existe neste usecase.`);
    return;
  }
  const nextOrder = currentBusinessRules.reduce((max, rule) => Math.max(max, Number(rule.execution_order || 0)), 0) + 1;
  const rule = {
    rule_id: ruleID,
    domain: serviceName,
    context: fileName.replace(/\.(ya?ml)$/i, ""),
    execution_order: nextOrder,
    status: "DRAFT",
    human_context: {
      name: raw.trim(),
      description: "Descreva a regra de negocio, sua motivacao e o resultado esperado.",
      business_owner: "",
    },
    engineering_context: {
      application_name: serviceName,
      application_type: "workflow",
      repository_url: "",
      entrypoint: "",
    },
    ai_logic: "Explique como uma IA deve interpretar, validar ou investigar esta regra.",
    technical_metadata: {
      dependencies: [],
      observability: {
        datadog_monitor_ids: [],
        custom_metrics: [],
        log_markers: [],
      },
    },
  };
  currentBusinessRules = normalizeBusinessRules([...currentBusinessRules, rule]);
  markProjectDirty();
  renderWorkspace();
  openBusinessRule(serviceName, fileName, ruleID, { edit: true });
}

async function openBusinessRule(serviceName, fileName, ruleID, options = {}) {
  if (!currentWorkspaceFile.handle || currentWorkspaceFile.serviceName !== serviceName || currentWorkspaceFile.fileName !== fileName) {
    await openWorkflowFile(serviceName, fileName, { preserveLogs: true });
  }
  const rule = currentBusinessRules.find((item) => item.rule_id === ruleID);
  if (!rule) return;
  activeBusinessRuleID = ruleID;
  if (!options.preserveBackStack) businessRuleBackStack = [];
  renderWorkspace();
  renderBusinessRuleViewer(rule, { edit: Boolean(options.edit) });
}

function renderBusinessRuleViewer(rule, options = {}) {
  const editing = Boolean(options.edit);
  const index = currentBusinessRuleIndex(rule.rule_id);
  const owner = rule.human_context?.business_owner || "";
  const observability = normalizeBusinessRuleTechnicalMetadata(rule.technical_metadata || {}).observability;
  els.timeline.classList.remove("timeline--docs");
  els.timeline.classList.add("timeline--business-rule");
  document.querySelector("#workspace-mode-label").textContent = "Business rules";
  document.querySelector("#result-panel-title").textContent = "Regra de negocio";
  document.querySelector("#result-panel-meta").textContent = editing ? "Edicao" : "Visualizacao";
  els.summary.textContent = `${rule.rule_id} ${rule.status || ""}`.trim();
  els.timeline.innerHTML = `
    <section class="business-rule-viewer">
      <header class="business-rule-form-head">
        <div class="business-rule-actions">
          ${businessRuleBackStack.length ? `<button id="business-rule-back" type="button"><i class="fa-solid fa-arrow-left" aria-hidden="true"></i> Voltar</button>` : ""}
          <button id="business-rule-edit-toggle" class="workflow-view-toggle business-rule-edit-toggle ${editing ? "micro" : ""}" type="button" aria-pressed="${editing ? "true" : "false"}" title="${editing ? "Salvar e visualizar" : "Editar regra"}">
            <i aria-hidden="true"></i>
            <span>Visualizar</span>
            <span>Editar</span>
          </button>
          <button id="business-rule-delete" class="business-rule-delete danger" type="button" title="Excluir regra" aria-label="Excluir regra"><i class="fa-solid fa-trash" aria-hidden="true"></i></button>
        </div>
        <div class="business-rule-title-row">
          <h3>${escapeHtml(rule.rule_id)}</h3>
          ${businessRuleStatusField(rule, editing)}
        </div>
      </header>

      <div class="business-rule-form-grid">
        ${businessRuleField("rule_id", rule.rule_id, { label: "ID da Regra", editing, locked: true })}
        ${businessRuleField("execution_order", rule.execution_order || 0, { label: "Ordem de Execucao", type: "number", editing })}
        ${businessRuleField("domain", rule.domain || "", { label: "Dominio", editing })}
        ${businessRuleField("context", rule.context || "", { label: "Contexto", editing })}
        ${businessRuleField("human_context.business_owner", owner, { label: "Owner", editing })}
      </div>

      <h3 class="business-rule-section-title">1. Visao Humana</h3>
      ${businessRuleField("human_context.name", rule.human_context?.name || "", { label: "Nome da Regra", full: true, editing })}
      ${businessRuleField("human_context.description", rule.human_context?.description || "", { label: "Descricao de Negocio", multiline: true, rows: 4, full: true, editing })}

      <h3 class="business-rule-section-title">2. Visao de Engenharia</h3>
      <div class="business-rule-form-grid">
        ${businessRuleField("engineering_context.application_name", rule.engineering_context?.application_name || "", { label: "Aplicacao", editing })}
        ${businessRuleField("engineering_context.application_type", rule.engineering_context?.application_type || "", { label: "Tipo", editing })}
        ${businessRuleField("engineering_context.repository_url", rule.engineering_context?.repository_url || "", { label: "Repositorio", full: true, editing })}
        ${businessRuleField("engineering_context.entrypoint", rule.engineering_context?.entrypoint || "", { label: "Entrypoint (Classe/Arquivo/Workflow)", full: true, editing })}
      </div>

      <h3 class="business-rule-section-title">3. IA e Observabilidade</h3>
      ${businessRuleField("ai_logic", rule.ai_logic || "", { label: "Logica Deterministica / AI Logic", multiline: true, rows: 5, full: true, editing })}
      <div class="business-rule-form-grid">
        ${businessRuleField("technical_metadata.observability.datadog_monitor_ids", businessRuleArrayValue(observability.datadog_monitor_ids), { label: "Datadog Monitor IDs", editing })}
        ${businessRuleField("technical_metadata.observability.log_markers", businessRuleArrayValue(observability.log_markers), { label: "Log Markers", full: true, editing })}
      </div>
      ${businessRuleField("technical_metadata.observability.custom_metrics", businessRuleCustomMetricsText(observability.custom_metrics), { label: "Metricas Customizadas (YAML)", multiline: true, rows: 7, full: true, editing })}

      <h3 class="business-rule-section-title">4. Dependencias</h3>
      ${renderBusinessRuleDependencies(rule, editing)}

      <footer class="business-rule-navigation">
        <button id="business-rule-prev" type="button" ${index <= 0 ? "disabled" : ""}><i class="fa-solid fa-arrow-left" aria-hidden="true"></i> Anterior</button>
        <span>${escapeHtml(String(index + 1))} de ${escapeHtml(String(currentBusinessRules.length))}</span>
        <button id="business-rule-next" type="button" ${index >= currentBusinessRules.length - 1 ? "disabled" : ""}>Proxima <i class="fa-solid fa-arrow-right" aria-hidden="true"></i></button>
      </footer>
    </section>
  `;
  document.querySelector("#business-rule-back")?.addEventListener("click", backToBusinessRuleDependencyOrigin);
  document.querySelector("#business-rule-edit-toggle")?.addEventListener("click", () => toggleBusinessRuleEdit(rule.rule_id));
  document.querySelector("#business-rule-delete")?.addEventListener("click", () => deleteBusinessRule(currentWorkspaceFile.serviceName, currentWorkspaceFile.fileName, rule.rule_id));
  document.querySelector("#business-rule-prev")?.addEventListener("click", () => navigateBusinessRule(-1));
  document.querySelector("#business-rule-next")?.addEventListener("click", () => navigateBusinessRule(1));
  document.querySelectorAll("[data-dependency-rule]").forEach((button) => {
    button.addEventListener("click", () => openBusinessRuleDependency(button.dataset.dependencyRule));
  });
}

function toggleBusinessRuleEdit(ruleID) {
  const editing = Boolean(document.querySelector(".business-rule-status--edit"));
  if (!editing) {
    const rule = currentBusinessRules.find((item) => item.rule_id === ruleID);
    if (rule) renderBusinessRuleViewer(rule, { edit: true });
    return;
  }
  try {
    const updated = collectBusinessRuleForm(ruleID);
    if (!updated?.rule_id) throw new Error("rule_id e obrigatorio.");
    const duplicated = currentBusinessRules.some((rule) => rule.rule_id === updated.rule_id && rule.rule_id !== ruleID);
    if (duplicated) throw new Error(`rule_id "${updated.rule_id}" ja existe neste usecase.`);
    currentBusinessRules = normalizeBusinessRules(currentBusinessRules.map((rule) => rule.rule_id === ruleID ? updated : rule));
    activeBusinessRuleID = updated.rule_id;
    markProjectDirty();
    renderWorkspace();
    renderBusinessRuleViewer(currentBusinessRules.find((rule) => rule.rule_id === updated.rule_id), { edit: false });
  } catch (err) {
    alert(`Regra invalida: ${err.message}`);
  }
}

async function deleteBusinessRule(serviceName, fileName, ruleID) {
  if (!currentWorkspaceFile.handle || currentWorkspaceFile.serviceName !== serviceName || currentWorkspaceFile.fileName !== fileName) {
    await openWorkflowFile(serviceName, fileName, { preserveLogs: true });
  }
  const dependents = currentBusinessRules.filter((rule) =>
    normalizeBusinessRuleDependencies(rule.technical_metadata?.dependencies || []).some((dep) => dep.type === "business_rule" && dep.rule_id === ruleID)
  );
  const warning = dependents.length
    ? `\n\nAtenção: ${dependents.length} regra(s) dependem dela: ${dependents.map((rule) => rule.rule_id).join(", ")}.`
    : "";
  if (!confirm(`Excluir a regra "${ruleID}"?${warning}`)) return;
  currentBusinessRules = currentBusinessRules.filter((rule) => rule.rule_id !== ruleID);
  if (activeBusinessRuleID === ruleID) activeBusinessRuleID = "";
  markProjectDirty();
  renderWorkspace();
  clearLogs();
}

function markProjectDirty() {
  if (!currentWorkspaceFile.handle) return;
  projectDirty = true;
  const service = findService(currentWorkspaceFile.serviceName);
  const file = service?.files.find((item) => item.name === currentWorkspaceFile.fileName);
  if (file) file.project = summarizeCurrentProject();
  if (els.saveWorkflowFile) els.saveWorkflowFile.disabled = false;
  scheduleStudioSave();
}

function scrollActiveLogPanelToTop() {
  const panel = document.querySelector(".execution-log-panel.active") || els.timeline;
  panel.scrollTo({ top: 0, behavior: "smooth" });
}
