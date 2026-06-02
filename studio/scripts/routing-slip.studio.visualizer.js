async function openWorkflowVisualizer() {
  const issues = lintWorkflow();
  if (issues.some((item) => item.level === "error")) {
    addLog("error", "Visualizacao bloqueada pelo lint", "Corrija os erros do workflow antes de visualizar.");
    return;
  }
  try {
    const workflow = parseWorkflow();
    workflowVisualizerGraph = {
      root: workflow,
      source: currentWorkspaceFile?.handle ? { ...currentWorkspaceFile } : null,
      workflows: await collectWorkflowVisualization(workflow, currentWorkspaceFile),
    };
    workflowVisualizerMode = "macro";
    els.visualizerModal.classList.add("active");
    els.visualizerModal.setAttribute("aria-hidden", "false");
    document.body.classList.add("visualizer-open");
    renderWorkflowVisualizer();
    setupWorkflowVisualizerPanZoom();
    renderIcons();
  } catch (err) {
    addLog("error", "Falha ao montar visualizacao", err.message);
  }
}

function closeWorkflowVisualizer() {
  els.visualizerModal.classList.remove("active");
  els.visualizerModal.setAttribute("aria-hidden", "true");
  document.body.classList.remove("visualizer-open");
}

function toggleWorkflowVisualizerMode() {
  workflowVisualizerMode = workflowVisualizerMode === "macro" ? "micro" : "macro";
  renderWorkflowVisualizer();
}

async function collectWorkflowVisualization(workflow, source, seen = new Set()) {
  const sourceKey = workflowSourceKey(source, workflow);
  if (seen.has(sourceKey)) return [];
  seen.add(sourceKey);
  const item = {
    key: sourceKey,
    label: workflowSourceLabel(source, workflow),
    workflow,
    source,
    refs: [],
    integrations: [],
  };
  for (const step of workflow.steps || []) {
    if (step.enabled === false) continue;
    if (step.name === "workflow_ref") {
      const ref = workflowRefFile(step);
      const resolved = resolveWorkspaceWorkflow(ref, source);
      const text = await WorkspaceFS.readFile(resolved.handle);
      const child = workflowScriptFromProjectText(text, resolved.serviceName, resolved.fileName);
      const childSource = { handle: resolved.handle, serviceName: resolved.serviceName, fileName: resolved.fileName };
      item.refs.push({
        step,
        key: workflowSourceKey(childSource, child),
        label: workflowSourceLabel(childSource, child),
      });
      const children = await collectWorkflowVisualization(child, childSource, seen);
      children.forEach((childItem) => {
        if (!item.__children) item.__children = [];
        item.__children.push(childItem);
      });
    }
    if (["graphql_enrich", "rest_call", "notify", "aws_action", "datadog_metric"].includes(step.name)) {
      item.integrations.push({ step, type: workflowIntegrationType(step), label: workflowIntegrationLabel(step) });
    }
  }
  return [item, ...(item.__children || [])];
}

function workflowIntegrationType(step) {
  if (step.name === "graphql_enrich") return "graphql";
  if (step.name === "rest_call") return "rest";
  if (step.name === "notify") return "notify";
  if (step.name === "aws_action") return "aws";
  if (step.name === "datadog_metric") return "metrics";
  return "service";
}

function workflowIntegrationLabel(step) {
  const params = step.params || {};
  if (step.name === "graphql_enrich") return params.target || "GraphQL";
  if (step.name === "rest_call") return params.target || params.endpoint || "REST API";
  if (step.name === "notify") return params.channel || params.target || "Notificacao";
  if (step.name === "aws_action") return `${params.service || "AWS"}:${params.action || "action"}`;
  if (step.name === "datadog_metric") return params.metric || "Datadog metric";
  return step.name;
}

function renderWorkflowVisualizer() {
  if (!workflowVisualizerGraph) return;
  const graph = workflowVisualizerMode === "macro"
    ? buildMacroVisualizerGraph(workflowVisualizerGraph)
    : buildMicroVisualizerGraph(workflowVisualizerGraph.root);
  els.visualizerTitle.textContent = `${workflowVisualizerGraph.root.name || "Workflow"} - ${workflowVisualizerMode === "macro" ? "Macro" : "Micro"}`;
  els.visualizerToggle.classList.toggle("micro", workflowVisualizerMode === "micro");
  els.visualizerToggle.setAttribute("aria-pressed", workflowVisualizerMode === "micro" ? "true" : "false");
  drawWorkflowGraph(graph);
  setupWorkflowVisualizerPanZoom();
}

