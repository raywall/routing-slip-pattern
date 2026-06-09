function renderExecutionSummary(workflow, state, status) {
  const rootKey = workflow?.__sourceWorkflow || runtimeRootSourceKey || "workflow";
  const sources = new Map([[rootKey, { label: workflow?.__sourceLabel || workflow?.name || "Workflow", root: true }]]);
  state.history.forEach((item) => {
    if (item.sourceKey && item.sourceKey !== rootKey) sources.set(item.sourceKey, { label: item.sourceLabel || item.sourceKey, root: false });
  });
  state.errors.forEach((item) => {
    if (item.sourceKey && item.sourceKey !== rootKey) {
      const step = workflow.steps[item.cursor] || {};
      sources.set(item.sourceKey, { label: step.__sourceLabel || item.sourceKey, root: false });
    }
  });
  (state.metrics?.integrations || []).forEach((item) => {
    if (item.source_key && item.source_key !== rootKey) {
      const step = workflow.steps[item.step_index] || {};
      sources.set(item.source_key, { label: step.__sourceLabel || item.source_key, root: false });
    }
  });
  sources.forEach((source, sourceKey) => renderExecutionSummaryForSource(workflow, state, status, sourceKey, source.label, source.root));
}

function renderExecutionSummaryForSource(workflow, state, status, sourceKey, sourceLabel, isRoot) {
  const sourceHistory = isRoot ? state.history : state.history.filter((item) => item.sourceKey === sourceKey);
  const totalDuration = isRoot ? state.metrics?.total_duration_ms || 0 : sourceHistory.reduce((sum, item) => sum + (item.duration_ms || 0), 0);
  const previousDuration = state.reprocess?.previousDurationMs;
  const durationDelta = isRoot && typeof previousDuration === "number" ? totalDuration - previousDuration : null;
  const integrations = (state.metrics?.integrations || []).filter((item) => isRoot || item.source_key === sourceKey);
  const byType = integrations.reduce((acc, item) => {
    acc[item.type] = (acc[item.type] || 0) + 1;
    return acc;
  }, {});
  const realCount = integrations.filter((item) => item.mode === "real").length;
  const simulatedCount = integrations.filter((item) => item.mode === "simulated").length;
  const errorCount = isRoot ? state.errors.length : state.errors.filter((item) => item.sourceKey === sourceKey).length;
  const skippedCount = sourceHistory.filter((item) => item.skipped).length;
  const executedCount = sourceHistory.length;
  const preservedCount = isRoot ? state.reprocess?.previousCursor || 0 : 0;
  const avgStepDuration = executedCount ? Math.round(sourceHistory.reduce((sum, item) => sum + (item.duration_ms || 0), 0) / executedCount) : 0;

  const summary = document.createElement("section");
  summary.className = "execution-summary";
  summary.innerHTML = `
    <div class="execution-summary-head">
      <div>
        <p class="execution-summary-eyebrow">Resumo da execucao</p>
        <h3>${escapeHtml(sourceLabel || workflow.name || "Workflow")} ${escapeHtml(status)}</h3>
      </div>
      <span>${escapeHtml(formatDuration(totalDuration))}</span>
    </div>
    <div class="execution-summary-grid">
      ${summaryMetric("Etapas", "executadas", executedCount)}
      ${state.reprocess ? summaryMetric("Etapas", "preservadas", preservedCount) : ""}
      ${summaryMetric("Etapas", "saltos e paradas", skippedCount)}
      ${summaryMetric("Erros", "e falhas", errorCount)}
      ${summaryMetric("Tempo médio", "por etapa", formatDuration(avgStepDuration))}
      ${durationDelta === null ? "" : summaryMetric("Diferença", "do tempo anterior", formatDurationDelta(durationDelta))}
      ${summaryMetric("Integrações", "realizadas", integrations.length)}
      ${summaryMetric("Integrações", "reais", realCount)}
      ${summaryMetric("Integrações", "simuladas", simulatedCount)}
      ${summaryMetric("Integrações", "GraphQL", byType.graphql || 0)}
      ${summaryMetric("Integrações", "REST", byType.rest || 0)}
      ${summaryMetric("Notificações", "e alertas", byType.notify || 0)}
    </div>
    <div class="execution-summary-identifiers">
      ${summaryIdentifier("Trace ID", state.trace_id || "-")}
      ${summaryIdentifier("Correlation ID", state.correlation_id || "-")}
    </div>
    ${integrations.length ? `
      <div class="execution-summary-integrations">
        <h4>Integrações acionadas</h4>
        ${integrations.map((item) => `
          <div class="integration-row">
            <span>${escapeHtml(item.type)}</span>
            <strong>${escapeHtml(item.status)}</strong>
            <em>${escapeHtml([item.method, item.target || item.endpoint, item.trace_id ? `trace:${item.trace_id}` : ""].filter(Boolean).join(" "))}</em>
          </div>
        `).join("")}
      </div>
    ` : ""}
  `;
  const source = ensureLogSource(sourceKey, sourceLabel);
  source.panel.appendChild(summary);
  source.panel.scrollTop = source.panel.scrollHeight;
}

function summaryMetric(title, subtitle, value) {
  return `
    <div class="summary-metric">
      <span class="summary-metric-title">${escapeHtml(title)}</span>
      <span class="summary-metric-subtitle">${escapeHtml(subtitle)}</span>
      <strong>${escapeHtml(String(value))}</strong>
    </div>
  `;
}

function summaryIdentifier(label, value) {
  return `
    <label class="summary-identifier">
      <span>${escapeHtml(label)}</span>
      <input type="text" readonly value="${escapeHtml(String(value))}" />
    </label>
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
