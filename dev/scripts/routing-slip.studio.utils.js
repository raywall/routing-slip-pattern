function getPath(obj, path) {
  if (!path) return undefined;
  return String(path).split(".").reduce((acc, key) => {
    if (Array.isArray(acc) && /^\d+$/.test(key)) return acc[Number(key)];
    if (acc && typeof acc === "object" && key in acc) return acc[key];
    return undefined;
  }, obj);
}

function findWorkflowStepIndex(ref) {
  const steps = (activeRuntimeWorkflow || lastWorkflow)?.steps || [];
  let index = steps.findIndex((step) => step.id === ref);
  if (index >= 0) return index;
  return steps.findIndex((step) => step.name === ref);
}

function evaluateValueConfig(config, state) {
  if ("literal" in config) return config.literal;
  if (typeof config.count === "string") {
    const value = getPath(state.payload, config.count);
    if (!isCountable(value)) throw new Error(`compute: field "${config.count}" is not countable`);
    return value.length ?? Object.keys(value).length;
  }
  if ("exists" in config) return getPath(state.payload, config.exists) !== undefined;
  if (config.field && Object.keys(config).length === 1) {
    const value = getPath(state.payload, config.field);
    if (value === undefined) throw new Error(`compute: field "${config.field}" not found`);
    return value;
  }
  return evaluateConditionConfig(config, state);
}

function evaluateAssertConfig(config, state) {
  if (Array.isArray(config.all)) {
    const failures = [];
    config.all.forEach((condition, index) => {
      try {
        if (!evaluateConditionConfig(condition, state)) failures.push(`all[${index}]: condition not satisfied`);
      } catch (err) {
        failures.push(`all[${index}]: ${err.message}`);
      }
    });
    return { matched: failures.length === 0, failures };
  }
  if (Array.isArray(config.any)) {
    const failures = [];
    for (let index = 0; index < config.any.length; index += 1) {
      try {
        if (evaluateConditionConfig(config.any[index], state)) return { matched: true, failures: [] };
        failures.push(`any[${index}]: condition not satisfied`);
      } catch (err) {
        failures.push(`any[${index}]: ${err.message}`);
      }
    }
    return { matched: false, failures };
  }
  return { matched: evaluateConditionConfig(config, state), failures: ["condition not satisfied"] };
}

function evaluateConditionConfig(config, state) {
  if (typeof config.exists === "string") return getPath(state.payload, config.exists) !== undefined;
  const value = getPath(state.payload, config.field);
  if (value === undefined) throw new Error(`field "${config.field}" not found`);
  if ("equals" in config) return String(value) === String(config.equals);
  if ("not_equals" in config) return String(value) !== String(config.not_equals);
  if ("less_than" in config) return Number(value) < Number(config.less_than);
  if ("less_than_or_equal" in config) return Number(value) <= Number(config.less_than_or_equal);
  if ("greater_than" in config) return Number(value) > Number(config.greater_than);
  if ("greater_than_or_equal" in config) return Number(value) >= Number(config.greater_than_or_equal);
  if ("min_items" in config) {
    if (!isCountable(value)) throw new Error(`field "${config.field}" is not countable`);
    return value.length >= Number(config.min_items);
  }
  if ("max_items" in config) {
    if (!isCountable(value)) throw new Error(`field "${config.field}" is not countable`);
    return value.length <= Number(config.max_items);
  }
  throw new Error("no supported comparison configured");
}

function isCountable(value) {
  return Array.isArray(value) || typeof value === "string" || Boolean(value && typeof value === "object");
}

function setPath(obj, path, value) {
  const parts = String(path).split(".");
  let current = obj;
  parts.slice(0, -1).forEach((part) => {
    if (!current[part] || typeof current[part] !== "object") current[part] = {};
    current = current[part];
  });
  current[parts[parts.length - 1]] = value;
}

function interpolateAny(value, payload) {
  if (typeof value === "string") return interpolateString(value, payload);
  if (Array.isArray(value)) return value.map((item) => interpolateAny(item, payload));
  if (value && typeof value === "object") {
    return Object.fromEntries(Object.entries(value).map(([key, item]) => [key, interpolateAny(item, payload)]));
  }
  return value;
}

function interpolateString(text, payload) {
  const exact = text.match(/^\{([^{}]+)\}$/);
  if (exact) {
    const value = getPath(payload, exact[1]);
    return value === undefined ? text : value;
  }
  return text.replace(/\{([^{}]+)\}/g, (_, path) => {
    const value = getPath(payload, path);
    return value === undefined ? "" : String(value);
  });
}

function expandEnv(value) {
  return String(value).replace(/\$\{([^}:]+):-([^}]+)\}/g, (_, key, fallback) => {
    return localStorage.getItem(key) || fallback;
  });
}

function error(message) {
  return { level: "error", message };
}

function warn(message) {
  return { level: "warn", message };
}

function stringValue(value) {
  return typeof value === "string" && value.trim() !== "";
}

function escapeHtml(value) {
  return String(value)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}