function buildMacroVisualizerGraph(context) {
  const connector = context.root.trigger?.connector || context.root.connector || "rest";
  const nodes = [
    visualNode("start", "Inicio", "start", 60, 220),
    visualNode("connector", connector, "connector", 310, 220),
  ];
  const edges = [
    visualEdge("start", "connector"),
  ];
  const workflows = context.workflows;
  const columnStep = 360;
  workflows.forEach((item, index) => {
    const x = 620 + index * columnStep;
    const y = 220 + (index % 2 === 0 ? 0 : 120);
    const workflowNode = visualNode(item.key, item.label, "workflow", x, y);
    nodes.push(workflowNode);
    if (index === 0) {
      edges.push(visualEdge("connector", item.key, "evento"));
    }
    item.integrations.forEach((integration, integrationIndex) => {
      const integrationID = `${item.key}:integration:${integrationIndex}`;
      nodes.push(visualNode(integrationID, integration.label, integration.type, x, y + workflowNode.height + 130 + integrationIndex * 132));
      edges.push(visualEdge(item.key, integrationID, integration.type));
      edges.push(visualEdge(integrationID, item.key, "resultado", "dashed"));
    });
    item.refs.forEach((ref) => {
      edges.push(visualEdge(item.key, ref.key, "workflow_ref"));
    });
  });
  const last = workflows.at(-1)?.key || "connector";
  nodes.push(visualNode("return", "Retorno", "return", 700 + workflows.length * columnStep, 220));
  edges.push(visualEdge(last, "return"));
  return { nodes, edges };
}

function buildMicroVisualizerGraph(workflow) {
  const connector = workflow.trigger?.connector || workflow.connector || "rest";
  const nodes = [
    visualNode("start", "Inicio", "start", 60, 240),
    visualNode("connector", connector, "connector", 310, 240),
  ];
  const edges = [visualEdge("start", "connector")];
  let previous = "connector";
  const steps = (workflow.steps || []).filter((step) => step.enabled !== false);
  steps.forEach((step, index) => {
    const id = `step:${index}`;
    const kind = visualStepKind(step);
    const x = 620 + index * 310;
    const y = 240 + (index % 2 === 0 ? 0 : 98);
    const node = visualNode(id, visualStepLabel(step), kind, x, y);
    nodes.push(node);
    edges.push(visualEdge(previous, id, index === 0 ? "evento" : "", kind === "decision" ? "dashed" : "solid"));
    if (kind === "decision") {
      const failID = `step:${index}:fail`;
      nodes.push(visualNode(failID, "Retorno", "return", x, y + node.height + 142));
      edges.push(visualEdge(id, failID, "nao atende", "solid"));
    }
    previous = id;
  });
  nodes.push(visualNode("return", "Retorno", "return", 700 + steps.length * 310, 240));
  edges.push(visualEdge(previous, "return", "sucesso"));
  return { nodes, edges };
}

function visualStepKind(step) {
  if (["condition", "assert", "jump_if", "cel"].includes(step.name)) return "decision";
  if (step.name === "graphql_enrich") return "graphql";
  if (step.name === "rest_call") return "rest";
  if (step.name === "aws_action") return "aws";
  if (step.name === "datadog_metric") return "metrics";
  if (step.name === "workflow_ref") return "workflow";
  if (step.name === "notify") return "notify";
  if (["audit", "log"].includes(step.name)) return "audit";
  return "step";
}

function visualStepLabel(step) {
  if (step.id) return step.id;
  if (step.params?.event) return step.params.event;
  if (step.params?.target) return `${step.name}: ${step.params.target}`;
  if (step.name === "workflow_ref") return workflowRefFile(step);
  return step.name;
}

