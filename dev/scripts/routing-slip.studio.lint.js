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
  validateBusinessRulesCoverage(workflow, issues);
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
  validateResilience(step, label, issues);
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
  if (step.name === "filter_array") {
    if (!stringValue(params.source) && !stringValue(params.field)) {
      issues.push(error(`${label} filter_array precisa de source.`));
    }
    if (!params.where && !stringValue(params.expr) && !stringValue(params.expression)) {
      issues.push(error(`${label} filter_array precisa de where ou expr.`));
    }
    if (params.where && typeof params.where !== "object") {
      issues.push(error(`${label} filter_array where deve ser objeto ou lista.`));
    }
  }
  if (step.name === "array_transform") {
    if (!stringValue(params.source)) {
      issues.push(error(`${label} array_transform precisa de source.`));
    }
    if (params.updates && !Array.isArray(params.updates)) {
      issues.push(error(`${label} array_transform updates deve ser uma lista.`));
    }
    if (params.nested && !Array.isArray(params.nested)) {
      issues.push(error(`${label} array_transform nested deve ser uma lista.`));
    }
  }
  if (step.name === "jump_if") {
    if (!stringValue(params.field) && !stringValue(params.exists)) issues.push(error(`${label} jump_if precisa de field ou exists.`));
    if (!stringValue(params.to)) issues.push(error(`${label} jump_if precisa de to com id ou name do step destino.`));
    if (!("equals" in params) && !("not_equals" in params) && !("exists" in params) && !("min_items" in params) && !("max_items" in params) && !("less_than" in params) && !("less_than_or_equal" in params) && !("greater_than" in params) && !("greater_than_or_equal" in params)) {
      issues.push(error(`${label} jump_if precisa de um operador de comparação.`));
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
    const ref = params.file || params.path || params.workflow || "";
    if (String(ref).startsWith("../") || String(ref).startsWith("./")) {
      issues.push(warn(`${label} workflow_ref deve preferir path baseado no workspace, como service-first/A.`));
    }
    if (!currentWorkspaceFile.handle) {
      issues.push(warn(`${label} workflow_ref e melhor testado com um arquivo aberto no workspace.`));
    } else if (workspaceState?.rootHandle && stringValue(ref)) {
      try {
        resolveWorkspaceWorkflow(String(ref), currentWorkspaceFile);
      } catch (err) {
        issues.push(error(`${label} ${err.message}`));
      }
    }
  }
}

function validateResilience(step, label, issues) {
  if (!step.resilience) return;
  if (typeof step.resilience !== "object") {
    issues.push(error(`${label}.resilience deve ser objeto.`));
    return;
  }
  const retry = step.resilience.retry || {};
  if (retry.attempts !== undefined && Number(retry.attempts) < 1) {
    issues.push(error(`${label}.resilience.retry.attempts deve ser maior ou igual a 1.`));
  }
  const backoff = String(retry.backoff || "").toLowerCase();
  if (backoff && !["exponential", "fixed", "none"].includes(backoff)) {
    issues.push(error(`${label}.resilience.retry.backoff deve ser exponential, fixed ou none.`));
  }
  const onFailure = step.resilience.on_failure || {};
  const action = String(onFailure.action || "").toLowerCase();
  if (action && !["stop", "continue", "skip", "jump"].includes(action)) {
    issues.push(error(`${label}.resilience.on_failure.action deve ser stop, continue, skip ou jump.`));
  }
  if (action === "jump" && !stringValue(onFailure.to)) {
    issues.push(error(`${label}.resilience.on_failure.to e obrigatorio quando action for jump.`));
  }
}

