var __defProp = Object.defineProperty;
var __defNormalProp = (obj, key, value) => key in obj ? __defProp(obj, key, { enumerable: true, configurable: true, writable: true, value }) : obj[key] = value;
var __publicField = (obj, key, value) => __defNormalProp(obj, typeof key !== "symbol" ? key + "" : key, value);

// web/src/api.ts
var ConfigGenerationError = class extends Error {
  constructor(message, rawResponse, issues = []) {
    super(message);
    __publicField(this, "rawResponse");
    __publicField(this, "issues");
    this.name = "ConfigGenerationError";
    this.rawResponse = rawResponse;
    this.issues = issues;
  }
};
async function fetchRenderedDraftCard() {
  const response = await fetch("/api/draft-card/rendered", { cache: "no-store" });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to load draft card."));
  }
  return await response.json();
}
async function resetDraftCard() {
  const response = await fetch("/api/draft-card/reset", { method: "POST" });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to reset draft card."));
  }
  return await response.json();
}
async function generateConfig(componentKind, instruction, update = false) {
  const response = await fetch("/api/draft-card/configs/" + encodeURIComponent(componentKind), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ instruction, update })
  });
  if (!response.ok) {
    throw await readConfigError(response, "Failed to generate design.");
  }
  return await response.json();
}
async function applyDraftConfig(generatedConfig) {
  const response = await fetch("/api/draft-card/apply-config", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ generated_config: generatedConfig })
  });
  if (!response.ok) {
    throw await readConfigError(response, "Failed to apply design.");
  }
  return await response.json();
}
async function fetchDesignLibrary(componentKind = "") {
  const suffix = componentKind ? "?componentKind=" + encodeURIComponent(componentKind) : "";
  const response = await fetch("/api/draft-card/library" + suffix, { cache: "no-store" });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to load design library."));
  }
  const payload = await response.json();
  return payload.library || [];
}
async function saveAppliedDesign() {
  const response = await fetch("/api/draft-card/library/save-applied", { method: "POST" });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to save applied design."));
  }
  return await response.json();
}
async function applyLibraryDesign(itemID) {
  const response = await fetch("/api/draft-card/library/apply", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ item_id: itemID })
  });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to apply library design."));
  }
  return await response.json();
}
async function fetchGameSession() {
  const response = await fetch("/api/game/session", { cache: "no-store" });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to load game session."));
  }
  return await response.json();
}
async function resetGameSession() {
  const response = await fetch("/api/game/reset", { method: "POST" });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to reset game session."));
  }
  return await response.json();
}
async function cycleGameCard(direction) {
  const response = await fetch("/api/game/cycle", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ direction })
  });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to cycle card."));
  }
  return await response.json();
}
async function collectGameCard(cardId) {
  const response = await fetch("/api/game/collect", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ cardId })
  });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to collect card."));
  }
  return await response.json();
}
async function playGameCard(sourceCardId, targetCardId) {
  const response = await fetch("/api/game/play-card", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ sourceCardId, targetCardId })
  });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to play card."));
  }
  return await response.json();
}
async function selectGameCardComponent(cardId, componentId, componentKind) {
  const response = await fetch("/api/game/component/select", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ cardId, componentId, componentKind })
  });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to select component."));
  }
  return await response.json();
}
async function applyGameCardComponentControl(cardId, componentId, componentKind, control, value) {
  const response = await fetch("/api/game/component/control-change", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ cardId, componentId, componentKind, control, value })
  });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to update active card."));
  }
  return await response.json();
}
async function startGameEdit(cardId) {
  const response = await fetch("/api/game/edit/start", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ cardId })
  });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to start editing card."));
  }
  return await response.json();
}
async function installGameEditComponent(componentCardId) {
  const response = await fetch("/api/game/edit/install-component", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ componentCardId })
  });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to install component."));
  }
  return await response.json();
}
async function selectGameEditComponent(componentId, componentKind) {
  const response = await fetch("/api/game/edit/component/select", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ componentId, componentKind })
  });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to select draft component."));
  }
  return await response.json();
}
async function applyGameEditControl(componentId, control, value) {
  const response = await fetch("/api/game/edit/control-change", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ componentId, control, value })
  });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to update draft card."));
  }
  return await response.json();
}
async function applyGameLibraryComponentControl(cardId, componentId, componentKind, control, value) {
  const response = await fetch("/api/game/library/component/control-change", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ cardId, componentId, componentKind, control, value })
  });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to update library card."));
  }
  return await response.json();
}
async function saveGameEdit() {
  const response = await fetch("/api/game/edit/save", { method: "POST" });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to save edited card."));
  }
  return await response.json();
}
async function cancelGameEdit() {
  const response = await fetch("/api/game/edit/cancel", { method: "POST" });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to cancel editing."));
  }
  return await response.json();
}
async function addDraftComponent(componentKind, config) {
  const response = await fetch("/api/draft-card/components", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ componentKind, config })
  });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to add component."));
  }
  return await response.json();
}
async function readError(response, fallback) {
  const text = String(await response.text() || "").trim();
  return text || fallback;
}
async function readConfigError(response, fallback) {
  const contentType = response.headers.get("Content-Type") || "";
  if (contentType.includes("application/json")) {
    try {
      const payload = await response.json();
      const message = String(payload.message || fallback).trim();
      const rawResponse = String(payload.raw_response || "").trim();
      const issues = Array.isArray(payload.issues) ? payload.issues : [];
      if (rawResponse || issues.length) {
        return new ConfigGenerationError(message, rawResponse, issues);
      }
      return new Error(message);
    } catch {
      return new Error(fallback);
    }
  }
  return new Error(await readError(response, fallback));
}

// web/src/dom.ts
function byID(id) {
  return document.getElementById(id);
}

// web/src/designer/document.ts
function replacePreviewHTML(previewHTML) {
  const current = document.getElementById("draft-card-preview");
  if (!current) return;
  const template = document.createElement("template");
  template.innerHTML = String(previewHTML || "").trim();
  const next = template.content.firstElementChild;
  if (!(next instanceof HTMLElement) || next.id !== "draft-card-preview") {
    throw new Error("Server returned invalid preview HTML.");
  }
  current.replaceWith(next);
}

// web/src/designer/configs.ts
async function generateComponentConfig(componentKind, instruction, update = false) {
  return await generateConfig(componentKind, instruction, update);
}
function parseGeneratedConfigEnvelope(raw) {
  const trimmed = String(raw || "").trim();
  if (!trimmed || trimmed === "{}") {
    throw new Error("Generate or paste a config before applying.");
  }
  let parsed;
  try {
    parsed = JSON.parse(trimmed);
  } catch {
    throw new Error("Generated config is not valid JSON.");
  }
  return normalizeGeneratedConfigEnvelope(parsed);
}
function normalizeGeneratedConfigEnvelope(value) {
  if (!value || typeof value !== "object") {
    throw new Error("Generated config must be a JSON object.");
  }
  const record = value;
  const componentKind = String(record.componentKind || "").trim();
  const config = record.config;
  if (!config || typeof config !== "object") {
    throw new Error("Generated config must include a config object.");
  }
  return {
    componentKind,
    description: String(record.description || savedDesignFallbackName(componentKind)).trim(),
    config: cloneJSON(config)
  };
}
function isConfigGenerationError(error) {
  return error instanceof ConfigGenerationError;
}
function cloneJSON(value) {
  return JSON.parse(JSON.stringify(value));
}
function formatIssues(issues) {
  return issues.slice(0, 3).map((issue) => {
    const path = String(issue.path || "$");
    const message = String(issue.message || issue.code || "invalid value");
    return path + " " + message;
  }).join("; ");
}
function savedDesignFallbackName(componentKind) {
  switch (componentKind) {
    case "background":
      return "Saved Background";
    case "border":
      return "Saved Border";
    case "textarea":
      return "Saved Text Area";
    default:
      return "Saved Design";
  }
}