function visualNode(id, label, kind, x, y) {
  const lines = wrapVisualText(String(label || kind), kind === "decision" ? 18 : 22, kind === "decision" ? 3 : 2);
  const longest = lines.reduce((max, line) => Math.max(max, line.length), 0);
  const baseWidth = kind === "decision" ? 190 : 190;
  const width = Math.min(kind === "decision" ? 300 : 340, Math.max(baseWidth, longest * 8 + 44));
  const height = kind === "decision"
    ? Math.max(104, lines.length * 17 + 58)
    : Math.max(82, lines.length * 17 + 48);
  return { id, label, lines, kind, x, y, width, height };
}

function visualEdge(from, to, label = "", style = "solid") {
  return { from, to, label, style };
}

function drawWorkflowGraph(graph, options = {}) {
  const svg = els.visualizerSvg;
  workflowVisualizerRenderedGraph = graph;
  if (!options.preserveLayout) normalizeGraphLayout(graph);
  const maxX = Math.max(...graph.nodes.map((node) => node.x + node.width), 900) + 160;
  const maxY = Math.max(...graph.nodes.map((node) => node.y + node.height), 520) + 160;
  if (!options.preserveViewBox) {
    workflowVisualizerViewBox = { x: 0, y: 0, width: Math.min(1280, maxX), height: Math.min(720, maxY) };
  }
  svg.setAttribute("viewBox", `${workflowVisualizerViewBox.x} ${workflowVisualizerViewBox.y} ${workflowVisualizerViewBox.width} ${workflowVisualizerViewBox.height}`);
  svg.innerHTML = `
    <defs>
      <marker id="visual-arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse">
        <path d="M 0 0 L 10 5 L 0 10 z"></path>
      </marker>
    </defs>
    <g class="visualizer-viewport">
      ${graph.edges.map((edge, index) => drawVisualEdge(edge, graph.nodes, graph.edges, index)).join("")}
      ${graph.nodes.map(drawVisualNode).join("")}
    </g>
  `;
  bindWorkflowVisualizerNodeDrag();
}

function drawVisualNode(node) {
  const text = drawVisualTextLines(node.lines || [node.label], node.x + node.width / 2, node.y + node.height / 2 - ((node.lines?.length || 1) - 1) * 8);
  if (node.kind === "decision") {
    const cx = node.x + node.width / 2;
    const cy = node.y + node.height / 2;
    const points = `${cx},${node.y} ${node.x + node.width},${cy} ${cx},${node.y + node.height} ${node.x},${cy}`;
    return `
      <g class="visual-node visual-node-${node.kind}" data-node-id="${escapeHtml(node.id)}">
        <polygon points="${points}"></polygon>
        ${text}
        <text x="${cx}" y="${node.y + node.height - 22}" text-anchor="middle" class="visual-node-kind">decisao</text>
      </g>`;
  }
  return `
    <g class="visual-node visual-node-${node.kind}" data-node-id="${escapeHtml(node.id)}">
      <rect x="${node.x}" y="${node.y}" width="${node.width}" height="${node.height}" rx="12"></rect>
      ${text}
      <text x="${node.x + node.width / 2}" y="${node.y + node.height - 18}" text-anchor="middle" class="visual-node-kind">${escapeHtml(node.kind)}</text>
    </g>`;
}

function drawVisualEdge(edge, nodes, edges = [], edgeIndex = 0) {
  const from = nodes.find((node) => node.id === edge.from);
  const to = nodes.find((node) => node.id === edge.to);
  if (!from || !to) return "";
  const points = visualEdgeAnchors(from, to);
  const sibling = visualEdgeSiblingOffset(edge, edges, edgeIndex);
  const labelGap = edge.label ? Math.max(92, String(edge.label).length * 8 + 42) : 0;
  const horizontal = Math.abs(points.x2 - points.x1) >= Math.abs(points.y2 - points.y1);
  let mid = horizontal
    ? points.x1 + Math.sign(points.x2 - points.x1 || 1) * Math.max(labelGap, Math.abs(points.x2 - points.x1) / 2)
    : points.y1 + Math.sign(points.y2 - points.y1 || 1) * Math.max(labelGap, Math.abs(points.y2 - points.y1) / 2);
  mid += sibling.curveOffset;
  const path = horizontal
    ? `M ${points.x1} ${points.y1} C ${mid} ${points.y1}, ${mid} ${points.y2}, ${points.x2} ${points.y2}`
    : `M ${points.x1} ${points.y1} C ${points.x1} ${mid}, ${points.x2} ${mid}, ${points.x2} ${points.y2}`;
  const labelX = horizontal ? (points.x1 + points.x2) / 2 : (points.x1 + points.x2) / 2 + 36 + sibling.labelOffset;
  const labelY = horizontal ? (points.y1 + points.y2) / 2 - 14 + sibling.labelOffset : (points.y1 + points.y2) / 2 + sibling.labelOffset;
  return `
    <g class="visual-edge ${edge.style === "dashed" ? "visual-edge-dashed" : ""}">
      <path d="${path}"></path>
      ${edge.label ? `<text x="${labelX}" y="${labelY}" text-anchor="middle">${escapeHtml(edge.label)}</text>` : ""}
    </g>`;
}

