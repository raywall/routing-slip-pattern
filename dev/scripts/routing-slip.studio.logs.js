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

function recordIntegration(state, integration) {
  const item = {
    type: integration.type || "service",
    mode: integration.mode || "real",
    target: integration.target || "",
    endpoint: integration.endpoint || "",
    method: integration.method || "",
    status: integration.status || "attempted",
    started_at: new Date().toISOString(),
  };
  if (!state.metrics) state.metrics = { integrations: [] };
  if (!Array.isArray(state.metrics.integrations)) state.metrics.integrations = [];
  state.metrics.integrations.push(item);
  return item;
}

function renderExecutionSummary(workflow, state, status) {
  const totalDuration = state.metrics?.total_duration_ms || 0;
  const previousDuration = state.reprocess?.previousDurationMs;
  const durationDelta = typeof previousDuration === "number" ? totalDuration - previousDuration : null;
  const integrations = state.metrics?.integrations || [];
  const byType = integrations.reduce((acc, item) => {
    acc[item.type] = (acc[item.type] || 0) + 1;
    return acc;
  }, {});
  const realCount = integrations.filter((item) => item.mode === "real").length;
  const simulatedCount = integrations.filter((item) => item.mode === "simulated").length;
  const errorCount = state.errors.length;
  const skippedCount = state.history.filter((item) => item.skipped).length;
  const executedCount = state.history.length;
  const preservedCount = state.reprocess?.previousCursor || 0;
  const avgStepDuration = executedCount ? Math.round(state.history.reduce((sum, item) => sum + (item.duration_ms || 0), 0) / executedCount) : 0;

  const summary = document.createElement("section");
  summary.className = "execution-summary";
  summary.innerHTML = `
    <div class="execution-summary-head">
      <div>
        <p class="execution-summary-eyebrow">Resumo da execucao</p>
        <h3>${escapeHtml(workflow.name || "Workflow")} ${escapeHtml(status)}</h3>
      </div>
      <span>${escapeHtml(formatDuration(totalDuration))}</span>
    </div>
    <div class="execution-summary-grid">
      ${summaryMetric("Steps executados", executedCount)}
      ${state.reprocess ? summaryMetric("Steps preservados", preservedCount) : ""}
      ${summaryMetric("Steps pulados/parados", skippedCount)}
      ${summaryMetric("Erros", errorCount)}
      ${summaryMetric("Tempo medio por step", formatDuration(avgStepDuration))}
      ${durationDelta === null ? "" : summaryMetric("Dif. tempo anterior", formatDurationDelta(durationDelta))}
      ${summaryMetric("Integracoes API/servico", integrations.length)}
      ${summaryMetric("Reais", realCount)}
      ${summaryMetric("Simuladas", simulatedCount)}
      ${summaryMetric("GraphQL", byType.graphql || 0)}
      ${summaryMetric("REST", byType.rest || 0)}
      ${summaryMetric("Notify", byType.notify || 0)}
    </div>
    ${integrations.length ? `
      <div class="execution-summary-integrations">
        <h4>Integracoes acionadas</h4>
        ${integrations.map((item) => `
          <div class="integration-row">
            <span>${escapeHtml(item.type)}</span>
            <strong>${escapeHtml(item.status)}</strong>
            <em>${escapeHtml([item.method, item.target || item.endpoint].filter(Boolean).join(" "))}</em>
          </div>
        `).join("")}
      </div>
    ` : ""}
  `;
  els.timeline.appendChild(summary);
  els.timeline.scrollTop = els.timeline.scrollHeight;
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

function summaryMetric(label, value) {
  return `
    <div class="summary-metric">
      <span>${escapeHtml(label)}</span>
      <strong>${escapeHtml(String(value))}</strong>
    </div>
  `;
}

function formatDuration(ms) {
  const value = Number(ms) || 0;
  if (value < 1000) return `${value} ms`;
  return `${(value / 1000).toFixed(value < 10000 ? 2 : 1)} s`;
}

function formatDurationDelta(ms) {
  const value = Number(ms) || 0;
  const sign = value > 0 ? "+" : value < 0 ? "-" : "";
  return `${sign}${formatDuration(Math.abs(value))}`;
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
