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

// internal/web/components/appheader/client.ts
function initAppHeader(deps) {
  const reset = byID("reset-draft-btn");
  if (reset) {
    reset.addEventListener("click", () => {
      void deps.resetDraft();
    });
  }
}

// web/src/app.ts
document.addEventListener("DOMContentLoaded", () => {
  initAppHeader({ resetDraft });
  initDesigner();
});
//# sourceMappingURL=app.js.map
