function validatePayload() {
  try {
    JSON.parse(els.payload.value);
    return true;
  } catch (err) {
    return false;
  }
}

function handlePayloadKeydown(event) {
  if (event.key !== "Tab") return;
  event.preventDefault();
  insertTabIndent(event.currentTarget, { outdent: event.shiftKey });
  invalidateExecutionSnapshot();
  validatePayload();
  scheduleStudioSave();
}
