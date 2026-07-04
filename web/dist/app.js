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
async function tapCardZone(target, zone, x, y) {
  const response = await fetch("/api/draft-card/tap", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ target, zone, x, y })
  });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to apply card tap."));
  }
  return await response.json();
}
async function applyColorControl(target, control) {
  const response = await fetch("/api/draft-card/control-change", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ target, ...control })
  });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to apply color."));
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

// web/src/stage/colorControls.ts
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
function openColorControls(options) {
  const root = options.root;
  if (!root) return;
  closeColorControls(root);
  const panel = document.createElement("div");
  panel.dataset.stageControlOverlay = "color";
  panel.className = "stage-control-panel pointer-events-auto fixed rounded-md border border-[var(--app-border-strong)] bg-[var(--app-surface-muted)] p-2 shadow-2xl backdrop-blur";
  panel.style.left = clampPX(options.anchorX, 12, window.innerWidth - 260) + "px";
  panel.style.top = clampPX(options.anchorY, 88, window.innerHeight - 270) + "px";
  const header = document.createElement("div");
  header.className = "flex items-center justify-between gap-2";
  const palette = document.createElement("button");
  palette.type = "button";
  palette.className = "flex h-9 items-center gap-2 rounded-md border border-[var(--app-border)] bg-[var(--app-panel)] px-2 text-sm font-semibold text-[var(--app-fg)]";
  palette.title = labelForTarget(options.target) + " color";
  palette.innerHTML = '<span class="block h-5 w-5 rounded-sm border border-white/35" style="background:' + escapeAttribute(hexOrFallback(options.currentColor)) + '"></span><span>Palette</span>';
  const close = document.createElement("button");
  close.type = "button";
  close.className = "h-9 rounded-md border border-[var(--app-border)] bg-[var(--app-panel)] px-2 text-sm text-[var(--app-fg-soft)]";
  close.textContent = "Close";
  close.addEventListener("click", () => panel.remove());
  const chooser = document.createElement("div");
  chooser.className = "mt-2 hidden grid w-56 gap-3";
  const primaryColor = hexOrFallback(options.currentColor);
  let secondaryColor = defaultSecondary(primaryColor);
  let angle = 135;
  let gradientEnabled = false;
  const swatchGrid = document.createElement("div");
  swatchGrid.className = "grid grid-cols-4 gap-1.5";
  swatches.forEach((color) => {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "h-8 rounded-md border border-white/25 shadow-sm";
    button.title = color;
    button.style.background = color;
    button.addEventListener("click", () => {
      if (gradientEnabled) {
        secondaryColor = color;
        secondaryInput.value = color;
        preview.style.background = gradientCSS(input.value, secondaryColor, angle);
        return;
      }
      options.onColor({ color });
    });
    swatchGrid.appendChild(button);
  });
  const input = document.createElement("input");
  input.type = "color";
  input.className = "h-9 w-full cursor-pointer rounded-md border border-[var(--app-border)] bg-[var(--app-panel)]";
  input.value = primaryColor;
  input.addEventListener("change", () => {
    preview.style.background = gradientEnabled ? gradientCSS(input.value, secondaryColor, angle) : input.value;
    if (!gradientEnabled) {
      options.onColor({ color: input.value });
    }
  });
  const gradientRow = document.createElement("label");
  gradientRow.className = "flex items-center justify-between gap-3 rounded-md border border-[var(--app-border)] bg-[var(--app-panel)] px-2 py-2 text-sm text-[var(--app-fg)]";
  gradientRow.innerHTML = "<span>Gradient</span>";
  const gradientToggle = document.createElement("input");
  gradientToggle.type = "checkbox";
  gradientToggle.className = "h-4 w-4 accent-emerald-300";
  const gradientControls = document.createElement("div");
  gradientControls.className = "hidden grid gap-2";
  const secondaryInput = document.createElement("input");
  secondaryInput.type = "color";
  secondaryInput.className = "h-9 w-full cursor-pointer rounded-md border border-[var(--app-border)] bg-[var(--app-panel)]";
  secondaryInput.value = secondaryColor;
  secondaryInput.addEventListener("input", () => {
    secondaryColor = secondaryInput.value;
    preview.style.background = gradientCSS(input.value, secondaryColor, angle);
  });
  const angleInput = document.createElement("input");
  angleInput.type = "range";
  angleInput.min = "0";
  angleInput.max = "315";
  angleInput.step = "45";
  angleInput.value = String(angle);
  angleInput.className = "w-full accent-emerald-300";
  angleInput.addEventListener("input", () => {
    angle = Number(angleInput.value) || 135;
    angleValue.textContent = angle + "deg";
    preview.style.background = gradientCSS(input.value, secondaryColor, angle);
  });
  const angleValue = document.createElement("span");
  angleValue.className = "text-xs font-semibold text-[var(--app-fg-soft)]";
  angleValue.textContent = angle + "deg";
  const preview = document.createElement("div");
  preview.className = "h-8 rounded-md border border-white/20";
  preview.style.background = primaryColor;
  const applyGradient = document.createElement("button");
  applyGradient.type = "button";
  applyGradient.className = "h-9 rounded-md border border-emerald-300/30 bg-emerald-300 px-3 text-sm font-semibold text-zinc-950";
  applyGradient.textContent = "Apply Gradient";
  applyGradient.addEventListener("click", () => {
    options.onColor({
      color: input.value,
      secondaryColor,
      gradient: true,
      angle
    });
  });
  gradientToggle.addEventListener("change", () => {
    gradientEnabled = gradientToggle.checked;
    gradientControls.classList.toggle("hidden", !gradientEnabled);
    preview.style.background = gradientEnabled ? gradientCSS(input.value, secondaryColor, angle) : input.value;
  });
  palette.addEventListener("click", () => {
    chooser.classList.toggle("hidden");
  });
  gradientRow.appendChild(gradientToggle);
  gradientControls.append(
    labeledControl("Second Color", secondaryInput),
    labeledControl("Angle", angleInput, angleValue),
    preview,
    applyGradient
  );
  header.append(palette, close);
  chooser.append(swatchGrid, labeledControl("Color", input), gradientRow, gradientControls);
  panel.append(header, chooser);
  root.appendChild(panel);
}
function closeColorControls(root) {
  root?.querySelectorAll("[data-stage-control-overlay]").forEach((element) => element.remove());
}
function clampPX(value, min, max) {
  if (!Number.isFinite(value)) return min;
  return Math.max(min, Math.min(max, value));
}
function hexOrFallback(value) {
  const trimmed = String(value || "").trim();
  return /^#[0-9a-fA-F]{6}$/.test(trimmed) ? trimmed : "#22c55e";
}
function defaultSecondary(primary) {
  return primary.toLowerCase() === "#38bdf8" ? "#22c55e" : "#38bdf8";
}
function gradientCSS(primary, secondary, angle) {
  return "linear-gradient(" + angle + "deg, " + primary + " 0%, " + secondary + " 100%)";
}
function labeledControl(label, control, aside) {
  const wrapper = document.createElement("label");
  wrapper.className = "grid gap-1 text-xs font-semibold uppercase text-[var(--app-fg-soft)]";
  const row = document.createElement("span");
  row.className = "flex items-center justify-between gap-2";
  const text = document.createElement("span");
  text.textContent = label;
  row.appendChild(text);
  if (aside) row.appendChild(aside);
  wrapper.append(row, control);
  return wrapper;
}
function labelForTarget(target) {
  switch (target) {
    case "background":
      return "Background";
    case "border":
      return "Border";
    default:
      return "Card";
  }
}
function escapeAttribute(value) {
  return value.replace(/&/g, "&amp;").replace(/"/g, "&quot;");
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
    return { target: "border", zone: "border", x, y, clientX: event.clientX, clientY: event.clientY };
  }
  const target = event.target instanceof Element ? event.target : null;
  if (target?.closest('[data-component-type="textarea"]')) {
    return { target: "textarea", zone: "textarea", x, y, clientX: event.clientX, clientY: event.clientY };
  }
  return { target: "background", zone: "background", x, y, clientX: event.clientX, clientY: event.clientY };
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
  stopCardTapEvents(section);
  const history = notificationHistoryPanel();
  if (history) {
    stopCardTapEvents(history);
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
  events.forEach((event) => showEvent(root, event));
}
function showMessage(root, message, tone = "info") {
  if (!root) return;
  enqueueNotification({ message, tone });
}
function showEvent(root, event) {
  switch (event.type) {
    case "fragmentApplied":
      return;
    case "xpGained":
      return;
    case "levelUp":
      return;
    case "targetUnlocked":
      showMessage(root, labelForTarget2(event.target) + " unlocked");
      return;
    case "modeUnlocked":
      showMessage(root, labelForTarget2(event.target) + " " + event.mode + " unlocked");
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
function stopCardTapEvents(element) {
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
function labelForTarget2(target) {
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

// web/src/stage/StageController.ts
var tapBusy = false;
var colorBusy = false;
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
  closeColorControls(overlayRoot());
  cancelPress();
  workspace.setPointerCapture?.(event.pointerId);
  pressState = {
    pointerID: event.pointerId,
    hit,
    startX: event.clientX,
    startY: event.clientY,
    longPressed: false,
    timer: window.setTimeout(() => {
      if (!pressState || pressState.pointerID !== event.pointerId) return;
      pressState.longPressed = true;
      openLongPressControls(hit);
    }, longPressMS)
  };
}
function maybeCancelPressMove(event) {
  if (!pressState || pressState.pointerID !== event.pointerId || pressState.longPressed) return;
  const dx = event.clientX - pressState.startX;
  const dy = event.clientY - pressState.startY;
  if (Math.hypot(dx, dy) > moveCancelPX) {
    cancelPress();
  }
}
async function finishPress(event, workspace) {
  if (!pressState || pressState.pointerID !== event.pointerId) return;
  const state = pressState;
  cancelPress();
  workspace.releasePointerCapture?.(event.pointerId);
  if (state.longPressed) return;
  await handleCardTap(state.hit);
}
function cancelPress() {
  if (pressState) {
    window.clearTimeout(pressState.timer);
  }
  pressState = null;
}
async function handleCardTap(hit) {
  if (tapBusy || document.body.classList.contains("designer-open")) return;
  const preview = currentPreview();
  if (!preview) return;
  animateCardTap(preview, hit.x, hit.y);
  tapBusy = true;
  try {
    const response = await tapCardZone(hit.target, hit.zone, hit.x, hit.y);
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
function openLongPressControls(hit) {
  const preview = currentPreview();
  if (!targetSupportsColorControls(hit.target) || !colorControlsUnlocked(hit.target)) {
    if (preview) animateInvalidTap(preview);
    showMessage(notificationRoot(), lockedControlMessage(hit.target), "error");
    return;
  }
  openColorControls({
    root: overlayRoot(),
    target: hit.target,
    anchorX: hit.clientX,
    anchorY: hit.clientY,
    currentColor: currentColorForTarget(hit.target),
    onColor: (control) => {
      void applyColor(hit.target, control);
    }
  });
}
async function applyColor(target, control) {
  if (colorBusy) return;
  colorBusy = true;
  try {
    const response = await applyColorControl(target, control);
    applyTapResponse(response);
    renderEvents(notificationRoot(), response.events);
  } catch (error) {
    showMessage(notificationRoot(), error instanceof Error ? error.message : "Color change failed.", "error");
  } finally {
    colorBusy = false;
  }
}
function applyTapResponse(response) {
  latestDocument = response.document;
  latestGameState = response.gameState;
  replacePreviewHTML(response.preview_html);
  updateHUD(response.gameState);
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
  if (level) level.textContent = "Lv " + gameState.level;
  if (xp) xp.textContent = gameState.xp + " XP";
  if (taps) taps.textContent = gameState.tapCount + " taps";
  if (bar) {
    const currentLevelStart = Math.max(0, gameState.level - 1) * 5;
    const progress = Math.max(0, Math.min(5, gameState.xp - currentLevelStart));
    bar.style.width = String(progress / 5 * 100) + "%";
  }
}
function colorControlsUnlocked(target) {
  if (!latestGameState) return false;
  const modes = latestGameState.targetProgress[target]?.unlockedModes || [];
  return modes.includes("simpleControls");
}
function targetSupportsColorControls(target) {
  return target === "background" || target === "border";
}
function lockedControlMessage(target) {
  if (targetSupportsColorControls(target)) {
    return "Color controls unlock at level 5.";
  }
  return "Color controls are locked.";
}
function currentColorForTarget(target) {
  const node = latestDocument ? findNode(latestDocument.root, target) : null;
  const fragment = node?.fragment || {};
  if (target === "background") {
    return typeof fragment.background_color === "string" ? fragment.background_color : "#22c55e";
  }
  if (target === "border") {
    return typeof fragment.border_color === "string" ? fragment.border_color : "#22c55e";
  }
  return "#22c55e";
}
function findNode(node, target) {
  if (node.type === target) return node;
  for (const child of node.children || []) {
    const match = findNode(child, target);
    if (match) return match;
  }
  return null;
}
function hasInvalidAction(events) {
  return events.some((event) => event.type === "invalidAction");
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

// web/src/app.ts
document.addEventListener("DOMContentLoaded", () => {
  initDesigner();
  initStage({ resetDraft });
});
//# sourceMappingURL=app.js.map
