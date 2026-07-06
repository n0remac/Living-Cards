var __defProp = Object.defineProperty;
var __defNormalProp = (obj, key, value) => key in obj ? __defProp(obj, key, { enumerable: true, configurable: true, writable: true, value }) : obj[key] = value;
var __publicField = (obj, key, value) => __defNormalProp(obj, typeof key !== "symbol" ? key + "" : key, value);

// web/src/api.ts
var FragmentGenerationError = class extends Error {
  constructor(message, rawResponse, issues = []) {
    super(message);
    __publicField(this, "rawResponse");
    __publicField(this, "issues");
    this.name = "FragmentGenerationError";
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
async function generateFragment(target, instruction, update = false) {
  const response = await fetch("/api/draft-card/fragments/" + encodeURIComponent(target), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ instruction, update })
  });
  if (!response.ok) {
    throw await readFragmentError(response, "Failed to generate fragment.");
  }
  return await response.json();
}
async function applyDraftFragment(generatedFragment) {
  const response = await fetch("/api/draft-card/apply-fragment", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ generated_fragment: generatedFragment })
  });
  if (!response.ok) {
    throw await readFragmentError(response, "Failed to apply fragment.");
  }
  return await response.json();
}
async function fetchDesignLibrary(target = "") {
  const suffix = target ? "?target=" + encodeURIComponent(target) : "";
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
async function saveControllerCard(templateCardId, document2) {
  const response = await fetch("/api/game/save-controller", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ templateCardId, document: document2 })
  });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to save controller."));
  }
  return await response.json();
}
async function addDraftComponent(componentType, fragment) {
  const response = await fetch("/api/draft-card/components", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ componentType, fragment })
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
async function readFragmentError(response, fallback) {
  const contentType = response.headers.get("Content-Type") || "";
  if (contentType.includes("application/json")) {
    try {
      const payload = await response.json();
      const message = String(payload.message || fallback).trim();
      const rawResponse = String(payload.raw_response || "").trim();
      const issues = Array.isArray(payload.issues) ? payload.issues : [];
      if (rawResponse || issues.length) {
        return new FragmentGenerationError(message, rawResponse, issues);
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

// web/src/designer/fragments.ts
async function generateTargetFragment(target, instruction, update = false) {
  return await generateFragment(target, instruction, update);
}
function parseGeneratedFragment(raw) {
  const trimmed = String(raw || "").trim();
  if (!trimmed || trimmed === "{}") {
    throw new Error("Generate or paste a fragment before applying.");
  }
  let parsed;
  try {
    parsed = JSON.parse(trimmed);
  } catch {
    throw new Error("Generated fragment is not valid JSON.");
  }
  return normalizeGeneratedFragment(parsed);
}
function normalizeGeneratedFragment(value) {
  if (!value || typeof value !== "object") {
    throw new Error("Generated fragment must be a JSON object.");
  }
  const record = value;
  const target = String(record.target || "").trim();
  const fragment = record.fragment;
  if (!fragment || typeof fragment !== "object") {
    throw new Error("Generated fragment must include a fragment object.");
  }
  return {
    target,
    description: String(record.description || savedDesignFallbackName(target)).trim(),
    fragment: cloneJSON(fragment)
  };
}
function isFragmentGenerationError(error) {
  return error instanceof FragmentGenerationError;
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
function savedDesignFallbackName(target) {
  switch (target) {
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
      void generateFragment2(event);
    });
  }
  const apply = byID("apply-fragment-btn");
  if (apply) {
    apply.addEventListener("click", () => {
      void applyFragment();
    });
  }
  const editor = byID("fragment-preview");
  if (editor) {
    editor.addEventListener("input", () => {
      const hasCandidate = hasEditableFragmentCandidate();
      const description = byID("fragment-description");
      if (description && hasCandidate) {
        description.textContent = "Edited fragment. Apply to validate it against the selected target.";
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
  const target = byID("fragment-target");
  if (target) {
    target.addEventListener("change", () => {
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
  renderProposedFragment(null);
  setSaveEnabled(false);
  void loadDesigner();
}
async function resetDraft() {
  setDesignerStatus("Resetting draft...", false);
  try {
    const response = await resetDraftCard();
    replacePreviewHTML(response.preview_html);
    renderProposedFragment(null);
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
    renderProposedFragment(null);
    renderDesignLibraryItems(response.library);
    setSaveEnabled(false);
    setDesignerStatus("Ready.", false);
  } catch (error) {
    setDesignerStatus(error instanceof Error ? error.message : "Failed to load draft card.", true);
  }
}
function renderProposedFragment(fragment) {
  const preview = byID("fragment-preview");
  const description = byID("fragment-description");
  const apply = byID("apply-fragment-btn");
  if (preview) {
    preview.value = fragment ? JSON.stringify(fragment, null, 2) : "{}";
  }
  if (description) {
    description.textContent = fragment ? fragment.description : "No generated fragment yet.";
  }
  if (apply) {
    apply.disabled = !fragment;
  }
}
function renderFailedFragment(rawResponse, message, issues = []) {
  const preview = byID("fragment-preview");
  const description = byID("fragment-description");
  const apply = byID("apply-fragment-btn");
  if (preview) {
    preview.value = rawResponse || "{}";
  }
  if (description) {
    const issueSummary = formatIssues(issues);
    description.textContent = issueSummary ? "Generation failed: " + issueSummary + " Edit the response below, then apply it to validate." : "Generation failed. Edit the response below, then apply it to validate.";
  }
  if (apply) {
    apply.disabled = !hasEditableFragmentCandidate();
  }
  setSaveEnabled(false);
  setDesignerStatus(message, true);
}
async function loadDesignLibrary() {
  try {
    renderDesignLibraryItems(await fetchDesignLibrary(readTarget()));
  } catch (error) {
    setDesignerStatus(error instanceof Error ? error.message : "Failed to load design library.", true);
  }
}
function renderDesignLibraryItems(items) {
  const list = byID("design-library-list");
  if (!list) return;
  const target = readTarget();
  const visibleItems = items.filter((item) => item.target === target);
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
async function generateFragment2(event) {
  event.preventDefault();
  const target = readTarget();
  const isUpdate = event.submitter?.id === "update-fragment-btn";
  const input = byID("fragment-instruction");
  const instruction = String(input?.value || "").trim();
  if (!instruction) {
    setDesignerStatus("Instruction cannot be empty.", true);
    return;
  }
  setBusy(true, isUpdate);
  setSaveEnabled(false);
  setDesignerStatus(isUpdate ? "Updating fragment..." : "Generating fragment...", false);
  try {
    const fragment = await generateTargetFragment(target, instruction, isUpdate);
    renderProposedFragment(fragment);
    setDesignerStatus(isUpdate ? "Fragment updated. Review it before applying." : "Fragment generated. Review it before applying.", false);
  } catch (error) {
    if (isFragmentGenerationError(error)) {
      renderFailedFragment(error.rawResponse, error.message, error.issues);
      return;
    }
    setDesignerStatus(error instanceof Error ? error.message : "Fragment generation failed.", true);
  } finally {
    setBusy(false, isUpdate);
  }
}
async function applyFragment() {
  try {
    const fragment = readFragmentFromEditor();
    setBusy(true);
    setDesignerStatus("Applying fragment...", false);
    const response = await applyDraftFragment(fragment);
    replacePreviewHTML(response.preview_html);
    renderProposedFragment(response.normalized_fragment);
    renderDesignLibraryItems(response.library);
    setSaveEnabled(true);
    setDesignerStatus("Fragment applied to the preview.", false);
  } catch (error) {
    if (isFragmentGenerationError(error)) {
      const editor = byID("fragment-preview");
      renderFailedFragment(error.rawResponse || String(editor?.value || ""), error.message, error.issues);
      return;
    }
    setDesignerStatus(error instanceof Error ? error.message : "Fragment could not be applied.", true);
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
    renderProposedFragment(response.normalized_fragment);
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
function readTarget() {
  const select = byID("fragment-target");
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
async function addComponent(componentType) {
  try {
    setDesignerStatus("Adding component...", false);
    const response = await addDraftComponent(componentType);
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
  const generate = byID("generate-fragment-btn");
  const update = byID("update-fragment-btn");
  const apply = byID("apply-fragment-btn");
  if (generate) {
    generate.disabled = isBusy;
    generate.textContent = isBusy && !isUpdate ? "Generating..." : "Generate";
  }
  if (update) {
    update.disabled = isBusy;
    update.textContent = isBusy && isUpdate ? "Updating..." : "Update";
  }
  if (apply) {
    apply.disabled = isBusy || !hasEditableFragmentCandidate();
  }
}
function readFragmentFromEditor() {
  const editor = byID("fragment-preview");
  return parseGeneratedFragment(String(editor?.value || ""));
}
function hasEditableFragmentCandidate() {
  const editor = byID("fragment-preview");
  const raw = String(editor?.value || "").trim();
  return Boolean(raw && raw !== "{}");
}
function escapeHTML(value) {
  return String(value || "").replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;").replace(/'/g, "&#39;");
}

// web/src/game/GameController.ts
var latestSession = null;
var busy = false;
function initGameStage() {
  bindControls();
  bindControllerBuilder();
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
  const dropTarget = byID("game-world-card");
  dropTarget?.addEventListener("dragover", (event) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = "move";
  });
  dropTarget?.addEventListener("drop", (event) => {
    event.preventDefault();
    const sourceCardId = event.dataTransfer?.getData("text/plain") || "";
    const targetCardId = latestSession?.activeWorldCardId || "";
    if (sourceCardId && targetCardId) {
      void play(sourceCardId, targetCardId);
    }
  });
}
function bindControllerBuilder() {
  byID("controller-builder-close")?.addEventListener("click", closeControllerBuilder);
  byID("controller-builder-cancel")?.addEventListener("click", closeControllerBuilder);
  byID("controller-builder-save")?.addEventListener("click", () => {
    void saveController();
  });
  byID("controller-builder-overlay")?.addEventListener("click", (event) => {
    if (event.target === event.currentTarget) {
      closeControllerBuilder();
    }
  });
  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape" && document.body.classList.contains("controller-builder-open")) {
      closeControllerBuilder();
    }
  });
  const range = byID("controller-slider-input");
  const number = byID("controller-slider-number");
  range?.addEventListener("input", () => syncControllerInputs(Number(range.value)));
  number?.addEventListener("input", () => syncControllerInputs(Number(number.value)));
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
  try {
    renderSession(await playGameCard(sourceCardId, targetCardId));
  } catch (error) {
    setStatus(errorMessage(error), true);
  } finally {
    busy = false;
  }
}
function renderSession(session) {
  latestSession = session;
  renderActiveCard(session.activeWorldCard);
  renderLibrary(session.library);
  renderAction(session.activeWorldCard);
  setStatus(session.message || "");
  const progress = byID("game-progress");
  if (progress) {
    progress.textContent = `${session.library.length} cards collected`;
  }
  const count = byID("game-library-count");
  if (count) {
    count.textContent = session.library.length ? `${session.library.length} card${session.library.length === 1 ? "" : "s"}` : "Empty";
  }
}
function renderActiveCard(card) {
  const root = byID("game-world-card");
  if (!root) return;
  root.innerHTML = card.preview_html;
  root.dataset.activeCardId = card.id;
  root.dataset.cardKind = card.kind;
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
    item.className = "game-library-card";
    item.draggable = true;
    item.dataset.cardId = card.id;
    item.addEventListener("dragstart", (event) => {
      event.dataTransfer?.setData("text/plain", card.id);
      event.dataTransfer.effectAllowed = "move";
    });
    const preview = document.createElement("div");
    preview.innerHTML = card.preview_html;
    const label = document.createElement("div");
    label.className = "game-library-card-name";
    label.textContent = card.name;
    item.append(preview, label);
    if (card.id === "blank-controller") {
      const build = document.createElement("button");
      build.type = "button";
      build.className = "game-library-build";
      build.textContent = "Build";
      build.disabled = !hasLibraryCard("slider-component");
      build.title = build.disabled ? "Collect Slider Component first." : "Build Regulator Controller";
      build.addEventListener("pointerdown", (event) => {
        event.stopPropagation();
      });
      build.addEventListener("click", (event) => {
        event.preventDefault();
        event.stopPropagation();
        openControllerBuilder();
      });
      item.appendChild(build);
    }
    root.appendChild(item);
  });
}
function openControllerBuilder() {
  if (!hasLibraryCard("blank-controller")) {
    setStatus("Collect the Blank Controller first.", true);
    return;
  }
  if (!hasLibraryCard("slider-component")) {
    setStatus("Collect the Slider Component first.", true);
    return;
  }
  syncControllerInputs(existingControllerValue() ?? 50);
  document.body.classList.add("controller-builder-open");
}
function closeControllerBuilder() {
  document.body.classList.remove("controller-builder-open");
}
async function saveController() {
  if (busy) return;
  busy = true;
  setStatus("Saving controller...");
  try {
    const value = readControllerValue();
    const snapshot = await saveControllerCard("blank-controller", createControllerDocument(value));
    closeControllerBuilder();
    renderSession(snapshot);
  } catch (error) {
    setStatus(errorMessage(error), true);
  } finally {
    busy = false;
  }
}
function hasLibraryCard(cardId) {
  return Boolean(latestSession?.library.some((card) => card.id === cardId));
}
function existingControllerValue() {
  const controller = latestSession?.library.find((card) => card.id === "generator-regulator-controller");
  return controller ? sliderValueFromNode(controller.document.root) : null;
}
function sliderValueFromNode(node) {
  if (node.type === "slider" && node.fragment && typeof node.fragment.value === "number") {
    return clampControllerValue(node.fragment.value);
  }
  for (const child of node.children || []) {
    const value = sliderValueFromNode(child);
    if (value !== null) return value;
  }
  return null;
}
function syncControllerInputs(value) {
  const normalized = clampControllerValue(value);
  const range = byID("controller-slider-input");
  const number = byID("controller-slider-number");
  if (range) range.value = String(normalized);
  if (number) number.value = String(normalized);
}
function readControllerValue() {
  const number = byID("controller-slider-number");
  return clampControllerValue(Number(number?.value ?? 50));
}
function clampControllerValue(value) {
  if (!Number.isFinite(value)) return 50;
  return Math.max(0, Math.min(100, Math.round(value)));
}
function createControllerDocument(value) {
  return {
    card_id: "generator-regulator-controller",
    name: "Regulator Controller",
    root: {
      id: "generator-regulator-controller-root",
      type: "card",
      fragment: {
        padding_px: 18,
        shadow: "0 24px 60px rgba(8,47,73,0.34)"
      },
      children: [{
        id: "regulator-output-slider",
        type: "slider",
        fragment: {
          label: "Output",
          min: 0,
          max: 100,
          step: 1,
          value: clampControllerValue(value)
        }
      }]
    }
  };
}
function setStatus(message, isError = false) {
  const status = byID("game-status");
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
