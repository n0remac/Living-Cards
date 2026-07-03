import { applyDraftFragment, applyLibraryDesign, fetchDesignLibrary, fetchRenderedDraftCard, resetDraftCard, saveAppliedDesign } from "../api";
import { byID } from "../dom";
import { replacePreviewHTML } from "./document";
import { formatIssues, generateTargetFragment, isFragmentGenerationError, parseGeneratedFragment } from "./fragments";
import type { DesignLibraryItem, FragmentIssue, GeneratedStyleFragment } from "../types";

export function initDesigner(): void {
  const form = byID<HTMLFormElement>("card-designer-form");
  if (form) {
    form.addEventListener("submit", (event) => {
      void generateFragment(event);
    });
  }
  const apply = byID<HTMLButtonElement>("apply-fragment-btn");
  if (apply) {
    apply.addEventListener("click", () => {
      void applyFragment();
    });
  }
  const editor = byID<HTMLTextAreaElement>("fragment-preview");
  if (editor) {
    editor.addEventListener("input", () => {
      const hasCandidate = hasEditableFragmentCandidate();
      const description = byID<HTMLParagraphElement>("fragment-description");
      if (description && hasCandidate) {
        description.textContent = "Edited fragment. Apply to validate it against the selected target.";
      }
      if (apply) {
        apply.disabled = !hasCandidate;
      }
      setSaveEnabled(false);
    });
  }
  const save = byID<HTMLButtonElement>("save-design-btn");
  if (save) {
    save.addEventListener("click", () => {
      void saveAppliedDesignToLibrary();
    });
  }
  const target = byID<HTMLSelectElement>("fragment-target");
  if (target) {
    target.addEventListener("change", () => {
      void loadDesignLibrary();
    });
  }
  const reset = byID<HTMLButtonElement>("designer-reset-btn");
  if (reset) {
    reset.addEventListener("click", () => {
      void resetDraft();
    });
  }
  renderProposedFragment(null);
  setSaveEnabled(false);
  void loadDesigner();
}

export async function resetDraft(): Promise<void> {
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

async function loadDesigner(): Promise<void> {
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

function renderProposedFragment(fragment: GeneratedStyleFragment | null): void {
  const preview = byID<HTMLTextAreaElement>("fragment-preview");
  const description = byID<HTMLParagraphElement>("fragment-description");
  const apply = byID<HTMLButtonElement>("apply-fragment-btn");
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

function renderFailedFragment(rawResponse: string, message: string, issues: FragmentIssue[] = []): void {
  const preview = byID<HTMLTextAreaElement>("fragment-preview");
  const description = byID<HTMLParagraphElement>("fragment-description");
  const apply = byID<HTMLButtonElement>("apply-fragment-btn");
  if (preview) {
    preview.value = rawResponse || "{}";
  }
  if (description) {
    const issueSummary = formatIssues(issues);
    description.textContent = issueSummary
      ? "Generation failed: " + issueSummary + " Edit the response below, then apply it to validate."
      : "Generation failed. Edit the response below, then apply it to validate.";
  }
  if (apply) {
    apply.disabled = !hasEditableFragmentCandidate();
  }
  setSaveEnabled(false);
  setDesignerStatus(message, true);
}

async function loadDesignLibrary(): Promise<void> {
  try {
    renderDesignLibraryItems(await fetchDesignLibrary(readTarget()));
  } catch (error) {
    setDesignerStatus(error instanceof Error ? error.message : "Failed to load design library.", true);
  }
}

function renderDesignLibraryItems(items: DesignLibraryItem[]): void {
  const list = byID<HTMLDivElement>("design-library-list");
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
    button.innerHTML =
      '<div class="flex items-center justify-between gap-2">' +
        '<span class="text-sm font-semibold text-[var(--app-fg)]">' + escapeHTML(item.name) + "</span>" +
        '<span class="text-[0.68rem] font-semibold uppercase text-[var(--app-fg-soft)]">' + (item.saved ? "Saved" : "Preset") + "</span>" +
      "</div>" +
      '<p class="mt-1 text-xs text-[var(--app-fg-muted)]">' + escapeHTML(item.description) + "</p>";
    button.addEventListener("click", () => {
      void applyLibraryItem(item.id);
    });
    list.appendChild(button);
  });
}

function setDesignerStatus(message: string, isError: boolean): void {
  const status = byID<HTMLDivElement>("designer-status");
  if (!status) return;
  status.textContent = message;
  status.className = isError ? "mt-4 text-sm text-red-300" : "mt-4 text-sm text-[var(--app-fg-soft)]";
}

async function generateFragment(event: SubmitEvent): Promise<void> {
  event.preventDefault();
  const target = readTarget();
  const isUpdate = (event.submitter as HTMLElement | null)?.id === "update-fragment-btn";
  const input = byID<HTMLTextAreaElement>("fragment-instruction");
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

async function applyFragment(): Promise<void> {
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
      const editor = byID<HTMLTextAreaElement>("fragment-preview");
      renderFailedFragment(error.rawResponse || String(editor?.value || ""), error.message, error.issues);
      return;
    }
    setDesignerStatus(error instanceof Error ? error.message : "Fragment could not be applied.", true);
  } finally {
    setBusy(false);
  }
}

async function applyLibraryItem(itemID: string): Promise<void> {
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

async function saveAppliedDesignToLibrary(): Promise<void> {
  try {
    const response = await saveAppliedDesign();
    renderDesignLibraryItems(response.library);
    setSaveEnabled(false);
    setDesignerStatus(response.item ? "Design saved to the library." : "Design is already in the library.", false);
  } catch (error) {
    setDesignerStatus(error instanceof Error ? error.message : "Apply a generated design before saving.", true);
  }
}

function setSaveEnabled(enabled: boolean): void {
  const save = byID<HTMLButtonElement>("save-design-btn");
  if (!save) return;
  save.disabled = !enabled;
}

function readTarget(): string {
  const select = byID<HTMLSelectElement>("fragment-target");
  switch (select?.value) {
    case "border":
    case "textarea":
      return select.value;
    default:
      return "background";
  }
}

function setBusy(isBusy: boolean, isUpdate = false): void {
  const generate = byID<HTMLButtonElement>("generate-fragment-btn");
  const update = byID<HTMLButtonElement>("update-fragment-btn");
  const apply = byID<HTMLButtonElement>("apply-fragment-btn");
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

function readFragmentFromEditor(): GeneratedStyleFragment {
  const editor = byID<HTMLTextAreaElement>("fragment-preview");
  return parseGeneratedFragment(String(editor?.value || ""));
}

function hasEditableFragmentCandidate(): boolean {
  const editor = byID<HTMLTextAreaElement>("fragment-preview");
  const raw = String(editor?.value || "").trim();
  return Boolean(raw && raw !== "{}");
}

function escapeHTML(value: unknown): string {
  return String(value || "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}
