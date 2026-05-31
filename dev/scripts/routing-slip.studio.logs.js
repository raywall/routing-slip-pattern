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
  logSources = new Map();
  activeLogSourceKey = "";
}

function setStudioProcessing(active, title = "Processando workflow", copy = "Executando etapas e consolidando logs.") {
  studioExecutionRunning = Boolean(active);
  document.body.classList.toggle("studio-processing", studioExecutionRunning);
  if (els.processingOverlay) {
    els.processingOverlay.classList.toggle("active", studioExecutionRunning);
    els.processingOverlay.setAttribute("aria-hidden", studioExecutionRunning ? "false" : "true");
  }
  if (els.processingTitle) els.processingTitle.textContent = title;
  if (els.processingCopy) els.processingCopy.textContent = copy;
  if (els.run) els.run.disabled = studioExecutionRunning;
  if (els.reprocess) els.reprocess.disabled = studioExecutionRunning || !lastExecutionSnapshot;
}

function addLog(level, title, copy, data, options = {}) {
  const entry = document.createElement("article");
  entry.className = "log-entry";
  const stepIndex = options.stepIndex ?? activeExecutionStepIndex;
  const step = Number.isInteger(stepIndex) ? activeRuntimeWorkflow?.steps?.[stepIndex] : null;
  const sourceKey = options.sourceKey || step?.__sourceWorkflow || runtimeRootSourceKey || "workflow";
  const sourceLabel = options.sourceLabel || step?.__sourceLabel || sourceKey;
  if (Number.isInteger(stepIndex)) {
    entry.classList.add("log-entry--step");
    entry.dataset.stepIndex = String(stepIndex);
    entry.dataset.sourceKey = sourceKey;
    entry.dataset.sourceStepIndex = String(Number.isInteger(step?.__sourceStepIndex) ? step.__sourceStepIndex : stepIndex);
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
  const source = ensureLogSource(sourceKey, sourceLabel);
  const container = Number.isInteger(stepIndex) ? ensureStepGroup(stepIndex, step, activeRuntimeWorkflow?.steps?.length, sourceKey, sourceLabel).querySelector(".phase-logs") : source.panel;
  container.appendChild(entry);
  setActiveLogSource(sourceKey);
  source.panel.scrollTop = source.panel.scrollHeight;
}

function setupExecutionLogTabs(workflow) {
  els.timeline.innerHTML = `
    <div class="execution-log-tabs" role="tablist" aria-label="Logs por workflow"></div>
    <div class="execution-log-panels"></div>
  `;
  logSources = new Map();
  stepGroups = new Map();
  runtimeRootSourceKey = workflow?.__sourceWorkflow || workflowSourceKey(currentWorkspaceFile, workflow);
  ensureLogSource(runtimeRootSourceKey, workflow?.__sourceLabel || workflowSourceLabel(currentWorkspaceFile, workflow));
}

function ensureLogSource(sourceKey, sourceLabel = sourceKey) {
  if (!els.timeline.querySelector(".execution-log-tabs")) setupExecutionLogTabs(activeRuntimeWorkflow || lastWorkflow || {});
  if (logSources.has(sourceKey)) return logSources.get(sourceKey);

  const tabs = els.timeline.querySelector(".execution-log-tabs");
  const panels = els.timeline.querySelector(".execution-log-panels");
  const tab = document.createElement("button");
  tab.type = "button";
  tab.className = "execution-log-tab";
  tab.textContent = sourceLabel;
  tab.title = sourceKey;
  tab.addEventListener("click", () => setActiveLogSource(sourceKey));

  const panel = document.createElement("div");
  panel.className = "execution-log-panel";
  panel.dataset.sourceKey = sourceKey;

  tabs.appendChild(tab);
  panels.appendChild(panel);
  const source = { key: sourceKey, label: sourceLabel, tab, panel };
  logSources.set(sourceKey, source);
  if (!activeLogSourceKey) setActiveLogSource(sourceKey);
  return source;
}

function setActiveLogSource(sourceKey) {
  activeLogSourceKey = sourceKey;
  logSources.forEach((source) => {
    const active = source.key === sourceKey;
    source.tab.classList.toggle("active", active);
    source.panel.classList.toggle("active", active);
  });
}

function recordIntegration(state, integration) {
  const item = {
    type: integration.type || "service",
    mode: integration.mode || "real",
    target: integration.target || "",
    endpoint: integration.endpoint || "",
    method: integration.method || "",
    status: integration.status || "attempted",
    started_at: new Date().toISOString(),
    step_index: activeExecutionStepIndex,
    step: Number.isInteger(activeExecutionStepIndex) ? activeRuntimeWorkflow?.steps?.[activeExecutionStepIndex]?.name || "" : "",
    source_key: Number.isInteger(activeExecutionStepIndex) ? activeRuntimeWorkflow?.steps?.[activeExecutionStepIndex]?.__sourceWorkflow || runtimeRootSourceKey || "" : runtimeRootSourceKey || "",
    trace_id: state.trace_id || state.payload?.trace_id || "",
    correlation_id: state.correlation_id || "",
  };
  if (!state.metrics) state.metrics = { integrations: [] };
  if (!Array.isArray(state.metrics.integrations)) state.metrics.integrations = [];
  state.metrics.integrations.push(item);
  return item;
}

function resolveResumeCursor(state, workflow) {
  if (state.errors.length) {
    return Math.min(state.errors[0].cursor, workflow.steps.length);
  }
  if (state.stopped) {
    return Math.min(state.cursor, workflow.steps.length);
  }
  return workflow.steps.length;
}

function startStepGroup(stepIndex, step, totalSteps) {
  const group = ensureStepGroup(stepIndex, step, totalSteps, step?.__sourceWorkflow, step?.__sourceLabel);
  setActiveStepGroup(group);
}

function ensureStepGroup(stepIndex, step = null, totalSteps = null, sourceKey = null, sourceLabel = null) {
  const key = `${sourceKey || "workflow"}:${stepIndex}`;
  if (stepGroups.has(key)) return stepGroups.get(key);
  const source = ensureLogSource(sourceKey || "workflow", sourceLabel || sourceKey || "Workflow");
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
    focusWorkflowStepFromRuntime(stepIndex);
  });
  stepGroups.set(key, group);
  source.panel.appendChild(group);
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
  const sourceKey = entry.dataset.sourceKey || "workflow";
  const group = stepGroups.get(`${sourceKey}:${stepIndex}`);
  if (group) {
    if (activeStepGroup) activeStepGroup.classList.remove("phase-group--active");
    activeStepGroup = group;
    activeStepGroup.classList.add("phase-group--active");
  }
  focusWorkflowStepFromRuntime(stepIndex);
}

async function focusWorkflowStepFromRuntime(stepIndex) {
  const step = activeRuntimeWorkflow?.steps?.[stepIndex] || lastExecutionSnapshot?.workflow?.steps?.[stepIndex];
  const sourceKey = step?.__sourceWorkflow || runtimeRootSourceKey;
  const sourceStepIndex = Number.isInteger(step?.__sourceStepIndex) ? step.__sourceStepIndex : stepIndex;
  await openWorkflowSource(sourceKey);
  focusWorkflowStep(sourceStepIndex);
}

async function openWorkflowSource(sourceKey) {
  if (!sourceKey || !sourceKey.includes("/")) return;
  const [serviceName, fileName] = sourceKey.split("/");
  if (currentWorkspaceFile.serviceName === serviceName && currentWorkspaceFile.fileName === fileName) return;
  await openWorkflowFile(serviceName, fileName, { skipDirtyCheck: true, preserveLogs: true });
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
