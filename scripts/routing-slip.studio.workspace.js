const WorkspaceFS = {
  supported: typeof window.showDirectoryPicker === "function",

  async ensurePermission(handle) {
    if (!handle) return false;
    const options = { mode: "readwrite" };
    if (typeof handle.queryPermission === "function") {
      const current = await handle.queryPermission(options);
      if (current === "granted") return true;
    }
    if (typeof handle.requestPermission === "function") {
      try {
        return await handle.requestPermission(options) === "granted";
      } catch {
        return false;
      }
    }
    return true;
  },

  async buildTree(rootHandle) {
    const expanded = new Set(loadExpandedServices());
    const services = [];
    for await (const [name, handle] of rootHandle.entries()) {
      if (name.startsWith(".") || name === "node_modules") continue;
      if (handle.kind !== "directory") continue;
      const files = [];
      for await (const [fileName, fileHandle] of handle.entries()) {
        if (fileHandle.kind === "file" && /\.(ya?ml)$/i.test(fileName)) {
          files.push({ name: fileName, handle: fileHandle });
        }
      }
      files.sort((a, b) => a.name.localeCompare(b.name));
      services.push({ name, handle, expanded: expanded.has(name), files });
    }
    services.sort((a, b) => a.name.localeCompare(b.name));
    return services;
  },

  async readFile(fileHandle) {
    const file = await fileHandle.getFile();
    return file.text();
  },

  async writeFile(fileHandle, content) {
    const writable = await fileHandle.createWritable();
    await writable.write(content);
    await writable.close();
  },

  async createFile(dirHandle, fileName, content) {
    const fileHandle = await dirHandle.getFileHandle(fileName, { create: true });
    await this.writeFile(fileHandle, content);
    return fileHandle;
  },
};

async function openWorkspace() {
  if (!WorkspaceFS.supported) {
    alert("Workspace local requer Chrome ou Edge com suporte a File System Access API.");
    return;
  }
  try {
    const handle = await window.showDirectoryPicker({ mode: "readwrite" });
    workspaceState.rootHandle = handle;
    workspaceState.name = handle.name;
    workspaceState.services = await WorkspaceFS.buildTree(handle);
    await StudioDB.set(STUDIO_DB.workspaceHandleKey, handle);
    renderWorkspace();
  } catch (err) {
    if (err.name !== "AbortError") alert(`Erro ao abrir workspace: ${err.message}`);
  }
}

async function restoreWorkspace() {
  if (!WorkspaceFS.supported) {
    renderWorkspace();
    return;
  }
  const handle = await StudioDB.get(STUDIO_DB.workspaceHandleKey);
  if (!handle) {
    renderWorkspace();
    return;
  }
  const allowed = await WorkspaceFS.ensurePermission(handle);
  if (!allowed) {
    renderWorkspace();
    return;
  }
  try {
    workspaceState.rootHandle = handle;
    workspaceState.name = handle.name;
    workspaceState.services = await WorkspaceFS.buildTree(handle);
    renderWorkspace();
    const current = await StudioDB.get(STUDIO_DB.currentFileKey);
    if (current?.serviceName && current?.fileName) {
      await openWorkflowFile(current.serviceName, current.fileName, { skipDirtyCheck: true });
    }
  } catch (err) {
    console.warn("Nao foi possivel restaurar workspace:", err);
    renderWorkspace();
  }
}

async function refreshWorkspace() {
  if (!workspaceState.rootHandle) return;
  saveExpandedServices();
  workspaceState.services = await WorkspaceFS.buildTree(workspaceState.rootHandle);
  if (currentWorkspaceFile.serviceName && currentWorkspaceFile.fileName) {
    const service = findService(currentWorkspaceFile.serviceName);
    const file = service?.files.find((item) => item.name === currentWorkspaceFile.fileName);
    currentWorkspaceFile.handle = file?.handle || null;
  }
  renderWorkspace();
}