function visualEdgeSiblingOffset(edge, edges, edgeIndex) {
  const samePair = edges
    .map((item, index) => ({ item, index }))
    .filter(({ item }) => item.from === edge.from && item.to === edge.to);
  if (samePair.length <= 1) return { curveOffset: 0, labelOffset: 0 };
  const position = samePair.findIndex(({ index }) => index === edgeIndex);
  const centered = position - (samePair.length - 1) / 2;
  return {
    curveOffset: centered * 34,
    labelOffset: centered * 24,
  };
}

function normalizeGraphLayout(graph) {
  graph.nodes.sort((a, b) => a.x - b.x || a.y - b.y);
  for (let i = 1; i < graph.nodes.length; i += 1) {
    const current = graph.nodes[i];
    for (let j = 0; j < i; j += 1) {
      const previous = graph.nodes[j];
      const overlapsX = current.x < previous.x + previous.width + 46 && current.x + current.width + 46 > previous.x;
      const overlapsY = current.y < previous.y + previous.height + 38 && current.y + current.height + 38 > previous.y;
      if (overlapsX && overlapsY) {
        current.y = previous.y + previous.height + 78;
      }
    }
  }
}

function visualEdgeAnchors(from, to) {
  const fromCenterX = from.x + from.width / 2;
  const fromCenterY = from.y + from.height / 2;
  const toCenterX = to.x + to.width / 2;
  const toCenterY = to.y + to.height / 2;
  const dx = toCenterX - fromCenterX;
  const dy = toCenterY - fromCenterY;
  if (Math.abs(dx) >= Math.abs(dy)) {
    return dx >= 0
      ? { x1: from.x + from.width, y1: fromCenterY, x2: to.x, y2: toCenterY }
      : { x1: from.x, y1: fromCenterY, x2: to.x + to.width, y2: toCenterY };
  }
  return dy >= 0
    ? { x1: fromCenterX, y1: from.y + from.height, x2: toCenterX, y2: to.y }
    : { x1: fromCenterX, y1: from.y, x2: toCenterX, y2: to.y + to.height };
}

function wrapVisualText(value, maxChars = 22, maxLines = 2) {
  const words = String(value || "").replace(/[_./-]+/g, " ").split(/\s+/).filter(Boolean);
  if (!words.length) return [""];
  const lines = [];
  let current = "";
  words.forEach((word) => {
    if (!current) {
      current = word;
      return;
    }
    if ((current + " " + word).length <= maxChars) {
      current += " " + word;
      return;
    }
    lines.push(current);
    current = word;
  });
  if (current) lines.push(current);
  const clipped = lines.slice(0, maxLines);
  if (lines.length > maxLines) clipped[maxLines - 1] = `${clipped[maxLines - 1].replace(/\.+$/, "")}...`;
  return clipped;
}

function drawVisualTextLines(lines, x, y) {
  return (lines || [""]).map((line, index) => {
    const dy = index === 0 ? 0 : 17;
    return `<text x="${x}" y="${y + dy}" text-anchor="middle">${escapeHtml(line)}</text>`;
  }).join("");
}

