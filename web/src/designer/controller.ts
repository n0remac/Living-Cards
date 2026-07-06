import { addDraftComponent, applyDraftConfig, applyLibraryDesign, fetchDesignLibrary, fetchRenderedDraftCard, resetDraftCard, saveAppliedDesign } from "../api";
import { byID } from "../dom";
import { replacePreviewHTML } from "./document";
import { formatIssues, generateComponentConfig, isConfigGenerationError, parseGeneratedConfigEnvelope } from "./configs";
import type { DesignLibraryItem, ConfigIssue, GeneratedConfig } from "../types";

export function initDesigner(): void {
  const form = byID<HTMLFormElement>("card-designer-form");
  if (form) {
    form.addEventListener("submit", (event) => {
      void generateConfig(event);
    });
  }
  const apply = byID<HTMLButtonElement>("apply-config-btn");
  if (apply) {
    apply.addEventListener("click", () => {
      void applyConfig();
    });
  }
  const editor = byID<HTMLTextAreaElement>("config-preview");
  if (editor) {
    editor.addEventListener("input", () => {
      const hasCandidate = hasEditableConfigCandidate();
      const description = byID<HTMLParagraphElement>("config-description");
      if (description && hasCandidate) {
        description.textContent = "Edited design. Apply to validate it against the selected config kind.";
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
  const configKind = byID<HTMLSelectElement>("config-target");
  if (configKind) {
    configKind.addEventListener("change", () => {
      void loadDesignLibrary();
    });
  }
  const reset = byID<HTMLButtonElement>("designer-reset-btn");
  if (reset) {
    reset.addEventListener("click", () => {
      void resetDraft();
    });
  }
  bindAddComponentControls();
  renderProposedConfig(null);
  setSaveEnabled(false);
  void loadDesigner();
}

export async function resetDraft(): Promise<void> {
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

async function loadDesigner(): Promise<void> {
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

function renderProposedConfig(config: GeneratedConfig | null): void {
  const preview = byID<HTMLTextAreaElement>("config-preview");
  const description = byID<HTMLParagraphElement>("config-description");
  const apply = byID<HTMLButtonElement>("apply-config-btn");
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

function renderFailedConfig(rawResponse: string, message: string, issues: ConfigIssue[] = []): void {
  const preview = byID<HTMLTextAreaElement>("config-preview");
  const description = byID<HTMLParagraphElement>("config-description");
  const apply = byID<HTMLButtonElement>("apply-config-btn");
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
    apply.disabled = !hasEditableConfigCandidate();
  }
  setSaveEnabled(false);
  setDesignerStatus(message, true);
}

async function loadDesignLibrary(): Promise<void> {
  try {
    renderDesignLibraryItems(await fetchDesignLibrary(readConfigKind()));
  } catch (error) {
    setDesignerStatus(error instanceof Error ? error.message : "Failed to load design library.", true);
  }
}

function renderDesignLibraryItems(items: DesignLibraryItem[]): void {
  const list = byID<HTMLDivElement>("design-library-list");
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

async function generateConfig(event: SubmitEvent): Promise<void> {
  event.preventDefault();
  const configKind = readConfigKind();
  const isUpdate = (event.submitter as HTMLElement | null)?.id === "update-config-btn";
  const input = byID<HTMLTextAreaElement>("config-instruction");
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

async function applyConfig(): Promise<void> {
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
      const editor = byID<HTMLTextAreaElement>("config-preview");
      renderFailedConfig(error.rawResponse || String(editor?.value || ""), error.message, error.issues);
      return;
    }
    setDesignerStatus(error instanceof Error ? error.message : "Config could not be applied.", true);
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

function readConfigKind(): string {
  const select = byID<HTMLSelectElement>("config-target");
  switch (select?.value) {
    case "border":
    case "textarea":
    case "image":
      return select.value;
    default:
      return "background";
  }
}

function bindAddComponentControls(): void {
  byID<HTMLButtonElement>("add-textarea-component-btn")?.addEventListener("click", () => {
    void addComponent("textarea");
  });
  byID<HTMLButtonElement>("add-shape-component-btn")?.addEventListener("click", () => {
    void addComponent("shape");
  });
  byID<HTMLInputElement>("add-image-component-input")?.addEventListener("change", (event) => {
    const input = event.currentTarget;
    const file = input.files?.[0];
    input.value = "";
    if (!file) return;
    void addImageComponent(file);
  });
}

async function addComponent(componentKind: "textarea" | "shape"): Promise<void> {
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

async function addImageComponent(file: File): Promise<void> {
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
      border_radius_px: 14,
    });
    renderDesignLibraryItems(response.library);
    setDesignerStatus("Image component added to the draft card.", false);
    document.dispatchEvent(new CustomEvent("living-card:interactive-refresh"));
  } catch (error) {
    setDesignerStatus(error instanceof Error ? error.message : "Image component could not be added.", true);
  }
}

function fileToDataURL(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.addEventListener("load", () => resolve(String(reader.result || "")));
    reader.addEventListener("error", () => reject(new Error("Could not read image file.")));
    reader.readAsDataURL(file);
  });
}

function setBusy(isBusy: boolean, isUpdate = false): void {
  const generate = byID<HTMLButtonElement>("generate-config-btn");
  const update = byID<HTMLButtonElement>("update-config-btn");
  const apply = byID<HTMLButtonElement>("apply-config-btn");
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

function readConfigFromEditor(): GeneratedConfig {
  const editor = byID<HTMLTextAreaElement>("config-preview");
  return parseGeneratedConfigEnvelope(String(editor?.value || ""));
}

function hasEditableConfigCandidate(): boolean {
  const editor = byID<HTMLTextAreaElement>("config-preview");
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