function validateBusinessRulesCoverage(workflow, issues) {
  const rules = normalizeBusinessRules(currentBusinessRules || [])
    .filter((rule) => String(rule.status || "ACTIVE").toUpperCase() === "ACTIVE");
  if (!rules.length) return;
  const workflowText = JSON.stringify(workflow).toLowerCase();
  rules.forEach((rule) => {
    const tokens = [
      rule.rule_id,
      normalizeName(rule.rule_id || ""),
      rule.human_context?.name || "",
    ].map((value) => String(value || "").toLowerCase()).filter(Boolean);
    if (!tokens.some((token) => workflowText.includes(token))) {
      issues.push(warn(`Regra ativa "${rule.rule_id}" ainda nao esta coberta pelo workflow. Inclua o rule_id/nome em um step, audit, log ou metrica quando iniciar a implementacao.`));
    }
    const requiredFields = businessRuleRequiredFields(rule);
    requiredFields.forEach((field) => {
      if (!workflowRequiresField(workflow, field) && !workflowText.includes(String(field).toLowerCase())) {
        issues.push(warn(`Regra "${rule.rule_id}" espera o campo "${field}", mas ele nao aparece no workflow.`));
      }
    });
    const observability = rule.technical_metadata?.observability || {};
    businessRuleMetricNames(observability.custom_metrics || observability.custom_metric).forEach((metric) => {
      if (!workflowHasDatadogMetric(workflow, metric)) {
        issues.push(warn(`Regra "${rule.rule_id}" declara metrica "${metric}", mas nao ha step datadog_metric correspondente.`));
      }
    });
    businessRuleList(observability.log_markers || observability.logs).forEach((marker) => {
      if (!workflowHasLogMarker(workflow, marker)) {
        issues.push(warn(`Regra "${rule.rule_id}" declara log marker "${marker}", mas nao ha step log correspondente.`));
      }
    });
    businessRuleDependencies(rule.technical_metadata?.dependencies || []).forEach((dependency) => {
      const dependencyID = dependency.rule_id || "";
      if (dependency.type === "business_rule" && dependencyID && !rules.some((item) => item.rule_id === dependencyID)) {
        issues.push(warn(`Regra "${rule.rule_id}" depende de "${dependencyID}", que nao esta ativa neste projeto.`));
      }
    });
  });
}

function businessRuleRequiredFields(rule) {
  return [
    ...businessRuleList(rule.required_inputs),
    ...businessRuleList(rule.required_fields),
    ...businessRuleList(rule.engineering_context?.required_fields),
    ...inferFieldsFromRuleText(`${rule.human_context?.description || ""}\n${rule.ai_logic || ""}`),
  ].filter((field, index, list) => field && list.indexOf(field) === index);
}

function inferFieldsFromRuleText(text) {
  return [...String(text || "").matchAll(/\{([a-zA-Z0-9_.-]+)\}/g)].map((match) => match[1]);
}

function businessRuleList(value) {
  if (Array.isArray(value)) return value.map((item) => String(item || "").trim()).filter(Boolean);
  if (typeof value === "string") return value.split(",").map((item) => item.trim()).filter(Boolean);
  if (value === undefined || value === null) return [];
  return [String(value).trim()].filter(Boolean);
}

function businessRuleMetricNames(value) {
  if (Array.isArray(value)) {
    return value.map((item) => typeof item === "string" ? item : (item?.name || item?.Name || item?.metric || "")).map((item) => String(item || "").trim()).filter(Boolean);
  }
  if (value && typeof value === "object") return businessRuleMetricNames([value]);
  return businessRuleList(value);
}

function businessRuleDependencies(value) {
  const items = Array.isArray(value) ? value : value ? [value] : [];
  return items.map((item) => {
    if (typeof item === "string") return { type: "business_rule", rule_id: item, relation: "depends_on" };
    if (!item || typeof item !== "object") return null;
    const type = String(item.type || item.Type || "").toLowerCase().replace("-", "_") || (item.rule_id || item.Rule_id ? "business_rule" : "system");
    if (type === "business_rule" || type === "business-rule") {
      return {
        type: "business_rule",
        rule_id: item.rule_id || item.Rule_id || item.id || "",
        relation: item.relation || item.Relation || "depends_on",
      };
    }
    return {
      type: "system",
      name: item.name || item.Name || "",
      component: item.component || item.Component || "",
      action: item.action || item.Action || "",
    };
  }).filter(Boolean);
}

function workflowRequiresField(workflow, field) {
  return (workflow.steps || []).some((step) =>
    step.name === "validate" && Array.isArray(step.params?.required) && step.params.required.includes(field)
  );
}

function workflowHasDatadogMetric(workflow, metric) {
  const expected = String(metric || "").toLowerCase();
  return (workflow.steps || []).some((step) =>
    step.name === "datadog_metric" && String(step.params?.metric || "").toLowerCase() === expected
  );
}

function workflowHasLogMarker(workflow, marker) {
  const expected = String(marker || "").toLowerCase();
  return (workflow.steps || []).some((step) =>
    step.name === "log" && String(step.params?.message || "").toLowerCase().includes(expected)
  );
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