function setupWorkflowVisualizerPanZoom() {
  const svg = els.visualizerSvg;
  svg.onpointerdown = (event) => {
    if (event.target.closest?.(".visual-node")) return;
    workflowVisualizerDrag = { x: event.clientX, y: event.clientY, viewBox: { ...workflowVisualizerViewBox } };
    svg.setPointerCapture(event.pointerId);
  };
  svg.onpointermove = (event) => {
    if (!workflowVisualizerDrag) return;
    if (workflowVisualizerDrag.type === "node") {
      const node = workflowVisualizerRenderedGraph?.nodes.find((item) => item.id === workflowVisualizerDrag.nodeId);
      if (!node) return;
      const scaleX = workflowVisualizerViewBox.width / Math.max(1, svg.clientWidth);
      const scaleY = workflowVisualizerViewBox.height / Math.max(1, svg.clientHeight);
      node.x = workflowVisualizerDrag.nodeStartX + (event.clientX - workflowVisualizerDrag.x) * scaleX;
      node.y = workflowVisualizerDrag.nodeStartY + (event.clientY - workflowVisualizerDrag.y) * scaleY;
      drawWorkflowGraph(workflowVisualizerRenderedGraph, { preserveViewBox: true, preserveLayout: true });
      return;
    }
    const scaleX = workflowVisualizerViewBox.width / Math.max(1, svg.clientWidth);
    const scaleY = workflowVisualizerViewBox.height / Math.max(1, svg.clientHeight);
    workflowVisualizerViewBox.x = workflowVisualizerDrag.viewBox.x - (event.clientX - workflowVisualizerDrag.x) * scaleX;
    workflowVisualizerViewBox.y = workflowVisualizerDrag.viewBox.y - (event.clientY - workflowVisualizerDrag.y) * scaleY;
    applyWorkflowVisualizerViewBox();
  };
  svg.onpointerup = () => {
    workflowVisualizerDrag = null;
  };
  svg.onwheel = (event) => {
    event.preventDefault();
    const factor = event.deltaY > 0 ? 1.12 : 0.88;
    const rect = svg.getBoundingClientRect();
    const mouseX = workflowVisualizerViewBox.x + ((event.clientX - rect.left) / rect.width) * workflowVisualizerViewBox.width;
    const mouseY = workflowVisualizerViewBox.y + ((event.clientY - rect.top) / rect.height) * workflowVisualizerViewBox.height;
    workflowVisualizerViewBox.width *= factor;
    workflowVisualizerViewBox.height *= factor;
    workflowVisualizerViewBox.x = mouseX - ((event.clientX - rect.left) / rect.width) * workflowVisualizerViewBox.width;
    workflowVisualizerViewBox.y = mouseY - ((event.clientY - rect.top) / rect.height) * workflowVisualizerViewBox.height;
    applyWorkflowVisualizerViewBox();
  };
}

function resetWorkflowVisualizerView() {
  renderWorkflowVisualizer();
}

function applyWorkflowVisualizerViewBox() {
  els.visualizerSvg.setAttribute("viewBox", `${workflowVisualizerViewBox.x} ${workflowVisualizerViewBox.y} ${workflowVisualizerViewBox.width} ${workflowVisualizerViewBox.height}`);
}

function bindWorkflowVisualizerNodeDrag() {
  els.visualizerSvg.querySelectorAll(".visual-node").forEach((nodeEl) => {
    nodeEl.addEventListener("pointerdown", (event) => {
      event.stopPropagation();
      const node = workflowVisualizerRenderedGraph?.nodes.find((item) => item.id === nodeEl.dataset.nodeId);
      if (!node) return;
      workflowVisualizerDrag = {
        type: "node",
        nodeId: node.id,
        x: event.clientX,
        y: event.clientY,
        nodeStartX: node.x,
        nodeStartY: node.y,
      };
      els.visualizerSvg.setPointerCapture(event.pointerId);
    });
  });
}