// web/src/designer/controller.ts
function initDesigner() {
  const form = byID("card-designer-form");
  if (form) {
    form.addEventListener("submit", (event) => {
      void generateConfig2(event);
    });
  }
  const apply = byID("apply-config-btn");
  if (apply) {
    apply.addEventListener("click", () => {
      void applyConfig();
    });
  }
  const editor = byID("config-preview");
  if (editor) {
    editor.addEventListener("input", () => {
      const hasCandidate = hasEditableConfigCandidate();
      const description = byID("config-description");
      if (description && hasCandidate) {
        description.textContent = "Edited design. Apply to validate it against the selected config kind.";
      }
      if (apply) {
        apply.disabled = !hasCandidate;
      }
      setSaveEnabled(false);
    });
  }
  const save = byID("save-design-btn");
  if (save) {
    save.addEventListener("click", () => {
      void saveAppliedDesignToLibrary();
    });
  }
  const configKind = byID("config-target");
  if (configKind) {
    configKind.addEventListener("change", () => {
      void loadDesignLibrary();
    });
  }
  const reset2 = byID("designer-reset-btn");
  if (reset2) {
    reset2.addEventListener("click", () => {
      void resetDraft();
    });
  }
  bindAddComponentControls();
  renderProposedConfig(null);
  setSaveEnabled(false);
  void loadDesigner();
}
async function resetDraft() {
  setDesignerStatus("Resetting draft...", false);
  try {
    const response = await resetDraftCard();
    replacePreviewHTML(response.preview_html);
    renderProposedConfig(null);
    renderDesignLibraryItems(response.library);
    setSaveEnabled(false);
    setDesignerStatus("Ready.", false);
    document.dispatchEvent(new CustomEvent("living-card:interactive-refresh"));
  } catch (error) {
    setDesignerStatus(error instanceof Error ? error.message : "Failed to reset draft card.", true);
  }
}
async function loadDesigner() {
  setDesignerStatus("Loading draft card...", false);
  try {
    const response = await fetchRenderedDraftCard();
    replacePreviewHTML(response.preview_html);
    renderProposedConfig(null);
    renderDesignLibraryItems(response.library);
    setSaveEnabled(false);
    setDesignerStatus("Ready.", false);
  } catch (error) {
    setDesignerStatus(error instanceof Error ? error.message : "Failed to load draft card.", true);
  }
}
function renderProposedConfig(config) {
  const preview = byID("config-preview");
  const description = byID("config-description");
  const apply = byID("apply-config-btn");
  if (preview) {
    preview.value = config ? JSON.stringify(config, null, 2) : "{}";
  }
  if (description) {
    description.textContent = config ? config.description : "No generated config yet.";
  }
  if (apply) {
    apply.disabled = !config;
  }
}
function renderFailedConfig(rawResponse, message, issues = []) {
  const preview = byID("config-preview");
  const description = byID("config-description");
  const apply = byID("apply-config-btn");
  if (preview) {
    preview.value = rawResponse || "{}";
  }
  if (description) {
    const issueSummary = formatIssues(issues);
    description.textContent = issueSummary ? "Generation failed: " + issueSummary + " Edit the response below, then apply it to validate." : "Generation failed. Edit the response below, then apply it to validate.";
  }
  if (apply) {
    apply.disabled = !hasEditableConfigCandidate();
  }
  setSaveEnabled(false);
  setDesignerStatus(message, true);
}
async function loadDesignLibrary() {
  try {
    renderDesignLibraryItems(await fetchDesignLibrary(readConfigKind()));
  } catch (error) {
    setDesignerStatus(error instanceof Error ? error.message : "Failed to load design library.", true);
  }
}
function renderDesignLibraryItems(items) {
  const list = byID("design-library-list");
  if (!list) return;
  const configKind = readConfigKind();
  const visibleItems = items.filter((item) => item.componentKind === configKind);
  if (!visibleItems.length) {
    list.innerHTML = '<div class="rounded-md border border-dashed border-[var(--app-border)] px-3 py-4 text-center text-sm text-[var(--app-fg-soft)]">No saved designs.</div>';
    return;
  }
  list.innerHTML = "";
  visibleItems.forEach((item) => {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "rounded-md border border-[var(--app-border)] bg-[var(--app-panel)] px-3 py-2 text-left shadow-sm transition hover:border-cyan-400/50 hover:bg-cyan-500/10";
    button.innerHTML = '<div class="flex items-center justify-between gap-2"><span class="text-sm font-semibold text-[var(--app-fg)]">' + escapeHTML(item.name) + '</span><span class="text-[0.68rem] font-semibold uppercase text-[var(--app-fg-soft)]">' + (item.saved ? "Saved" : "Preset") + '</span></div><p class="mt-1 text-xs text-[var(--app-fg-muted)]">' + escapeHTML(item.description) + "</p>";
    button.addEventListener("click", () => {
      void applyLibraryItem(item.id);
    });
    list.appendChild(button);
  });
}
function setDesignerStatus(message, isError) {
  const status = byID("designer-status");
  if (!status) return;
  status.textContent = message;
  status.className = isError ? "mt-4 text-sm text-red-300" : "mt-4 text-sm text-[var(--app-fg-soft)]";
}
async function generateConfig2(event) {
  event.preventDefault();
  const configKind = readConfigKind();
  const isUpdate = event.submitter?.id === "update-config-btn";
  const input = byID("config-instruction");
  const instruction = String(input?.value || "").trim();
  if (!instruction) {
    setDesignerStatus("Instruction cannot be empty.", true);
    return;
  }
  setBusy(true, isUpdate);
  setSaveEnabled(false);
  setDesignerStatus(isUpdate ? "Updating design..." : "Generating design...", false);
  try {
    const config = await generateComponentConfig(configKind, instruction, isUpdate);
    renderProposedConfig(config);
    setDesignerStatus(isUpdate ? "Config updated. Review it before applying." : "Config generated. Review it before applying.", false);
  } catch (error) {
    if (isConfigGenerationError(error)) {
      renderFailedConfig(error.rawResponse, error.message, error.issues);
      return;
    }
    setDesignerStatus(error instanceof Error ? error.message : "Config generation failed.", true);
  } finally {
    setBusy(false, isUpdate);
  }
}
async function applyConfig() {
  try {
    const config = readConfigFromEditor();
    setBusy(true);
    setDesignerStatus("Applying design...", false);
    const response = await applyDraftConfig(config);
    replacePreviewHTML(response.preview_html);
    renderProposedConfig(response.normalized_config);
    renderDesignLibraryItems(response.library);
    setSaveEnabled(true);
    setDesignerStatus("Config applied to the preview.", false);
  } catch (error) {
    if (isConfigGenerationError(error)) {
      const editor = byID("config-preview");
      renderFailedConfig(error.rawResponse || String(editor?.value || ""), error.message, error.issues);
      return;
    }
    setDesignerStatus(error instanceof Error ? error.message : "Config could not be applied.", true);
  } finally {
    setBusy(false);
  }
}
async function applyLibraryItem(itemID) {
  try {
    setBusy(true);
    setDesignerStatus("Applying library design...", false);
    const response = await applyLibraryDesign(itemID);
    replacePreviewHTML(response.preview_html);
    renderProposedConfig(response.normalized_config);
    renderDesignLibraryItems(response.library);
    setSaveEnabled(true);
    setDesignerStatus("Library design applied to the preview.", false);
  } catch (error) {
    setDesignerStatus(error instanceof Error ? error.message : "Library design could not be applied.", true);
  } finally {
    setBusy(false);
  }
}
async function saveAppliedDesignToLibrary() {
  try {
    const response = await saveAppliedDesign();
    renderDesignLibraryItems(response.library);
    setSaveEnabled(false);
    setDesignerStatus(response.item ? "Design saved to the library." : "Design is already in the library.", false);
  } catch (error) {
    setDesignerStatus(error instanceof Error ? error.message : "Apply a generated design before saving.", true);
  }
}
function setSaveEnabled(enabled) {
  const save = byID("save-design-btn");
  if (!save) return;
  save.disabled = !enabled;
}
function readConfigKind() {
  const select = byID("config-target");
  switch (select?.value) {
    case "border":
    case "textarea":
    case "image":
      return select.value;
    default:
      return "background";
  }
}
function bindAddComponentControls() {
  byID("add-textarea-component-btn")?.addEventListener("click", () => {
    void addComponent("textarea");
  });
  byID("add-shape-component-btn")?.addEventListener("click", () => {
    void addComponent("shape");
  });
  byID("add-image-component-input")?.addEventListener("change", (event) => {
    const input = event.currentTarget;
    const file = input.files?.[0];
    input.value = "";
    if (!file) return;
    void addImageComponent(file);
  });
}
async function addComponent(componentKind) {
  try {
    setDesignerStatus("Adding component...", false);
    const response = await addDraftComponent(componentKind);
    renderDesignLibraryItems(response.library);
    setDesignerStatus("Component added to the draft card.", false);
    document.dispatchEvent(new CustomEvent("living-card:interactive-refresh"));
  } catch (error) {
    setDesignerStatus(error instanceof Error ? error.message : "Component could not be added.", true);
  }
}
async function addImageComponent(file) {
  try {
    if (!["image/png", "image/jpeg", "image/webp", "image/gif"].includes(file.type)) {
      throw new Error("Image must be PNG, JPEG, WebP, or GIF.");
    }
    const src = await fileToDataURL(file);
    setDesignerStatus("Adding image component...", false);
    const response = await addDraftComponent("image", {
      src,
      alt: file.name || "Uploaded image",
      x: 50,
      y: 48,
      width: 46,
      height: 32,
      rotation: 0,
      border_color: "rgba(255,255,255,0.2)",
      border_width_px: 1,
      border_radius_px: 14
    });
    renderDesignLibraryItems(response.library);
    setDesignerStatus("Image component added to the draft card.", false);
    document.dispatchEvent(new CustomEvent("living-card:interactive-refresh"));
  } catch (error) {
    setDesignerStatus(error instanceof Error ? error.message : "Image component could not be added.", true);
  }
}
function fileToDataURL(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.addEventListener("load", () => resolve(String(reader.result || "")));
    reader.addEventListener("error", () => reject(new Error("Could not read image file.")));
    reader.readAsDataURL(file);
  });
}
function setBusy(isBusy, isUpdate = false) {
  const generate = byID("generate-config-btn");
  const update = byID("update-config-btn");
  const apply = byID("apply-config-btn");
  if (generate) {
    generate.disabled = isBusy;
    generate.textContent = isBusy && !isUpdate ? "Generating..." : "Generate";
  }
  if (update) {
    update.disabled = isBusy;
    update.textContent = isBusy && isUpdate ? "Updating..." : "Update";
  }
  if (apply) {
    apply.disabled = isBusy || !hasEditableConfigCandidate();
  }
}
function readConfigFromEditor() {
  const editor = byID("config-preview");
  return parseGeneratedConfigEnvelope(String(editor?.value || ""));
}
function hasEditableConfigCandidate() {
  const editor = byID("config-preview");
  const raw = String(editor?.value || "").trim();
  return Boolean(raw && raw !== "{}");
}
function escapeHTML(value) {
  return String(value || "").replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;").replace(/'/g, "&#39;");
}