function renderWorkspace() {
  els.workspaceName.textContent = workspaceState.name || "Nenhum diretorio";
  const enabled = Boolean(workspaceState.rootHandle);
  [els.newService, els.refreshWorkspace].forEach((button) => {
    button.disabled = !enabled;
  });
  els.newWorkflow.disabled = !enabled || workspaceState.services.length === 0;
  els.saveWorkflowFile.disabled = !currentWorkspaceFile.handle || !workflowDirty;
  els.exportWorkflowFile.disabled = !currentWorkspaceFile.handle;

  if (!enabled) {
    els.workspaceTree.innerHTML = `
      <div class="workspace-empty">
        <div>Nenhum workspace aberto</div>
        <button type="button" data-open-empty>Abrir pasta</button>
      </div>`;
    els.workspaceTree.querySelector("[data-open-empty]")?.addEventListener("click", openWorkspace);
    els.workspaceCurrent.textContent = WorkspaceFS.supported
      ? "Abra um diretorio para organizar microservicos e workflows YAML."
      : "Seu navegador nao suporta workspace local.";
    return;
  }

  if (!workspaceState.services.length) {
    els.workspaceTree.innerHTML = `
      <div class="workspace-empty">
        <div>Workspace vazio</div>
        <small>Crie um microservico para comecar.</small>
      </div>`;
  } else {
    els.workspaceTree.innerHTML = workspaceState.services.map(renderServiceNode).join("");
    bindWorkspaceTree();
    renderIcons();
  }

  const path = currentWorkspaceFile.serviceName && currentWorkspaceFile.fileName
    ? `${workspaceState.name}/${currentWorkspaceFile.serviceName}/${currentWorkspaceFile.fileName}${workflowDirty ? " *" : ""}`
    : "Nenhum workflow aberto.";
  els.workspaceCurrent.textContent = path;
}

function renderServiceNode(service) {
  const active = currentWorkspaceFile.serviceName === service.name;
  const files = service.expanded ? service.files.map((file) => renderWorkflowFile(service.name, file)).join("") : "";
  return `
    <div class="workspace-service ${active ? "active" : ""}">
      <button class="workspace-service-head" type="button" data-service="${escapeHtml(service.name)}">
        <span>${service.expanded ? "▾" : "▸"}</span>
        <i class="workspace-icon" data-lucide="folder"></i>
        <span class="workspace-name">${escapeHtml(service.name)}</span>
        <span class="workspace-count">${service.files.length}</span>
      </button>
      <div>${files}</div>
    </div>`;
}

function renderWorkflowFile(serviceName, file) {
  const active = currentWorkspaceFile.serviceName === serviceName && currentWorkspaceFile.fileName === file.name;
  const label = file.name.replace(/\.(ya?ml)$/i, "");
  return `
    <button class="workspace-file ${active ? "active" : ""}" type="button" data-service="${escapeHtml(serviceName)}" data-file="${escapeHtml(file.name)}">
      <i class="workspace-icon" data-lucide="file"></i>
      <span class="workspace-file-name">${escapeHtml(label)}</span>
      <span class="workspace-dirty">${active && workflowDirty ? "●" : ""}</span>
    </button>`;
}

function bindWorkspaceTree() {
  els.workspaceTree.querySelectorAll(".workspace-service-head").forEach((button) => {
    button.addEventListener("click", () => toggleService(button.dataset.service));
    button.addEventListener("contextmenu", (event) => {
      event.preventDefault();
      showContextMenu(event.clientX, event.clientY, [
        { label: "Novo workflow", action: () => createWorkflow(button.dataset.service) },
        { label: "Renomear microservico", action: () => renameService(button.dataset.service) },
        { separator: true },
        { label: "Excluir microservico", action: () => deleteService(button.dataset.service), danger: true },
      ]);
    });
  });
  els.workspaceTree.querySelectorAll(".workspace-file").forEach((button) => {
    button.addEventListener("click", () => openWorkflowFile(button.dataset.service, button.dataset.file));
    button.addEventListener("contextmenu", (event) => {
      event.preventDefault();
      showContextMenu(event.clientX, event.clientY, [
        { label: "Abrir", action: () => openWorkflowFile(button.dataset.service, button.dataset.file) },
        { label: "Salvar", action: () => saveCurrentWorkflowFile() },
        { label: "Renomear workflow", action: () => renameWorkflow(button.dataset.service, button.dataset.file) },
        { separator: true },
        { label: "Excluir workflow", action: () => deleteWorkflow(button.dataset.service, button.dataset.file), danger: true },
      ]);
    });
  });
}

