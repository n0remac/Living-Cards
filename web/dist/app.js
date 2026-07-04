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
async function fetchInteractiveDraftCard() {
  const response = await fetch("/api/draft-card/interactive", { cache: "no-store" });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to load interactive card."));
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
async function interactWithComponent(componentId, trait, interaction, x, y) {
  const response = await fetch("/api/draft-card/interact", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ componentId, trait, interaction, x, y })
  });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to apply interaction."));
  }
  return await response.json();
}
async function applyComponentControl(componentId, trait, control, value) {
  const response = await fetch("/api/draft-card/control-change", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ componentId, trait, control, value })
  });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to apply control."));
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
  const reset = byID("designer-reset-btn");
  if (reset) {
    reset.addEventListener("click", () => {
      void resetDraft();
    });
  }
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
      return select.value;
    default:
      return "background";
  }
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

// web/src/stage/cardMotion.ts
function animateCardTap(preview, x, y) {
  const rotateY = (x - 0.5) * 8;
  const rotateX = (0.5 - y) * 8;
  runAnimation(preview, [
    { transform: "translateY(0) scale(1) rotateX(0deg) rotateY(0deg)" },
    { transform: `translateY(-8px) scale(1.018) rotateX(${rotateX}deg) rotateY(${rotateY}deg)` },
    { transform: "translateY(0) scale(1) rotateX(0deg) rotateY(0deg)" }
  ], 260);
}
function animateInvalidTap(preview) {
  runAnimation(preview, [
    { transform: "translateX(0)" },
    { transform: "translateX(-7px)" },
    { transform: "translateX(6px)" },
    { transform: "translateX(-4px)" },
    { transform: "translateX(0)" }
  ], 230);
}
function runAnimation(element, keyframes, duration) {
  if (typeof element.animate !== "function") return;
  element.getAnimations().forEach((animation) => animation.cancel());
  element.animate(keyframes, {
    duration,
    easing: "cubic-bezier(.2,.8,.2,1)"
  });
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
  const panel = document.createElement("div");
  panel.dataset.stageControlOverlay = "component";
  panel.className = "stage-component-panel pointer-events-auto fixed grid max-h-[min(28rem,calc(100dvh-6rem))] w-72 gap-3 overflow-y-auto rounded-md border border-[var(--app-border-strong)] bg-[var(--app-surface-muted)] p-3 shadow-2xl backdrop-blur";
  panel.style.left = clampPX(options.anchorX, 12, window.innerWidth - 300) + "px";
  panel.style.top = clampPX(options.anchorY, 84, window.innerHeight - 360) + "px";
  stopCardTapEvents(panel);
  const header = document.createElement("div");
  header.className = "flex items-center justify-between gap-3";
  const title = document.createElement("div");
  title.className = "min-w-0 text-sm font-semibold text-[var(--app-fg)]";
  title.textContent = options.overlay.title;
  const close = document.createElement("button");
  close.type = "button";
  close.className = "h-8 rounded-md border border-[var(--app-border)] bg-[var(--app-panel)] px-2 text-xs font-semibold text-[var(--app-fg-soft)]";
  close.textContent = "Close";
  close.addEventListener("click", () => panel.remove());
  header.append(title, close);
  panel.appendChild(header);
  const controls = document.createElement("div");
  controls.className = "grid gap-3";
  options.overlay.controls.forEach((control) => {
    controls.appendChild(renderControl(control, (value) => options.onControl(control, value)));
  });
  panel.appendChild(controls);
  root.appendChild(panel);
}
function closeComponentOverlay(root) {
  root?.querySelectorAll("[data-stage-control-overlay]").forEach((element) => element.remove());
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
  const wrapper = controlWrapper(control.label);
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
  const wrapper = controlWrapper(control.label);
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
  const wrapper = controlWrapper(control.label);
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
  const wrapper = controlWrapper(control.label);
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
function controlWrapper(label) {
  const wrapper = document.createElement("label");
  wrapper.className = "grid gap-1 text-xs font-semibold uppercase text-[var(--app-fg-soft)]";
  const text = document.createElement("span");
  text.textContent = label;
  wrapper.appendChild(text);
  return wrapper;
}
function stopCardTapEvents(element) {
  for (const eventName of ["pointerdown", "pointermove", "pointerup", "pointercancel", "contextmenu"]) {
    element.addEventListener(eventName, (event) => {
      event.stopPropagation();
    });
  }
}
function clampPX(value, min, max) {
  if (!Number.isFinite(value)) return min;
  return Math.max(min, Math.min(max, value));
}
function hexOrFallback(value) {
  const trimmed = String(value || "").trim();
  return /^#[0-9a-fA-F]{6}$/.test(trimmed) ? trimmed : "#22c55e";
}

// web/src/stage/hitTesting.ts
var borderBandPX = 24;
function hitTestCard(event, preview) {
  const rect = preview.getBoundingClientRect();
  const x = clamp((event.clientX - rect.left) / rect.width);
  const y = clamp((event.clientY - rect.top) / rect.height);
  const localX = event.clientX - rect.left;
  const localY = event.clientY - rect.top;
  const inBorderBand = localX <= borderBandPX || localY <= borderBandPX || rect.width - localX <= borderBandPX || rect.height - localY <= borderBandPX;
  if (inBorderBand) {
    return cardRootHit("border", "border", x, y, event);
  }
  const target = event.target instanceof Element ? event.target : null;
  const component = target?.closest("[data-component-id][data-component-type]");
  const componentType = component?.dataset.componentType;
  if (component && componentType === "shape") {
    return componentHit(component, "shape", "shape", "geometry", x, y, event);
  }
  if (component && componentType === "textarea") {
    return componentHit(component, "textarea", "textarea", "text", x, y, event);
  }
  return cardRootHit("background", "background", x, y, event);
}
function cardRootHit(target, zone, x, y, event) {
  return {
    target,
    zone,
    componentId: "card-root",
    componentType: "card",
    trait: target === "border" ? "border" : "background",
    x,
    y,
    clientX: event.clientX,
    clientY: event.clientY
  };
}
function componentHit(element, target, zone, trait, x, y, event) {
  return {
    target,
    zone,
    componentId: element.dataset.componentId || "",
    componentType: element.dataset.componentType || target,
    trait,
    x,
    y,
    clientX: event.clientX,
    clientY: event.clientY
  };
}
function clamp(value) {
  if (!Number.isFinite(value)) return 0;
  return Math.max(0, Math.min(1, value));
}

// web/src/stage/overlays.ts
var visibleMS = 2600;
var notificationQueue = [];
var notificationHistory = [];
var notificationTimer = 0;
var activeNotification = false;
function initNotifications() {
  const section = notificationSection();
  if (!section) return;
  stopCardTapEvents2(section);
  const history = notificationHistoryPanel();
  if (history) {
    stopCardTapEvents2(history);
  }
  section.addEventListener("click", (event) => {
    event.stopPropagation();
    toggleHistory();
  });
  document.addEventListener("click", (event) => {
    const history2 = notificationHistoryPanel();
    if (!history2 || history2.classList.contains("hidden")) return;
    const target = event.target instanceof Node ? event.target : null;
    if (target && (history2.contains(target) || section.contains(target))) return;
    history2.classList.add("hidden");
  });
  renderHistory();
}
function renderEvents(root, events) {
  if (!root) return;
  (events || []).forEach((event) => showEvent(root, event));
}
function showMessage(root, message, tone = "info") {
  if (!root) return;
  enqueueNotification({ message, tone });
}
function showEvent(root, event) {
  switch (event.type) {
    case "fragmentApplied":
      return;
    case "controlChanged":
      return;
    case "xpGained":
      return;
    case "levelUp":
      return;
    case "componentLevelUp":
      showMessage(root, labelForComponent(event.componentType) + " level " + event.level);
      return;
    case "componentUnlocked":
      showMessage(root, event.message || labelForComponent(event.componentType) + " unlocked");
      return;
    case "componentSelected":
      return;
    case "overlayOpened":
      return;
    case "targetUnlocked":
      showMessage(root, labelForTarget(event.target) + " unlocked");
      return;
    case "modeUnlocked":
      showMessage(root, labelForTarget(event.target) + " " + event.mode + " unlocked");
      return;
    case "invalidAction":
      showMessage(root, event.message || "Locked", "error");
      return;
  }
}
function enqueueNotification(item) {
  notificationQueue.push(item);
  notificationHistory.unshift(item);
  if (notificationHistory.length > 30) {
    notificationHistory.length = 30;
  }
  renderHistory();
  showNextNotification();
}
function showNextNotification() {
  if (activeNotification) return;
  const current = notificationCurrent();
  const section = notificationSection();
  if (!current || !section) return;
  const item = notificationQueue.shift();
  if (!item) {
    current.textContent = "No notifications";
    section.dataset.tone = "empty";
    return;
  }
  activeNotification = true;
  current.textContent = item.message;
  section.dataset.tone = item.tone;
  window.clearTimeout(notificationTimer);
  notificationTimer = window.setTimeout(() => {
    activeNotification = false;
    showNextNotification();
  }, visibleMS);
}
function toggleHistory() {
  const history = notificationHistoryPanel();
  if (!history) return;
  history.classList.toggle("hidden");
  renderHistory();
}
function stopCardTapEvents2(element) {
  for (const eventName of ["pointerdown", "pointermove", "pointerup", "pointercancel", "contextmenu"]) {
    element.addEventListener(eventName, (event) => {
      event.stopPropagation();
    });
  }
}
function renderHistory() {
  const list = notificationHistoryList();
  if (!list) return;
  list.innerHTML = "";
  if (!notificationHistory.length) {
    const empty = document.createElement("div");
    empty.className = "stage-notification-history-empty";
    empty.textContent = "No notifications yet.";
    list.appendChild(empty);
    return;
  }
  notificationHistory.forEach((item) => {
    const row = document.createElement("div");
    row.className = "stage-notification-history-item";
    row.dataset.tone = item.tone;
    row.textContent = item.message;
    list.appendChild(row);
  });
}
function notificationSection() {
  return document.getElementById("stage-notification-section");
}
function notificationCurrent() {
  return document.getElementById("stage-notification-current");
}
function notificationHistoryPanel() {
  return document.getElementById("stage-notification-history");
}
function notificationHistoryList() {
  return document.getElementById("stage-notification-history-list");
}
function labelForTarget(target) {
  switch (target) {
    case "background":
      return "Background";
    case "border":
      return "Border";
    case "textarea":
      return "Text";
    default:
      return "Card";
  }
}
function labelForComponent(componentType) {
  switch (componentType) {
    case "textarea":
      return "Text";
    case "shape":
      return "Shape";
    default:
      return "Card";
  }
}

// web/src/stage/StageController.ts
var tapBusy = false;
var controlBusy = false;
var latestDocument = null;
var latestGameState = null;
var pressState = null;
var longPressMS = 540;
var moveCancelPX = 12;
function initStage(deps) {
  initNotifications();
  bindTapLayer();
  bindDesignerOverlay();
  bindReset(deps);
  document.addEventListener("living-card:interactive-refresh", () => {
    void loadInteractive(false);
  });
  void loadInteractive(true);
}
async function loadInteractive(showErrors) {
  try {
    const response = await fetchInteractiveDraftCard();
    applyInteractiveResponse(response);
  } catch (error) {
    if (showErrors) {
      showMessage(notificationRoot(), error instanceof Error ? error.message : "Failed to load card.", "error");
    }
  }
}
function applyInteractiveResponse(response) {
  latestDocument = response.document;
  latestGameState = response.gameState;
  replacePreviewHTML(response.preview_html);
  updateHUD(response.gameState);
  renderSelection(response.gameState);
}
function bindTapLayer() {
  const workspace = byID("card-workspace");
  if (!workspace) return;
  workspace.addEventListener("pointerdown", (event) => {
    startPress(event, workspace);
  });
  workspace.addEventListener("pointermove", (event) => {
    maybeCancelPressMove(event);
  });
  workspace.addEventListener("pointerup", (event) => {
    void finishPress(event, workspace);
  });
  workspace.addEventListener("pointercancel", () => {
    cancelPress();
  });
  workspace.addEventListener("contextmenu", (event) => {
    event.preventDefault();
  });
}
function startPress(event, workspace) {
  if (tapBusy || document.body.classList.contains("designer-open")) return;
  const preview = currentPreview();
  if (!preview) return;
  const hit = hitTestCard(event, preview);
  closeComponentOverlay(overlayRoot());
  cancelPress();
  workspace.setPointerCapture?.(event.pointerId);
  pressState = {
    pointerID: event.pointerId,
    hit,
    startX: event.clientX,
    startY: event.clientY,
    dragStartXPercent: 0,
    dragStartYPercent: 0,
    dragNextXPercent: 0,
    dragNextYPercent: 0,
    dragElement: null,
    dragging: false,
    longPressed: false,
    timer: window.setTimeout(() => {
      if (!pressState || pressState.pointerID !== event.pointerId) return;
      pressState.longPressed = true;
      void openLongPressControls(hit);
    }, longPressMS)
  };
}
function maybeCancelPressMove(event) {
  if (!pressState || pressState.pointerID !== event.pointerId || pressState.longPressed) return;
  if (pressState.dragging) {
    updateDragPreview(event, pressState);
    return;
  }
  const dx = event.clientX - pressState.startX;
  const dy = event.clientY - pressState.startY;
  if (Math.hypot(dx, dy) > moveCancelPX) {
    if (!startDrag(event, pressState)) {
      cancelPress();
    }
  }
}
async function finishPress(event, workspace) {
  if (!pressState || pressState.pointerID !== event.pointerId) return;
  const state = pressState;
  cancelPress();
  workspace.releasePointerCapture?.(event.pointerId);
  if (state.dragging) {
    await commitDragPosition(state);
    return;
  }
  if (state.longPressed) return;
  await handleCardTap(state.hit);
}
function cancelPress() {
  if (pressState) {
    window.clearTimeout(pressState.timer);
  }
  pressState = null;
}
function startDrag(event, state) {
  if (!canDragComponent(state.hit)) return false;
  const preview = currentPreview();
  if (!preview) return false;
  const element = componentElement(preview, state.hit.componentId);
  if (!element) return false;
  window.clearTimeout(state.timer);
  state.dragElement = element;
  state.dragging = true;
  state.dragStartXPercent = readElementPercent(element, preview, "left");
  state.dragStartYPercent = readElementPercent(element, preview, "top");
  state.dragNextXPercent = state.dragStartXPercent;
  state.dragNextYPercent = state.dragStartYPercent;
  updateDragPreview(event, state);
  return true;
}
function updateDragPreview(event, state) {
  const preview = currentPreview();
  const element = state.dragElement;
  if (!preview || !element) return;
  const rect = preview.getBoundingClientRect();
  const dxPercent = (event.clientX - state.startX) / Math.max(1, rect.width) * 100;
  const dyPercent = (event.clientY - state.startY) / Math.max(1, rect.height) * 100;
  state.dragNextXPercent = clampPercent(state.dragStartXPercent + dxPercent);
  state.dragNextYPercent = clampPercent(state.dragStartYPercent + dyPercent);
  element.style.left = formatPercent(state.dragNextXPercent);
  element.style.top = formatPercent(state.dragNextYPercent);
}
async function commitDragPosition(state) {
  if (controlBusy) return;
  const preview = currentPreview();
  controlBusy = true;
  try {
    const response = await applyComponentControl(state.hit.componentId, "position", "position", {
      x: Math.round(state.dragNextXPercent),
      y: Math.round(state.dragNextYPercent)
    });
    applyTapResponse(response);
    renderEvents(notificationRoot(), response.events);
    if (hasInvalidAction(response.events)) {
      const nextPreview = currentPreview();
      if (nextPreview) animateInvalidTap(nextPreview);
    }
  } catch (error) {
    if (preview) animateInvalidTap(preview);
    showMessage(notificationRoot(), error instanceof Error ? error.message : "Drag failed.", "error");
  } finally {
    controlBusy = false;
  }
}
async function handleCardTap(hit) {
  if (tapBusy || document.body.classList.contains("designer-open")) return;
  const preview = currentPreview();
  if (!preview) return;
  animateCardTap(preview, hit.x, hit.y);
  tapBusy = true;
  try {
    const response = await interactWithComponent(hit.componentId, hit.trait, "shortTap", hit.x, hit.y);
    applyTapResponse(response);
    const nextPreview = currentPreview();
    if (hasInvalidAction(response.events)) {
      if (nextPreview) animateInvalidTap(nextPreview);
    } else if (nextPreview) {
      animateCardTap(nextPreview, hit.x, hit.y);
    }
    renderEvents(notificationRoot(), response.events);
  } catch (error) {
    const nextPreview = currentPreview();
    if (nextPreview) animateInvalidTap(nextPreview);
    showMessage(notificationRoot(), error instanceof Error ? error.message : "Tap failed.", "error");
  } finally {
    tapBusy = false;
  }
}
async function openLongPressControls(hit) {
  if (tapBusy || document.body.classList.contains("designer-open")) return;
  const preview = currentPreview();
  tapBusy = true;
  try {
    const response = await interactWithComponent(hit.componentId, hit.trait, "longPress", hit.x, hit.y);
    applyTapResponse(response);
    renderEvents(notificationRoot(), response.events);
    if (response.overlay) {
      openOverlay(response.overlay, hit.clientX, hit.clientY);
      return;
    }
    if (preview) animateInvalidTap(preview);
  } catch (error) {
    if (preview) animateInvalidTap(preview);
    showMessage(notificationRoot(), error instanceof Error ? error.message : "Overlay failed.", "error");
  } finally {
    tapBusy = false;
  }
}
function openOverlay(overlay, anchorX, anchorY) {
  openComponentOverlay({
    root: overlayRoot(),
    overlay,
    anchorX,
    anchorY,
    onControl: (control, value) => {
      void applyControl(overlay.componentId, control.trait, control.control, value);
    }
  });
}
async function applyControl(componentId, trait, control, value) {
  if (controlBusy) return;
  controlBusy = true;
  try {
    const response = await applyComponentControl(componentId, trait, control, value);
    applyTapResponse(response);
    renderEvents(notificationRoot(), response.events);
    if (response.overlay) {
      openOverlay(response.overlay, window.innerWidth - 320, 110);
    }
  } catch (error) {
    showMessage(notificationRoot(), error instanceof Error ? error.message : "Control change failed.", "error");
  } finally {
    controlBusy = false;
  }
}
function applyTapResponse(response) {
  latestDocument = response.document;
  latestGameState = response.gameState;
  replacePreviewHTML(response.preview_html);
  updateHUD(response.gameState);
  renderSelection(response.gameState);
}
function bindDesignerOverlay() {
  const open = byID("designer-toggle-btn");
  const close = byID("designer-close-btn");
  const overlay = byID("designer-overlay");
  open?.addEventListener("click", () => {
    document.body.classList.add("designer-open");
  });
  close?.addEventListener("click", () => {
    document.body.classList.remove("designer-open");
  });
  overlay?.addEventListener("click", (event) => {
    if (event.target === overlay) {
      document.body.classList.remove("designer-open");
    }
  });
  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      document.body.classList.remove("designer-open");
    }
  });
}
function bindReset(deps) {
  const reset = byID("reset-draft-btn");
  reset?.addEventListener("click", () => {
    void deps.resetDraft();
  });
}
function updateHUD(gameState) {
  const level = byID("card-level");
  const xp = byID("card-xp");
  const taps = byID("card-taps");
  const bar = byID("card-xp-bar");
  const globalLevel = gameState.globalLevel || gameState.level || 1;
  const totalXP = gameState.totalXp ?? gameState.xp ?? 0;
  const totalInteractions = gameState.totalInteractions ?? gameState.tapCount ?? 0;
  if (level) level.textContent = "Lv " + globalLevel;
  if (xp) xp.textContent = totalXP + " XP";
  if (taps) taps.textContent = totalInteractions + " actions";
  if (bar) {
    const currentLevelStart = Math.max(0, globalLevel - 1) * 5;
    const progress = Math.max(0, Math.min(5, totalXP - currentLevelStart));
    bar.style.width = String(progress / 5 * 100) + "%";
  }
}
function renderSelection(gameState) {
  const preview = currentPreview();
  if (!preview) return;
  preview.querySelectorAll("[data-selected-component]").forEach((element2) => {
    element2.removeAttribute("data-selected-component");
    element2.style.outline = "";
    element2.style.outlineOffset = "";
  });
  const selected = gameState.selectedComponentId;
  if (!selected) return;
  const selector = '[data-component-id="' + escapeSelectorValue(selected) + '"]';
  const element = preview.matches(selector) ? preview : preview.querySelector(selector);
  if (!element) return;
  element.dataset.selectedComponent = "true";
  element.style.outline = "2px solid rgba(52, 211, 153, 0.86)";
  element.style.outlineOffset = "3px";
}
function canDragComponent(hit) {
  return hit.componentType === "shape" || hit.componentType === "textarea";
}
function componentElement(preview, componentId) {
  const selector = '[data-component-id="' + escapeSelectorValue(componentId) + '"]';
  return preview.matches(selector) ? preview : preview.querySelector(selector);
}
function readElementPercent(element, preview, property) {
  const inlineValue = property === "left" ? element.style.left : element.style.top;
  const parsed = parsePercent(inlineValue);
  if (parsed !== null) return parsed;
  const elementRect = element.getBoundingClientRect();
  const previewRect = preview.getBoundingClientRect();
  if (property === "left") {
    return clampPercent((elementRect.left - previewRect.left) / Math.max(1, previewRect.width) * 100);
  }
  return clampPercent((elementRect.top - previewRect.top) / Math.max(1, previewRect.height) * 100);
}
function parsePercent(value) {
  const trimmed = String(value || "").trim();
  if (!trimmed.endsWith("%")) return null;
  const number = Number(trimmed.slice(0, -1));
  return Number.isFinite(number) ? clampPercent(number) : null;
}
function clampPercent(value) {
  if (!Number.isFinite(value)) return 0;
  return Math.max(0, Math.min(100, value));
}
function formatPercent(value) {
  return value.toFixed(2).replace(/\.?0+$/, "") + "%";
}
function hasInvalidAction(events) {
  return (events || []).some((event) => event.type === "invalidAction");
}
function currentPreview() {
  return byID("draft-card-preview");
}
function overlayRoot() {
  return byID("stage-overlay-root");
}
function notificationRoot() {
  return byID("stage-notification-section");
}
function escapeSelectorValue(value) {
  return value.replace(/\\/g, "\\\\").replace(/"/g, '\\"');
}

// web/src/app.ts
document.addEventListener("DOMContentLoaded", () => {
  initDesigner();
  initStage({ resetDraft });
});
//# sourceMappingURL=app.js.map