// web/src/stage/componentControls.ts
var swatches = [
  "#22c55e",
  "#38bdf8",
  "#f59e0b",
  "#f43f5e",
  "#a78bfa",
  "#f8fafc",
  "#111827",
  "#f5e6c8"
];
function openComponentOverlay(options) {
  const root = options.root;
  if (!root) return;
  closeComponentOverlay(root);
  const slots = edgeControlSlots(root);
  if (!slots) return;
  document.body.classList.add("stage-controls-open");
  renderHeader(slots.top, options.overlay, () => {
    closeComponentOverlay(root);
    options.onClose();
  });
  const leftControls = [];
  const rightControls = [];
  options.overlay.controls.forEach((control) => {
    if (control.kind === "range" || control.kind === "color") {
      rightControls.push(control);
      return;
    }
    leftControls.push(control);
  });
  renderControlRail(slots.left, leftControls, options.onControl);
  renderControlRail(slots.right, rightControls, options.onControl);
  renderStatus(slots.bottom, "Controls stay fixed while you edit " + options.overlay.title.toLowerCase() + ".");
}
function closeComponentOverlay(root) {
  document.body.classList.remove("stage-controls-open");
  const slots = root ? edgeControlSlots(root) : null;
  slots?.all.forEach((slot) => {
    slot.innerHTML = "";
  });
  if (slots?.bottom) {
    renderStatus(slots.bottom, "");
  }
}
function edgeControlSlots(root) {
  const top = root.querySelector("#stage-edge-controls-top");
  const left = root.querySelector("#stage-edge-controls-left");
  const right = root.querySelector("#stage-edge-controls-right");
  const bottom = root.querySelector("#stage-edge-controls-bottom");
  if (!top || !left || !right || !bottom) return null;
  return { top, left, right, bottom, all: [top, left, right, bottom] };
}
function renderHeader(root, overlay, onClose) {
  root.innerHTML = "";
  stopCardTapEvents(root);
  const panel = document.createElement("div");
  panel.className = "stage-edge-panel stage-edge-header";
  const text = document.createElement("div");
  text.className = "min-w-0";
  const title = document.createElement("div");
  title.className = "stage-edge-title";
  title.textContent = overlay.title;
  const subtitle = document.createElement("div");
  subtitle.className = "stage-edge-subtitle";
  subtitle.textContent = overlay.componentKind + " controls";
  text.append(title, subtitle);
  const close = document.createElement("button");
  close.type = "button";
  close.className = "h-8 rounded-md border border-[var(--app-border)] bg-[var(--app-panel)] px-2 text-xs font-semibold text-[var(--app-fg-soft)]";
  close.textContent = "Close";
  close.addEventListener("click", onClose);
  panel.append(text, close);
  root.appendChild(panel);
}
function renderControlRail(root, controls, onControl) {
  root.innerHTML = "";
  stopCardTapEvents(root);
  const panel = document.createElement("div");
  panel.className = "stage-edge-panel stage-edge-controls-group";
  if (!controls.length) {
    const empty = document.createElement("div");
    empty.className = "stage-edge-controls-empty";
    empty.textContent = "No controls here.";
    panel.appendChild(empty);
  } else {
    controls.forEach((control) => {
      panel.appendChild(renderControl(control, (value) => onControl(control, value)));
    });
  }
  root.appendChild(panel);
}
function renderStatus(root, message, tone = "info") {
  root.innerHTML = "";
  stopCardTapEvents(root);
  const panel = document.createElement("div");
  panel.className = "stage-edge-panel";
  const status = document.createElement("div");
  status.id = "stage-edge-controls-status";
  status.className = "stage-edge-controls-status";
  status.dataset.tone = tone;
  status.textContent = message;
  panel.appendChild(status);
  root.appendChild(panel);
}
function renderControl(control, onValue) {
  switch (control.kind) {
    case "checkbox":
      return renderCheckboxControl(control, onValue);
    case "color":
      return renderColorControl(control, onValue);
    case "range":
      return renderRangeControl(control, onValue);
    case "select":
      return renderSelectControl(control, onValue);
    case "text":
      return renderTextControl(control, onValue);
    default:
      return document.createElement("div");
  }
}
function renderCheckboxControl(control, onValue) {
  const wrapper = document.createElement("label");
  wrapper.className = "flex items-center justify-between gap-3 rounded-md border border-[var(--app-border)] bg-[var(--app-panel)] px-2 py-2 text-sm font-semibold text-[var(--app-fg)]";
  const text = document.createElement("span");
  text.textContent = control.label;
  const input = document.createElement("input");
  input.type = "checkbox";
  input.className = "h-4 w-4 accent-emerald-300";
  input.checked = Boolean(control.value);
  input.addEventListener("change", () => onValue(input.checked));
  wrapper.append(text, input);
  return wrapper;
}
function renderColorControl(control, onValue) {
  const wrapper = controlWrapper(control);
  const current = hexOrFallback(String(control.value || "#22c55e"));
  const row = document.createElement("div");
  row.className = "grid grid-cols-[1fr_auto] gap-2";
  const input = document.createElement("input");
  input.type = "color";
  input.value = current;
  input.className = "h-9 w-full cursor-pointer rounded-md border border-[var(--app-border)] bg-[var(--app-panel)]";
  input.addEventListener("change", () => onValue(input.value));
  const preview = document.createElement("span");
  preview.className = "block h-9 w-9 rounded-md border border-white/25";
  preview.style.background = current;
  input.addEventListener("input", () => {
    preview.style.background = input.value;
  });
  row.append(input, preview);
  const grid = document.createElement("div");
  grid.className = "grid grid-cols-8 gap-1";
  swatches.forEach((color) => {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "h-6 rounded-sm border border-white/25";
    button.title = color;
    button.style.background = color;
    button.addEventListener("click", () => {
      input.value = color;
      preview.style.background = color;
      onValue(color);
    });
    grid.appendChild(button);
  });
  wrapper.append(row, grid);
  return wrapper;
}
function renderRangeControl(control, onValue) {
  const wrapper = controlWrapper(control);
  const value = document.createElement("span");
  value.className = "text-xs font-semibold text-[var(--app-fg-soft)]";
  value.textContent = String(control.value ?? control.min ?? 0);
  const input = document.createElement("input");
  input.type = "range";
  input.min = String(control.min ?? 0);
  input.max = String(control.max ?? 100);
  input.step = String(control.step ?? 1);
  input.value = String(control.value ?? control.min ?? 0);
  input.className = "w-full accent-emerald-300";
  input.addEventListener("input", () => {
    value.textContent = input.value;
  });
  input.addEventListener("change", () => onValue(Number(input.value)));
  wrapper.firstElementChild?.appendChild(value);
  wrapper.appendChild(input);
  return wrapper;
}
function renderSelectControl(control, onValue) {
  const wrapper = controlWrapper(control);
  const select = document.createElement("select");
  select.className = "h-9 rounded-md border border-[var(--app-border)] bg-[var(--app-panel)] px-2 text-sm text-[var(--app-fg)] outline-none";
  (control.options || []).forEach((option) => {
    const item = document.createElement("option");
    item.value = option.value;
    item.textContent = option.label;
    select.appendChild(item);
  });
  select.value = String(control.value ?? "");
  select.addEventListener("change", () => onValue(select.value));
  wrapper.appendChild(select);
  return wrapper;
}
function renderTextControl(control, onValue) {
  const wrapper = controlWrapper(control);
  const input = document.createElement("input");
  input.type = "text";
  input.value = String(control.value ?? "");
  input.className = "h-9 rounded-md border border-[var(--app-border)] bg-[var(--app-panel)] px-2 text-sm text-[var(--app-fg)] outline-none";
  input.addEventListener("change", () => onValue(input.value));
  input.addEventListener("keydown", (event) => {
    if (event.key === "Enter") {
      event.preventDefault();
      input.blur();
      onValue(input.value);
    }
  });
  wrapper.appendChild(input);
  return wrapper;
}
function controlWrapper(control) {
  const wrapper = document.createElement("label");
  wrapper.className = "grid gap-1 text-xs font-semibold uppercase text-[var(--app-fg-soft)]";
  const header = document.createElement("span");
  header.className = "grid gap-0.5";
  const text = document.createElement("span");
  text.textContent = control.label;
  header.appendChild(text);
  if (control.property) {
    const property = document.createElement("span");
    property.className = "stage-edge-property";
    property.textContent = control.property;
    header.appendChild(property);
  }
  wrapper.appendChild(header);
  return wrapper;
}
function stopCardTapEvents(element) {
  for (const eventName of ["pointerdown", "pointermove", "pointerup", "pointercancel", "contextmenu"]) {
    element.addEventListener(eventName, (event) => {
      event.stopPropagation();
    });
  }
}
function hexOrFallback(value) {
  const trimmed = String(value || "").trim();
  return /^#[0-9a-fA-F]{6}$/.test(trimmed) ? trimmed : "#22c55e";
}