function toggleService(serviceName) {
  const service = findService(serviceName);
  if (!service) return;
  service.expanded = !service.expanded;
  saveExpandedServices();
  renderWorkspace();
}

async function openWorkflowFile(serviceName, fileName, options = {}) {
  if (workflowDirty && !options.skipDirtyCheck) {
    const save = confirm(`"${currentWorkspaceFile.fileName}" tem alteracoes nao salvas. Salvar antes de continuar?`);
    if (save) await saveCurrentWorkflowFile();
  }
  const service = findService(serviceName);
  const file = service?.files.find((item) => item.name === fileName);
  if (!file) return;
  try {
    invalidateExecutionSnapshot();
    els.workflow.value = await WorkspaceFS.readFile(file.handle);
    currentWorkspaceFile = { handle: file.handle, serviceName, fileName };
    workflowDirty = false;
    service.expanded = true;
    await StudioDB.set(STUDIO_DB.currentFileKey, { serviceName, fileName });
    renderWorkspace();
    clearLogs();
    lintWorkflow();
    scheduleStudioSave();
  } catch (err) {
    alert(`Erro ao abrir workflow: ${err.message}`);
  }
}

async function saveCurrentWorkflowFile() {
  if (!currentWorkspaceFile.handle) {
    alert("Abra ou crie um workflow no workspace antes de salvar.");
    return;
  }
  try {
    await WorkspaceFS.writeFile(currentWorkspaceFile.handle, els.workflow.value);
    workflowDirty = false;
    renderWorkspace();
    scheduleStudioSave();
  } catch (err) {
    alert(`Erro ao salvar workflow: ${err.message}`);
  }
}

async function exportComposedWorkflow() {
  if (!currentWorkspaceFile.handle) {
    alert("Abra um workflow no workspace antes de exportar.");
    return;
  }
  const issues = lintWorkflow();
  if (issues.some((item) => item.level === "error")) {
    alert("Corrija os erros do workflow antes de exportar.");
    return;
  }
  try {
    const expanded = await expandWorkflowRefsForStudio(lastWorkflow);
    const exportable = stripStudioRuntimeFields(expanded);
    const yaml = window.jsyaml.dump(exportable, {
      lineWidth: 140,
      noRefs: true,
      sortKeys: false,
    });
    const baseName = (currentWorkspaceFile.fileName || "workflow.yaml").replace(/\.(ya?ml)$/i, "");
    downloadTextFile(`${baseName}-bundle.yaml`, yaml);
  } catch (err) {
    alert(`Erro ao exportar workflow composto: ${err.message}`);
  }
}

function stripStudioRuntimeFields(workflow) {
  const clean = structuredClone(workflow);
  clean.steps = (clean.steps || []).map((step) => {
    const next = { ...step };
    delete next.__sourceWorkflow;
    if (next.params && typeof next.params === "object") {
      next.params = stripPrivateFields(next.params);
    }
    return next;
  });
  return clean;
}

function stripPrivateFields(value) {
  if (Array.isArray(value)) return value.map(stripPrivateFields);
  if (!value || typeof value !== "object") return value;
  return Object.fromEntries(
    Object.entries(value)
      .filter(([key]) => !key.startsWith("__"))
      .map(([key, item]) => [key, stripPrivateFields(item)])
  );
}