async function downloadWorkflowDiagramImage() {
  const svg = els.visualizerSvg;
  if (!svg || !workflowVisualizerRenderedGraph) return;
  const bounds = workflowDiagramBounds(workflowVisualizerRenderedGraph, 100);
  const exportScale = 3;
  const width = Math.min(8192, Math.max(2400, Math.ceil(bounds.width * exportScale)));
  const height = Math.min(8192, Math.max(1400, Math.ceil(bounds.height * exportScale)));
  const clone = svg.cloneNode(true);
  clone.setAttribute("xmlns", "http://www.w3.org/2000/svg");
  clone.setAttribute("viewBox", `${bounds.x} ${bounds.y} ${bounds.width} ${bounds.height}`);
  clone.setAttribute("width", String(width));
  clone.setAttribute("height", String(height));
  clone.insertBefore(workflowDiagramExportStyle(), clone.firstChild);
  const source = new XMLSerializer().serializeToString(clone);
  const blob = new Blob([source], { type: "image/svg+xml;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const image = new Image();
  image.onload = () => {
    const canvas = document.createElement("canvas");
    canvas.width = Number(clone.getAttribute("width"));
    canvas.height = Number(clone.getAttribute("height"));
    const ctx = canvas.getContext("2d");
    ctx.fillStyle = computedCSSVar("--surface-3", "#fbfdff");
    ctx.fillRect(0, 0, canvas.width, canvas.height);
    ctx.drawImage(image, 0, 0, canvas.width, canvas.height);
    URL.revokeObjectURL(url);
    const link = document.createElement("a");
    link.download = `${cleanWorkflowRefPrefix(workflowVisualizerGraph.root.name || "workflow")}-${workflowVisualizerMode}.png`;
    link.href = canvas.toDataURL("image/png");
    link.click();
  };
  image.onerror = () => {
    URL.revokeObjectURL(url);
    const link = document.createElement("a");
    link.download = `${cleanWorkflowRefPrefix(workflowVisualizerGraph.root.name || "workflow")}-${workflowVisualizerMode}.svg`;
    link.href = URL.createObjectURL(blob);
    link.click();
  };
  image.src = url;
}

function workflowDiagramExportStyle() {
  const style = document.createElementNS("http://www.w3.org/2000/svg", "style");
  const text = computedCSSVar("--text", "#182230");
  const muted = computedCSSVar("--muted", "#64748b");
  const surface = computedCSSVar("--surface", "#ffffff");
  const surface2 = computedCSSVar("--surface-2", "#eef2f7");
  const accent = computedCSSVar("--accent", "#0f766e");
  style.textContent = `
    .visual-edge path{fill:none;stroke:${muted};stroke-width:2.2;marker-end:url(#visual-arrow)}
    .visual-edge-dashed path{stroke-dasharray:7 6}
    .visual-edge text{fill:${muted};dominant-baseline:middle;font-size:12px;font-weight:800;font-family:Inter,Arial,sans-serif}
    #visual-arrow path{fill:${muted}}
    .visual-node rect,.visual-node polygon{stroke:${accent};stroke-width:1.3;fill:${surface}}
    .visual-node text{fill:${text};font-size:13px;font-weight:900;dominant-baseline:middle;font-family:Inter,Arial,sans-serif}
    .visual-node .visual-node-kind{fill:${muted};font-size:10px;font-weight:800;text-transform:uppercase}
    .visual-node-start rect{fill:#dcfce7}
    .visual-node-connector rect{fill:#dbeafe}
    .visual-node-workflow rect{fill:#ccfbf1}
    .visual-node-graphql rect,.visual-node-rest rect,.visual-node-notify rect,.visual-node-aws rect,.visual-node-metrics rect{fill:#fef3c7}
    .visual-node-decision polygon{fill:#fee2e2}
    .visual-node-return rect,.visual-node-audit rect{fill:${surface2}}
  `;
  return style;
}

function workflowDiagramBounds(graph, padding = 80) {
  const minX = Math.min(...graph.nodes.map((node) => node.x));
  const minY = Math.min(...graph.nodes.map((node) => node.y));
  const maxX = Math.max(...graph.nodes.map((node) => node.x + node.width));
  const maxY = Math.max(...graph.nodes.map((node) => node.y + node.height));
  return {
    x: minX - padding,
    y: minY - padding,
    width: Math.max(1, maxX - minX + padding * 2),
    height: Math.max(1, maxY - minY + padding * 2),
  };
}

function computedCSSVar(name, fallback) {
  return getComputedStyle(document.body).getPropertyValue(name).trim() || fallback;
}