// web/src/game/GameController.ts
var latestSession = null;
var busy = false;
var controlBusy = false;
var activePressState = null;
var editPressState = null;
var longPressDelayMS = 560;
var dragThresholdPX = 6;
function initGameStage() {
  bindControls();
  void loadSession();
}
function bindControls() {
  byID("game-prev-card")?.addEventListener("click", () => {
    void cycle("previous");
  });
  byID("game-next-card")?.addEventListener("click", () => {
    void cycle("next");
  });
  byID("game-collect-card")?.addEventListener("click", () => {
    const active = latestSession?.activeWorldCard;
    if (!active) return;
    void collect(active.id);
  });
  byID("reset-draft-btn")?.addEventListener("click", () => {
    void reset();
  });
  byID("game-edit-save")?.addEventListener("click", () => {
    void saveEdit();
  });
  byID("game-edit-cancel")?.addEventListener("click", () => {
    void cancelEdit();
  });
  const worldTarget = byID("game-world-card");
  worldTarget?.addEventListener("dragover", (event) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = "move";
  });
  worldTarget?.addEventListener("drop", (event) => {
    event.preventDefault();
    const sourceCardId = event.dataTransfer?.getData("text/plain") || "";
    const targetCardId = latestSession?.activeWorldCardId || "";
    if (sourceCardId && targetCardId) {
      closeComponentOverlay(overlayRoot());
      void play(sourceCardId, targetCardId);
    }
  });
  worldTarget?.addEventListener("pointerdown", onActivePointerDown);
  worldTarget?.addEventListener("pointermove", onActivePointerMove);
  worldTarget?.addEventListener("pointerup", onActivePointerUp);
  worldTarget?.addEventListener("pointercancel", onActivePointerCancel);
  worldTarget?.addEventListener("contextmenu", (event) => {
    if (activeComponentHit(event)) {
      event.preventDefault();
    }
  });
  bindSliderInputEvents(worldTarget, "world");
  const editCanvas = byID("game-edit-canvas");
  editCanvas?.addEventListener("dragover", (event) => {
    event.preventDefault();
    editCanvas.dataset.dropActive = "true";
    event.dataTransfer.dropEffect = "move";
  });
  editCanvas?.addEventListener("dragleave", () => {
    delete editCanvas.dataset.dropActive;
  });
  editCanvas?.addEventListener("drop", (event) => {
    event.preventDefault();
    delete editCanvas.dataset.dropActive;
    const sourceCardId = event.dataTransfer?.getData("text/plain") || "";
    if (sourceCardId) {
      void installEditComponent(sourceCardId);
    }
  });
  const editCard = byID("game-edit-card");
  editCard?.addEventListener("pointerdown", onEditPointerDown);
  editCard?.addEventListener("pointermove", onEditPointerMove);
  editCard?.addEventListener("pointerup", onEditPointerUp);
  editCard?.addEventListener("pointercancel", onEditPointerCancel);
  editCard?.addEventListener("contextmenu", (event) => {
    if (editComponentHit(event)) {
      event.preventDefault();
    }
  });
  bindSliderInputEvents(editCard, "edit");
  bindSliderInputEvents(byID("game-library-list"), "library");
  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      closeComponentOverlay(overlayRoot());
    }
  });
}
function bindSliderInputEvents(root, scope) {
  root?.addEventListener("pointerdown", stopSliderInputEvent);
  root?.addEventListener("click", stopSliderInputEvent);
  root?.addEventListener("input", (event) => {
    const input = sliderInputFromEvent(event);
    if (!input) return;
    updateSliderValueDisplay(input);
    event.stopPropagation();
  });
  root?.addEventListener("change", (event) => {
    const input = sliderInputFromEvent(event);
    if (!input) return;
    updateSliderValueDisplay(input);
    event.stopPropagation();
    void commitSliderInputValue(scope, input);
  });
}
function stopSliderInputEvent(event) {
  if (sliderInputFromEvent(event)) {
    event.stopPropagation();
  }
}
function sliderInputFromEvent(event) {
  const target = event.target;
  if (!(target instanceof Element)) return null;
  const input = target.closest("input[data-slider-input]");
  return input && input.type === "range" ? input : null;
}
function isSliderInputTarget(event) {
  return Boolean(sliderInputFromEvent(event));
}
function updateSliderValueDisplay(input) {
  const component = input.closest("[data-component-id][data-component-kind='slider']");
  const value = component?.querySelector("[data-slider-value]");
  if (value) value.textContent = input.value;
}
async function commitSliderInputValue(scope, input) {
  if (controlBusy) return;
  const component = input.closest("[data-component-id][data-component-kind='slider']");
  const componentId = component?.dataset.componentId || "";
  const value = Number(input.value);
  if (!componentId || !Number.isFinite(value)) return;
  controlBusy = true;
  try {
    if (scope === "edit") {
      if (!latestSession?.editSession) return;
      renderSession(await applyGameEditControl(componentId, "value", value));
      return;
    }
    if (scope === "library") {
      const cardId2 = input.closest(".game-library-card")?.dataset.cardId || input.closest("[data-card-id]")?.dataset.cardId || "";
      if (!cardId2) return;
      renderSession(await applyGameLibraryComponentControl(cardId2, componentId, "slider", "value", value));
      return;
    }
    const cardId = latestSession?.activeWorldCardId || activeCardPreview()?.dataset.cardId || "";
    if (!cardId) return;
    renderSession(await applyGameCardComponentControl(cardId, componentId, "slider", "value", value));
  } catch (error) {
    if (scope === "edit") {
      setEditStatus(errorMessage(error), true);
    } else {
      setStatus(errorMessage(error), true);
    }
  } finally {
    controlBusy = false;
  }
}
function onActivePointerDown(event) {
  if (busy || latestSession?.editSession) return;
  if (event.pointerType === "mouse" && event.button !== 0) return;
  if (isSliderInputTarget(event)) return;
  const hit = activeComponentHit(event);
  if (!hit) return;
  const position = componentPercentPosition(hit.element, hit.preview, hit.componentKind);
  const longPressTimer = window.setTimeout(() => {
    const state = activePressState;
    if (!state || state.pointerId !== event.pointerId || state.dragging) return;
    state.longPressFired = true;
    void selectActiveComponent(state.hit);
  }, longPressDelayMS);
  activePressState = {
    hit,
    pointerId: event.pointerId,
    startClientX: event.clientX,
    startClientY: event.clientY,
    startX: position.x,
    startY: position.y,
    nextX: position.x,
    nextY: position.y,
    dragging: false,
    longPressFired: false,
    longPressTimer
  };
  try {
    event.currentTarget.setPointerCapture(event.pointerId);
  } catch {
  }
  event.preventDefault();
  event.stopPropagation();
}
function onActivePointerMove(event) {
  const state = activePressState;
  if (!state || state.pointerId !== event.pointerId) return;
  const dx = event.clientX - state.startClientX;
  const dy = event.clientY - state.startClientY;
  const distance = Math.hypot(dx, dy);
  if (distance > dragThresholdPX && !state.longPressFired) {
    window.clearTimeout(state.longPressTimer);
  }
  if (state.longPressFired) {
    event.preventDefault();
    event.stopPropagation();
    return;
  }
  if (!canDragActiveComponent(state.hit.componentKind)) return;
  if (!state.dragging && distance >= dragThresholdPX) {
    state.dragging = true;
    closeComponentOverlay(overlayRoot());
    state.hit.element.dataset.dragging = "true";
    setStatus("Release to place " + componentTitle(state.hit.componentKind).toLowerCase() + ".");
  }
  if (!state.dragging) return;
  const rect = state.hit.preview.getBoundingClientRect();
  if (!rect.width || !rect.height) return;
  state.nextX = clampDragPercent(state.startX + dx / rect.width * 100);
  state.nextY = clampDragPercent(state.startY + dy / rect.height * 100);
  state.hit.element.style.left = `${state.nextX.toFixed(2)}%`;
  state.hit.element.style.top = `${state.nextY.toFixed(2)}%`;
  event.preventDefault();
  event.stopPropagation();
}
function onActivePointerUp(event) {
  const state = activePressState;
  if (!state || state.pointerId !== event.pointerId) return;
  const shouldCommit = state.dragging;
  const hit = state.hit;
  const x = Math.round(state.nextX);
  const y = Math.round(state.nextY);
  cleanupActivePressState(event.currentTarget);
  if (shouldCommit) {
    void applyActivePosition(hit, x, y);
  }
  event.preventDefault();
  event.stopPropagation();
}
function onActivePointerCancel(event) {
  const state = activePressState;
  if (!state || state.pointerId !== event.pointerId) return;
  cleanupActivePressState(event.currentTarget);
  if (latestSession) {
    renderSession(latestSession);
  }
}
function cleanupActivePressState(target) {
  const state = activePressState;
  if (!state) return;
  window.clearTimeout(state.longPressTimer);
  delete state.hit.element.dataset.dragging;
  if (target) {
    try {
      target.releasePointerCapture(state.pointerId);
    } catch {
    }
  }
  activePressState = null;
}
function onEditPointerDown(event) {
  if (busy || !latestSession?.editSession) return;
  if (event.pointerType === "mouse" && event.button !== 0) return;
  if (isSliderInputTarget(event)) return;
  const hit = editComponentHit(event);
  if (!hit) return;
  const position = componentPercentPosition(hit.element, hit.preview, hit.componentKind);
  editPressState = {
    hit,
    pointerId: event.pointerId,
    startClientX: event.clientX,
    startClientY: event.clientY,
    startX: position.x,
    startY: position.y,
    nextX: position.x,
    nextY: position.y,
    dragging: false,
    longPressFired: false,
    longPressTimer: 0
  };
  try {
    event.currentTarget.setPointerCapture(event.pointerId);
  } catch {
  }
  event.preventDefault();
  event.stopPropagation();
}
function onEditPointerMove(event) {
  const state = editPressState;
  if (!state || state.pointerId !== event.pointerId) return;
  const dx = event.clientX - state.startClientX;
  const dy = event.clientY - state.startClientY;
  const distance = Math.hypot(dx, dy);
  if (!canDragEditComponent(state.hit.componentKind)) {
    event.preventDefault();
    event.stopPropagation();
    return;
  }
  if (!state.dragging && distance >= dragThresholdPX) {
    state.dragging = true;
    closeComponentOverlay(overlayRoot());
    state.hit.element.dataset.dragging = "true";
    setEditStatus("Release to place " + componentTitle(state.hit.componentKind).toLowerCase() + ".");
  }
  if (!state.dragging) return;
  const rect = state.hit.preview.getBoundingClientRect();
  if (!rect.width || !rect.height) return;
  state.nextX = clampDragPercent(state.startX + dx / rect.width * 100);
  state.nextY = clampDragPercent(state.startY + dy / rect.height * 100);
  state.hit.element.style.left = `${state.nextX.toFixed(2)}%`;
  state.hit.element.style.top = `${state.nextY.toFixed(2)}%`;
  event.preventDefault();
  event.stopPropagation();
}
function onEditPointerUp(event) {
  const state = editPressState;
  if (!state || state.pointerId !== event.pointerId) return;
  const shouldCommit = state.dragging;
  const hit = state.hit;
  const x = Math.round(state.nextX);
  const y = Math.round(state.nextY);
  cleanupEditPressState(event.currentTarget);
  if (shouldCommit) {
    void applyEditPosition(hit, x, y);
  } else {
    void selectEditComponent(hit);
  }
  event.preventDefault();
  event.stopPropagation();
}
function onEditPointerCancel(event) {
  const state = editPressState;
  if (!state || state.pointerId !== event.pointerId) return;
  cleanupEditPressState(event.currentTarget);
  if (latestSession) {
    renderSession(latestSession);
  }
}
function cleanupEditPressState(target) {
  const state = editPressState;
  if (!state) return;
  delete state.hit.element.dataset.dragging;
  if (target) {
    try {
      target.releasePointerCapture(state.pointerId);
    } catch {
    }
  }
  editPressState = null;
}
function activeComponentHit(event) {
  const preview = activeCardPreview();
  const target = event.target;
  if (!preview || !(target instanceof Node) || !preview.contains(target)) return null;
  const cardId = preview.dataset.cardId || latestSession?.activeWorldCardId || "";
  if (!cardId) return null;
  const elementTarget = target instanceof Element ? target : null;
  const component = elementTarget?.closest("[data-component-id][data-component-kind]");
  if (component && preview.contains(component)) {
    const componentKind2 = component.dataset.componentKind || "";
    if (componentKind2 && componentKind2 !== "card") {
      return {
        cardId,
        componentId: component.dataset.componentId || "",
        componentKind: componentKind2,
        element: component,
        preview
      };
    }
  }
  const rect = preview.getBoundingClientRect();
  const x = event.clientX - rect.left;
  const y = event.clientY - rect.top;
  const band = Math.max(12, Math.min(24, Math.min(rect.width, rect.height) * 0.08));
  const componentKind = x <= band || y <= band || rect.width - x <= band || rect.height - y <= band ? "border" : "background";
  return {
    cardId,
    componentId: "",
    componentKind,
    element: preview,
    preview
  };
}
function activeCardPreview() {
  const root = byID("game-world-card");
  const preview = root?.firstElementChild;
  return preview instanceof HTMLElement ? preview : null;
}
function editComponentHit(event) {
  const preview = editCardPreview();
  const target = event.target;
  if (!preview || !(target instanceof Node) || !preview.contains(target)) return null;
  if (isSliderInputTarget(event)) return null;
  const cardId = latestSession?.editSession?.draftCard.id || preview.dataset.cardId || "";
  if (!cardId) return null;
  const elementTarget = target instanceof Element ? target : null;
  const component = elementTarget?.closest("[data-component-id][data-component-kind]");
  if (component && preview.contains(component)) {
    const componentKind = component.dataset.componentKind || "";
    if (componentKind === "slider") {
      return {
        cardId,
        componentId: component.dataset.componentId || "",
        componentKind,
        element: component,
        preview
      };
    }
  }
  const rect = preview.getBoundingClientRect();
  const x = event.clientX - rect.left;
  const y = event.clientY - rect.top;
  const band = Math.max(12, Math.min(24, Math.min(rect.width, rect.height) * 0.08));
  if (x <= band || y <= band || rect.width - x <= band || rect.height - y <= band) {
    return {
      cardId,
      componentId: "",
      componentKind: "border",
      element: preview,
      preview
    };
  }
  return null;
}
function editCardPreview() {
  const root = byID("game-edit-card");
  const preview = root?.firstElementChild;
  return preview instanceof HTMLElement ? preview : null;
}
function componentPercentPosition(element, preview, componentKind) {
  const styleX = parsePercent(element.style.left) ?? parsePercent(getComputedStyle(element).left);
  const styleY = parsePercent(element.style.top) ?? parsePercent(getComputedStyle(element).top);
  if (styleX !== null && styleY !== null) {
    return { x: styleX, y: styleY };
  }
  const elementRect = element.getBoundingClientRect();
  const previewRect = preview.getBoundingClientRect();
  if (!previewRect.width || !previewRect.height) {
    return { x: 50, y: 50 };
  }
  const anchorX = componentKind === "shape" ? elementRect.left : elementRect.left + elementRect.width / 2;
  const anchorY = componentKind === "shape" ? elementRect.top : elementRect.top + elementRect.height / 2;
  return {
    x: (anchorX - previewRect.left) / previewRect.width * 100,
    y: (anchorY - previewRect.top) / previewRect.height * 100
  };
}
function parsePercent(value) {
  const match = String(value || "").trim().match(/^(-?\d+(?:\.\d+)?)%$/);
  if (!match) return null;
  const parsed = Number(match[1]);
  return Number.isFinite(parsed) ? parsed : null;
}
function canDragActiveComponent(componentKind) {
  return componentKind === "textarea" || componentKind === "shape" || componentKind === "image" || componentKind === "slider";
}
function canDragEditComponent(componentKind) {
  return componentKind === "slider";
}
function componentTitle(componentKind) {
  switch (componentKind) {
    case "background":
      return "Background";
    case "border":
      return "Border";
    case "textarea":
      return "Text";
    case "shape":
      return "Shape";
    case "image":
      return "Image";
    case "slider":
      return "Slider";
    default:
      return "Component";
  }
}
function clampDragPercent(value) {
  if (value < 1) return 1;
  if (value > 99) return 99;
  return value;
}
async function loadSession() {
  setStatus("Loading scene...");
  try {
    renderSession(await fetchGameSession());
  } catch (error) {
    setStatus(errorMessage(error), true);
  }
}
async function reset() {
  if (busy) return;
  busy = true;
  closeComponentOverlay(overlayRoot());
  setStatus("Resetting scene...");
  try {
    renderSession(await resetGameSession());
  } catch (error) {
    setStatus(errorMessage(error), true);
  } finally {
    busy = false;
  }
}
async function cycle(direction) {
  if (busy) return;
  busy = true;
  closeComponentOverlay(overlayRoot());
  try {
    renderSession(await cycleGameCard(direction));
  } catch (error) {
    setStatus(errorMessage(error), true);
  } finally {
    busy = false;
  }
}
async function collect(cardId) {
  if (busy) return;
  busy = true;
  closeComponentOverlay(overlayRoot());
  try {
    renderSession(await collectGameCard(cardId));
  } catch (error) {
    setStatus(errorMessage(error), true);
  } finally {
    busy = false;
  }
}
async function play(sourceCardId, targetCardId) {
  if (busy) return;
  busy = true;
  closeComponentOverlay(overlayRoot());
  try {
    renderSession(await playGameCard(sourceCardId, targetCardId));
  } catch (error) {
    setStatus(errorMessage(error), true);
  } finally {
    busy = false;
  }
}
async function selectActiveComponent(hit) {
  if (busy || latestSession?.editSession) return;
  busy = true;
  try {
    renderSession(await selectGameCardComponent(hit.cardId, hit.componentId, hit.componentKind), { openActiveOverlay: true });
  } catch (error) {
    setStatus(errorMessage(error), true);
  } finally {
    busy = false;
  }
}
async function applyActivePosition(hit, x, y) {
  if (busy || latestSession?.editSession) return;
  busy = true;
  try {
    renderSession(await applyGameCardComponentControl(hit.cardId, hit.componentId, hit.componentKind, "position", { x, y }));
  } catch (error) {
    if (latestSession) {
      renderSession(latestSession);
    }
    setStatus(errorMessage(error), true);
  } finally {
    busy = false;
  }
}
async function selectEditComponent(hit) {
  if (busy || !latestSession?.editSession) return;
  busy = true;
  try {
    renderSession(await selectGameEditComponent(hit.componentId, hit.componentKind), { openOverlay: true });
  } catch (error) {
    setEditStatus(errorMessage(error), true);
  } finally {
    busy = false;
  }
}
async function applyEditPosition(hit, x, y) {
  if (busy || !latestSession?.editSession) return;
  busy = true;
  try {
    renderSession(await applyGameEditControl(hit.componentId, "position", { x, y }));
  } catch (error) {
    if (latestSession) {
      renderSession(latestSession);
    }
    setEditStatus(errorMessage(error), true);
  } finally {
    busy = false;
  }
}
async function startEdit(cardId) {
  if (busy) return;
  busy = true;
  closeComponentOverlay(overlayRoot());
  try {
    renderSession(await startGameEdit(cardId));
  } catch (error) {
    setStatus(errorMessage(error), true);
  } finally {
    busy = false;
  }
}
async function installEditComponent(componentCardId) {
  if (busy || !latestSession?.editSession) return;
  const card = latestSession.library.find((candidate) => candidate.id === componentCardId);
  if (!card || !componentKindForCard(card) || pendingComponentIds().has(componentCardId)) return;
  busy = true;
  try {
    renderSession(await installGameEditComponent(componentCardId), { openOverlay: true });
  } catch (error) {
    setEditStatus(errorMessage(error), true);
  } finally {
    busy = false;
  }
}
async function saveEdit() {
  if (busy || !latestSession?.editSession) return;
  busy = true;
  closeComponentOverlay(overlayRoot());
  try {
    renderSession(await saveGameEdit());
  } catch (error) {
    setEditStatus(errorMessage(error), true);
  } finally {
    busy = false;
  }
}
async function cancelEdit() {
  if (busy || !latestSession?.editSession) return;
  busy = true;
  closeComponentOverlay(overlayRoot());
  try {
    renderSession(await cancelGameEdit());
  } catch (error) {
    setEditStatus(errorMessage(error), true);
  } finally {
    busy = false;
  }
}
function renderSession(session, options = {}) {
  latestSession = session;
  renderActiveCard(session.activeWorldCard);
  renderLibrary(session.library);
  renderEditMode(session);
  renderAction(session.activeWorldCard);
  setStatus(session.message || "");
  const progress = byID("game-progress");
  if (progress) {
    progress.textContent = `${session.library.length} cards in library`;
  }
  const count = byID("game-library-count");
  if (count) {
    count.textContent = session.library.length ? `${session.library.length} card${session.library.length === 1 ? "" : "s"}` : "Empty";
  }
  if (options.openActiveOverlay) {
    openActiveEditingOverlay(session.activeEditingOverlay);
    return;
  }
  if (!session.editSession) {
    closeComponentOverlay(overlayRoot());
    return;
  }
  if (options.openOverlay) {
    openEditingOverlay(session.editSession.editingOverlay);
  }
}
function renderActiveCard(card) {
  const root = byID("game-world-card");
  if (!root) return;
  root.innerHTML = card.preview_html;
  root.dataset.activeCardId = card.id;
  root.dataset.cardKind = card.kind;
}
function renderEditMode(session) {
  const shell = byID("card-workspace");
  const mode = byID("game-edit-mode");
  const cardRoot = byID("game-edit-card");
  const title = byID("game-edit-title");
  if (!shell || !mode || !cardRoot) return;
  const editing = Boolean(session.editSession);
  shell.dataset.editing = editing ? "true" : "false";
  mode.hidden = !editing;
  if (!session.editSession) {
    cardRoot.innerHTML = "";
    renderComponentTray([]);
    return;
  }
  cardRoot.innerHTML = session.editSession.draftCard.preview_html;
  cardRoot.dataset.cardId = session.editSession.draftCard.id;
  if (title) title.textContent = session.editSession.draftCard.name;
  setEditStatus(session.message || "Drag a component onto the card.");
  renderComponentTray(session.library);
}
function renderAction(card) {
  const collect2 = byID("game-collect-card");
  if (!collect2) return;
  collect2.hidden = !card.collectible || Boolean(card.collected);
  collect2.disabled = !card.collectible || Boolean(card.collected);
}
function renderLibrary(cards) {
  const root = byID("game-library-list");
  if (!root) return;
  root.innerHTML = "";
  if (!cards.length) {
    root.textContent = "No cards collected.";
    return;
  }
  cards.forEach((card) => {
    const item = document.createElement("div");
    item.setAttribute("role", "button");
    item.tabIndex = 0;
    item.className = "game-library-card";
    item.draggable = true;
    item.dataset.cardId = card.id;
    item.addEventListener("dragstart", (event) => {
      event.dataTransfer?.setData("text/plain", card.id);
      event.dataTransfer.effectAllowed = "move";
    });
    item.addEventListener("click", () => {
      void handleLibraryClick(card);
    });
    item.addEventListener("keydown", (event) => {
      if (event.key === "Enter" || event.key === " ") {
        event.preventDefault();
        void handleLibraryClick(card);
      }
    });
    const preview = document.createElement("div");
    preview.className = "game-card-thumbnail";
    preview.innerHTML = card.preview_html;
    stopSliderInputEventsInPreview(preview);
    const label = document.createElement("div");
    label.className = "game-library-card-name";
    label.textContent = card.name;
    item.append(preview, label);
    const action = libraryAction(card);
    if (action) {
      item.appendChild(action);
    }
    root.appendChild(item);
  });
}
function libraryAction(card) {
  if (!isEditableCard(card)) return null;
  const action = document.createElement("button");
  action.type = "button";
  action.className = "game-library-action";
  action.textContent = "Edit";
  action.title = "Edit this card";
  action.addEventListener("pointerdown", (event) => {
    event.stopPropagation();
  });
  action.addEventListener("click", (event) => {
    event.preventDefault();
    event.stopPropagation();
    void startEdit(card.id);
  });
  return action;
}
function stopSliderInputEventsInPreview(preview) {
  for (const eventName of ["pointerdown", "click"]) {
    preview.addEventListener(eventName, (event) => {
      if (sliderInputFromEvent(event)) {
        event.stopPropagation();
      }
    });
  }
}
function renderComponentTray(cards) {
  const root = byID("game-edit-component-tray");
  const count = byID("game-edit-component-count");
  if (!root) return;
  root.innerHTML = "";
  const pending = pendingComponentIds();
  const components = cards.filter((card) => componentKindForCard(card));
  if (count) {
    count.textContent = components.length ? `${components.length} component${components.length === 1 ? "" : "s"}` : "Empty";
  }
  if (!components.length) {
    root.textContent = "No component cards collected.";
    return;
  }
  components.forEach((card) => {
    const disabled = pending.has(card.id);
    const item = document.createElement("button");
    item.type = "button";
    item.className = "game-edit-component-card";
    item.dataset.cardId = card.id;
    item.dataset.componentKind = componentKindForCard(card);
    if (disabled) item.dataset.pending = "true";
    item.draggable = !disabled;
    item.disabled = disabled;
    item.addEventListener("dragstart", (event) => {
      if (disabled) return;
      event.dataTransfer?.setData("text/plain", card.id);
      event.dataTransfer.effectAllowed = "move";
    });
    item.addEventListener("click", () => {
      if (!disabled) void installEditComponent(card.id);
    });
    const preview = document.createElement("div");
    preview.className = "game-card-thumbnail";
    preview.innerHTML = card.preview_html;
    const label = document.createElement("div");
    label.className = "game-library-card-name";
    label.textContent = card.name;
    item.append(preview, label);
    root.appendChild(item);
  });
}
async function handleLibraryClick(card) {
  if (isEditableCard(card)) {
    await startEdit(card.id);
    return;
  }
  setStatus("Drag this card onto the active world card to use it.");
}
function openActiveEditingOverlay(overlay = latestSession?.activeEditingOverlay) {
  if (!overlay) {
    closeComponentOverlay(overlayRoot());
    setStatus("Long press a card component to edit it.", true);
    return;
  }
  openComponentOverlay({
    root: overlayRoot(),
    overlay,
    onClose: () => void 0,
    onControl: (control, value) => {
      void applyActiveControl(overlay, control, value);
    }
  });
}
function openEditingOverlay(overlay = latestSession?.editSession?.editingOverlay) {
  if (!latestSession?.editSession) {
    setStatus("Start editing a card first.", true);
    return;
  }
  if (!overlay) {
    setEditStatus("Add or select a component to edit its properties.", true);
    return;
  }
  openComponentOverlay({
    root: overlayRoot(),
    overlay,
    onClose: () => void 0,
    onControl: (control, value) => {
      void applyEditControl(overlay, control, value);
    }
  });
}
async function applyActiveControl(overlay, control, value) {
  if (controlBusy) return;
  const cardId = latestSession?.activeWorldCardId || "";
  if (!cardId) return;
  controlBusy = true;
  try {
    const snapshot = await applyGameCardComponentControl(cardId, overlay.componentId, overlay.componentKind, control.control, value);
    renderSession(snapshot, { openActiveOverlay: true });
  } catch (error) {
    setStatus(errorMessage(error), true);
  } finally {
    controlBusy = false;
  }
}
async function applyEditControl(overlay, control, value) {
  if (controlBusy) return;
  controlBusy = true;
  try {
    const snapshot = await applyGameEditControl(overlay.componentId, control.control, value);
    renderSession(snapshot, { openOverlay: true });
  } catch (error) {
    setEditStatus(errorMessage(error), true);
  } finally {
    controlBusy = false;
  }
}
function isEditableCard(card) {
  return Boolean(card.state?.editable);
}
function componentKindForCard(card) {
  const value = card.state?.componentKind;
  return typeof value === "string" ? value : "";
}
function pendingComponentIds() {
  return new Set(latestSession?.editSession?.pendingConsumedComponentIds || []);
}
function overlayRoot() {
  return byID("stage-overlay-root");
}
function setStatus(message, isError = false) {
  const status = byID("game-status");
  if (!status) return;
  status.textContent = message;
  status.dataset.tone = isError ? "error" : "info";
}
function setEditStatus(message, isError = false) {
  const status = byID("game-edit-status");
  if (!status) return;
  status.textContent = message;
  status.dataset.tone = isError ? "error" : "info";
}
function errorMessage(error) {
  return error instanceof Error ? error.message : "Something went wrong.";
}

// web/src/app.ts
document.addEventListener("DOMContentLoaded", () => {
  initDesigner();
  initGameStage();
});
//# sourceMappingURL=app.js.map