function downloadTextFile(fileName, content) {
  const blob = new Blob([content], { type: "text/yaml;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = fileName;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

async function createService() {
  if (!workspaceState.rootHandle) return;
  const raw = prompt("Nome do microservico:");
  if (!raw?.trim()) return;
  const name = normalizeName(raw);
  if (findService(name)) {
    alert(`O microservico "${name}" ja existe.`);
    return;
  }
  try {
    const handle = await workspaceState.rootHandle.getDirectoryHandle(name, { create: true });
    workspaceState.services.push({ name, handle, expanded: true, files: [] });
    workspaceState.services.sort((a, b) => a.name.localeCompare(b.name));
    saveExpandedServices();
    renderWorkspace();
  } catch (err) {
    alert(`Erro ao criar microservico: ${err.message}`);
  }
}

async function createWorkflowInActiveService() {
  const target = currentWorkspaceFile.serviceName || workspaceState.services[0]?.name;
  if (!target) {
    alert("Crie um microservico primeiro.");
    return;
  }
  await createWorkflow(target);
}

async function createWorkflow(serviceName) {
  const service = findService(serviceName);
  if (!service) return;
  const raw = prompt(`Nome do workflow em "${serviceName}":`);
  if (!raw?.trim()) return;
  const fileName = `${normalizeName(raw).replace(/\.(ya?ml)$/i, "")}.yaml`;
  if (service.files.some((file) => file.name === fileName)) {
    alert(`O workflow "${fileName}" ja existe.`);
    return;
  }
  try {
    const content = workflowTemplate(fileName);
    const handle = await WorkspaceFS.createFile(service.handle, fileName, content);
    service.files.push({ name: fileName, handle });
    service.files.sort((a, b) => a.name.localeCompare(b.name));
    service.expanded = true;
    saveExpandedServices();
    renderWorkspace();
    await openWorkflowFile(serviceName, fileName, { skipDirtyCheck: true });
  } catch (err) {
    alert(`Erro ao criar workflow: ${err.message}`);
  }
}

async function renameService(serviceName) {
  const service = findService(serviceName);
  if (!service) return;
  const raw = prompt("Novo nome do microservico:", serviceName);
  if (!raw?.trim()) return;
  const newName = normalizeName(raw);
  if (newName === serviceName) return;
  if (findService(newName)) {
    alert(`O microservico "${newName}" ja existe.`);
    return;
  }
  if (!confirm(`Renomear "${serviceName}" para "${newName}"?`)) return;
  try {
    const newDir = await workspaceState.rootHandle.getDirectoryHandle(newName, { create: true });
    for (const file of service.files) {
      await WorkspaceFS.createFile(newDir, file.name, await WorkspaceFS.readFile(file.handle));
    }
    await workspaceState.rootHandle.removeEntry(serviceName, { recursive: true });
    if (currentWorkspaceFile.serviceName === serviceName) {
      currentWorkspaceFile.serviceName = newName;
      await StudioDB.set(STUDIO_DB.currentFileKey, {
        serviceName: newName,
        fileName: currentWorkspaceFile.fileName,
      });
    }
    await refreshWorkspace();
  } catch (err) {
    alert(`Erro ao renomear microservico: ${err.message}`);
  }
}

async function renameWorkflow(serviceName, fileName) {
  const service = findService(serviceName);
  const file = service?.files.find((item) => item.name === fileName);
  if (!service || !file) return;
  const raw = prompt("Novo nome do workflow:", fileName);
  if (!raw?.trim()) return;
  const newName = `${normalizeName(raw).replace(/\.(ya?ml)$/i, "")}.yaml`;
  if (newName === fileName) return;
  if (service.files.some((item) => item.name === newName)) {
    alert(`O workflow "${newName}" ja existe.`);
    return;
  }
  try {
    const content = currentWorkspaceFile.serviceName === serviceName && currentWorkspaceFile.fileName === fileName
      ? els.workflow.value
      : await WorkspaceFS.readFile(file.handle);
    const newHandle = await WorkspaceFS.createFile(service.handle, newName, content);
    await service.handle.removeEntry(fileName);
    file.name = newName;
    file.handle = newHandle;
    if (currentWorkspaceFile.serviceName === serviceName && currentWorkspaceFile.fileName === fileName) {
      currentWorkspaceFile = { handle: newHandle, serviceName, fileName: newName };
      workflowDirty = false;
      await StudioDB.set(STUDIO_DB.currentFileKey, { serviceName, fileName: newName });
    }
    service.files.sort((a, b) => a.name.localeCompare(b.name));
    renderWorkspace();
  } catch (err) {
    alert(`Erro ao renomear workflow: ${err.message}`);
  }
}

async function deleteService(serviceName) {
  if (!workspaceState.rootHandle) return;
  if (!confirm(`Excluir a pasta "${serviceName}" e todos os workflows dentro dela?`)) return;
  try {
    await workspaceState.rootHandle.removeEntry(serviceName, { recursive: true });
    if (currentWorkspaceFile.serviceName === serviceName) {
      currentWorkspaceFile = { handle: null, serviceName: "", fileName: "" };
      workflowDirty = false;
      await StudioDB.set(STUDIO_DB.currentFileKey, null);
    }
    await refreshWorkspace();
  } catch (err) {
    alert(`Erro ao excluir microservico: ${err.message}`);
  }
}

async function deleteWorkflow(serviceName, fileName) {
  const service = findService(serviceName);
  if (!service) return;
  if (!confirm(`Excluir o workflow "${fileName}"?`)) return;
  try {
    await service.handle.removeEntry(fileName);
    service.files = service.files.filter((file) => file.name !== fileName);
    if (currentWorkspaceFile.serviceName === serviceName && currentWorkspaceFile.fileName === fileName) {
      currentWorkspaceFile = { handle: null, serviceName: "", fileName: "" };
      workflowDirty = false;
      await StudioDB.set(STUDIO_DB.currentFileKey, null);
    }
    renderWorkspace();
  } catch (err) {
    alert(`Erro ao excluir workflow: ${err.message}`);
  }
}

function markWorkflowDirty() {
  invalidateExecutionSnapshot();
  if (!currentWorkspaceFile.handle) return;
  workflowDirty = true;
  els.saveWorkflowFile.disabled = false;
  renderWorkspace();
}

function invalidateExecutionSnapshot() {
  lastExecutionSnapshot = null;
  if (els.reprocess) els.reprocess.disabled = true;
}

function findService(serviceName) {
  return workspaceState.services.find((service) => service.name === serviceName);
}

function normalizeName(value) {
  return String(value)
    .trim()
    .toLowerCase()
    .replace(/\.(ya?ml)$/i, "")
    .replace(/[^a-z0-9._-]+/g, "-")
    .replace(/^-+|-+$/g, "") || "workflow";
}

function workflowTemplate(fileName) {
  const name = fileName.replace(/\.(ya?ml)$/i, "");
  return `name: ${name}
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
      event: ${name}.completed
      fields:
        - id
        - correlation_id
`;
}

function saveExpandedServices() {
  const expanded = workspaceState.services.filter((service) => service.expanded).map((service) => service.name);
  localStorage.setItem("routing-slip-studio:workspace-expanded", JSON.stringify(expanded));
}

function loadExpandedServices() {
  try {
    const expanded = JSON.parse(localStorage.getItem("routing-slip-studio:workspace-expanded") || "[]");
    return Array.isArray(expanded) ? expanded : [];
  } catch {
    return [];
  }
}

function showContextMenu(x, y, items) {
  dismissContextMenu();
  const menu = document.createElement("div");
  menu.className = "context-menu";
  items.forEach((item) => {
    if (item.separator) {
      menu.appendChild(Object.assign(document.createElement("div"), { className: "context-sep" }));
      return;
    }
    const button = document.createElement("button");
    button.type = "button";
    button.textContent = item.label;
    if (item.danger) button.classList.add("danger");
    button.addEventListener("click", () => {
      dismissContextMenu();
      item.action();
    });
    menu.appendChild(button);
  });
  document.body.appendChild(menu);
  const rect = menu.getBoundingClientRect();
  menu.style.left = `${Math.min(x, window.innerWidth - rect.width - 8)}px`;
  menu.style.top = `${Math.min(y, window.innerHeight - rect.height - 8)}px`;
}

function dismissContextMenu() {
  document.querySelector(".context-menu")?.remove();
}

async function expandWorkflowRefsForStudio(workflow, seen = new Set(), source = currentWorkspaceFile) {
  if (!workflow || !Array.isArray(workflow.steps)) return workflow;
  const expanded = { ...workflow, steps: [] };
  for (const step of workflow.steps) {
    if (step.enabled === false) continue;
    if (step.name !== "workflow_ref") {
      expanded.steps.push(step);
      continue;
    }
    const ref = workflowRefFile(step);
    const resolved = resolveWorkspaceWorkflow(ref, source);
    const key = `${resolved.serviceName}/${resolved.fileName}`;
    if (seen.has(key)) throw new Error(`workflow_ref ciclo detectado em ${key}`);
    seen.add(key);
    const text = await WorkspaceFS.readFile(resolved.handle);
    const child = window.jsyaml.load(text);
    if (!child || !Array.isArray(child.steps)) {
      throw new Error(`workflow_ref ${key} nao possui steps validos.`);
    }
    const childSource = { handle: resolved.handle, serviceName: resolved.serviceName, fileName: resolved.fileName };
    const childExpanded = await expandWorkflowRefsForStudio(child, seen, childSource);
    const prefix = cleanWorkflowRefPrefix(step.params?.prefix || step.id || child.name || resolved.fileName);
    const childIDs = new Set(childExpanded.steps.map((childStep) => childStep.id).filter(Boolean));
    childExpanded.steps.forEach((childStep, index) => {
      const cloned = structuredClone(childStep);
      cloned.id = prefixedWorkflowStepID(prefix, cloned.id, cloned.name, index);
      cloned.params = rewriteWorkflowRefTargetsForStudio(cloned.params || {}, prefix, childIDs);
      cloned.__sourceWorkflow = key;
      expanded.steps.push(cloned);
    });
    seen.delete(key);
  }
  return expanded;
}

function workflowRefFile(step) {
  const params = step.params || {};
  const ref = params.file || params.path || params.workflow;
  if (!ref || typeof ref !== "string") throw new Error("workflow_ref precisa de params.file, params.path ou params.workflow.");
  return ref.trim();
}

function resolveWorkspaceWorkflow(ref, source) {
  if (!workspaceState.rootHandle) throw new Error("Abra um workspace para resolver workflow_ref.");
  const parts = ref.replace(/^\/+/, "").split("/").filter(Boolean);
  if (!parts.length) throw new Error("workflow_ref vazio.");
  let serviceName = source?.serviceName || currentWorkspaceFile.serviceName;
  let fileName = "";
  if (ref.startsWith("/") || parts.length > 1) {
    const rootRelative = ref.startsWith("/") || (parts[0] !== "." && parts[0] !== "..");
    const stack = rootRelative ? [] : [serviceName];
    parts.forEach((part) => {
      if (part === ".") return;
      if (part === "..") stack.pop();
      else stack.push(part);
    });
    fileName = stack.pop();
    serviceName = stack.pop();
    if (stack.length) throw new Error(`workflow_ref ${ref} deve apontar para um arquivo YAML dentro de um microservico.`);
  } else {
    fileName = parts[0];
  }
  if (!/\.(ya?ml)$/i.test(fileName)) fileName = `${fileName}.yaml`;
  const service = findService(serviceName);
  const file = service?.files.find((item) => item.name === fileName);
  if (!file) throw new Error(`workflow_ref nao encontrado: ${serviceName}/${fileName}`);
  return { serviceName, fileName, handle: file.handle };
}

function cleanWorkflowRefPrefix(value) {
  return String(value || "workflow")
    .trim()
    .toLowerCase()
    .replace(/\.(ya?ml)$/i, "")
    .replace(/[^a-z0-9._-]+/g, "-")
    .replace(/^-+|-+$/g, "") || "workflow";
}

function prefixedWorkflowStepID(prefix, id, name, index) {
  const cleaned = cleanWorkflowRefPrefix(id || "");
  if (cleaned) return `${prefix}.${cleaned}`;
  return `${prefix}.${String(index + 1).padStart(3, "0")}.${cleanWorkflowRefPrefix(name || "step")}`;
}

function rewriteWorkflowRefTargetsForStudio(params, prefix, childIDs) {
  if (!params || typeof params !== "object") return params;
  Object.entries(params).forEach(([key, value]) => {
    if (key === "to" && typeof value === "string" && childIDs.has(value)) {
      params[key] = `${prefix}.${value}`;
      return;
    }
    if (Array.isArray(value)) {
      value.forEach((item) => rewriteWorkflowRefTargetsForStudio(item, prefix, childIDs));
      return;
    }
    if (value && typeof value === "object") {
      rewriteWorkflowRefTargetsForStudio(value, prefix, childIDs);
    }
  });
  return params;
}
