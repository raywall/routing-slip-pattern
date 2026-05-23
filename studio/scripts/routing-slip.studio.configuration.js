function loadExample(key, options = {}) {
  if (workflowDirty && !confirm("Descartar alteracoes nao salvas no workflow atual?")) return;
  invalidateExecutionSnapshot();
  const example = examples[key] || examples.payment;
  els.example.value = key;
  els.workflow.value = example.workflow;
  els.payload.value = JSON.stringify(example.payload, null, 2);
  currentWorkspaceFile = { handle: null, serviceName: "", fileName: "" };
  workflowDirty = false;
  renderWorkspace();
  clearLogs();
  lintWorkflow();
  if (options.persist !== false) scheduleStudioSave();
}


function currentStudioState() {
  return {
    workflow: els.workflow.value,
    payload: els.payload.value,
    example: els.example.value,
    graphqlEndpoint: els.graphqlEndpoint.value,
    workflowEndpoint: els.workflowEndpoint.value,
    externalApiUrl: els.externalApiUrl.value,
    executeIntegrations: els.integrations.checked,
    updatedAt: new Date().toISOString(),
  };
}

async function restoreStudioState() {
  const state = await StudioDB.get(STUDIO_DB.key);
  if (!state) return false;
  els.workflow.value = typeof state.workflow === "string" ? state.workflow : "";
  els.payload.value = typeof state.payload === "string" ? state.payload : "{}";
  if (state.example && examples[state.example]) els.example.value = state.example;
  if (state.graphqlEndpoint) els.graphqlEndpoint.value = state.graphqlEndpoint;
  if (state.workflowEndpoint) els.workflowEndpoint.value = state.workflowEndpoint;
  if (state.externalApiUrl) els.externalApiUrl.value = state.externalApiUrl;
  els.integrations.checked = Boolean(state.executeIntegrations);
  return els.workflow.value.trim() !== "";
}

function scheduleStudioSave() {
  clearTimeout(saveTimer);
  saveTimer = setTimeout(() => {
    StudioDB.set(STUDIO_DB.key, currentStudioState());
  }, 250);
}

const StudioDB = {
  open() {
    return new Promise((resolve, reject) => {
      if (!window.indexedDB) {
        reject(new Error("IndexedDB indisponivel"));
        return;
      }
      const request = indexedDB.open(STUDIO_DB.name, 1);
      request.onupgradeneeded = () => {
        if (!request.result.objectStoreNames.contains(STUDIO_DB.store)) {
          request.result.createObjectStore(STUDIO_DB.store);
        }
      };
      request.onsuccess = () => resolve(request.result);
      request.onerror = () => reject(request.error);
    });
  },

  async get(key) {
    try {
      const db = await this.open();
      const value = await new Promise((resolve, reject) => {
        const tx = db.transaction(STUDIO_DB.store, "readonly");
        const request = tx.objectStore(STUDIO_DB.store).get(key);
        request.onsuccess = () => resolve(request.result || null);
        request.onerror = () => reject(request.error);
      });
      db.close();
      return value;
    } catch (_) {
      try {
        return JSON.parse(localStorage.getItem(`routing-slip-studio:${key}`) || "null");
      } catch {
        return null;
      }
    }
  },

  async set(key, value) {
    try {
      const db = await this.open();
      await new Promise((resolve, reject) => {
        const tx = db.transaction(STUDIO_DB.store, "readwrite");
        tx.objectStore(STUDIO_DB.store).put(value, key);
        tx.oncomplete = resolve;
        tx.onerror = () => reject(tx.error);
      });
      db.close();
    } catch (_) {
      localStorage.setItem(`routing-slip-studio:${key}`, JSON.stringify(value));
    }
  },
};
