function validatePayload() {
  try {
    JSON.parse(els.payload.value);
    return true;
  } catch (err) {
    return false;
  }
}
