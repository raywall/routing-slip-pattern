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
  insertTabIndent(event.currentTarget, { outdent: event.shiftKey });
  lintWorkflow();
  markWorkflowDirty();
  scheduleStudioSave();
}

function insertTabIndent(editor, options = {}) {
  const start = editor.selectionStart;
  const end = editor.selectionEnd;
  const value = editor.value;
  const indent = "  ";
  const outdent = Boolean(options.outdent);

  if (start !== end && value.slice(start, end).includes("\n")) {
    const lineStart = value.lastIndexOf("\n", start - 1) + 1;
    const selected = value.slice(lineStart, end);
    const replacement = selected.split("\n").map((line) => {
      if (!outdent) return indent + line;
      if (line.startsWith(indent)) return line.slice(indent.length);
      if (line.startsWith(" ")) return line.slice(1);
      return line;
    }).join("\n");
    editor.value = value.slice(0, lineStart) + replacement + value.slice(end);
    editor.selectionStart = lineStart;
    editor.selectionEnd = lineStart + replacement.length;
  } else if (outdent) {
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
  invalidateExecutionSnapshot();
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
  const maxLineLength = Math.max(1, ...els.workflow.value.split("\n").map((line) => line.length));
  els.highlight.style.setProperty("--yaml-line-width", `${maxLineLength + 4}ch`);
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


function formatWorkflow() {
  try {
    parseWorkflow();
    els.workflow.value = formatWorkflowYamlForEditor(els.workflow.value);
    markWorkflowDirty();
    lintWorkflow();
    scheduleStudioSave();
  } catch (err) {
    addLog("error", "Nao foi possivel organizar YAML", err.message);
  }
}

function formatWorkflowYamlForEditor(yaml) {
  const lines = String(yaml || "").replace(/\s+$/g, "").split("\n");
  const formatted = [];
  let inSteps = false;
  let stepsIndent = 0;

  lines.forEach((line, index) => {
    const trimmed = line.trim();
    const indent = line.match(/^\s*/)[0].length;
    const isSteps = /^steps\s*:/.test(trimmed);

    if (isSteps) {
      if (formatted.length && formatted[formatted.length - 1].trim() !== "") {
        formatted.push("");
      }
      formatted.push(line);
      inSteps = true;
      stepsIndent = indent;
      return;
    }

    if (inSteps && trimmed && indent <= stepsIndent && !trimmed.startsWith("- ")) {
      inSteps = false;
    }

    if (inSteps && trimmed === "" && formatted.length && formatted[formatted.length - 1].trim().startsWith("#") && nextContentIsTopLevelStep(lines, index, stepsIndent)) {
      return;
    }

    const isStepComment = inSteps && isCommentBlockStartForStep(lines, index, stepsIndent);
    const isTopLevelStep = inSteps && /^-\s+(id|name)\s*:/.test(trimmed) && indent > stepsIndent;
    const previousTrimmed = formatted.length ? formatted[formatted.length - 1].trim() : "";
    if (isStepComment && previousTrimmed !== "" && !/^steps\s*:/.test(previousTrimmed) && !previousTrimmed.startsWith("#")) {
      formatted.push("");
    }
    if (isTopLevelStep && previousTrimmed !== "" && !/^steps\s*:/.test(previousTrimmed) && !previousTrimmed.startsWith("#")) {
      formatted.push("");
    }
    formatted.push(line);
  });

  return formatted.join("\n") + "\n";
}

function nextContentIsTopLevelStep(lines, index, stepsIndent) {
  for (let cursor = index + 1; cursor < lines.length; cursor += 1) {
    const line = lines[cursor] || "";
    const trimmed = line.trim();
    if (trimmed === "") continue;
    const indent = line.match(/^\s*/)[0].length;
    return indent > stepsIndent && /^-\s+(id|name)\s*:/.test(trimmed);
  }
  return false;
}

function isCommentBlockStartForStep(lines, index, stepsIndent) {
  const line = lines[index] || "";
  if (!line.trim().startsWith("#")) return false;
  if (index > 0 && (lines[index - 1] || "").trim().startsWith("#")) return false;
  for (let cursor = index + 1; cursor < lines.length; cursor += 1) {
    const next = lines[cursor] || "";
    const trimmed = next.trim();
    if (trimmed === "" || trimmed.startsWith("#")) continue;
    const indent = next.match(/^\s*/)[0].length;
    return indent > stepsIndent && /^-\s+(id|name)\s*:/.test(trimmed);
  }
  return false;
}
