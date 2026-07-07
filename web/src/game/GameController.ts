import {
  applyGameEditControl,
  cancelGameEdit,
  collectGameCard,
  cycleGameCard,
  fetchGameSession,
  installGameEditComponent,
  playGameCard,
  resetGameSession,
  saveGameEdit,
  startGameEdit,
} from "../api";
import { byID } from "../dom";
import { closeComponentOverlay, openComponentOverlay } from "../stage/componentControls";
import type { ComponentOverlay, ControlDescriptor, GameSessionSnapshot, RenderedGameCard } from "../types";

let latestSession: GameSessionSnapshot | null = null;
let busy = false;
let controlBusy = false;

interface RenderOptions {
  openOverlay?: boolean;
}

export function initGameStage(): void {
  bindControls();
  void loadSession();
}

function bindControls(): void {
  byID<HTMLButtonElement>("game-prev-card")?.addEventListener("click", () => {
    void cycle("previous");
  });
  byID<HTMLButtonElement>("game-next-card")?.addEventListener("click", () => {
    void cycle("next");
  });
  byID<HTMLButtonElement>("game-collect-card")?.addEventListener("click", () => {
    const active = latestSession?.activeWorldCard;
    if (!active) return;
    void collect(active.id);
  });
  byID<HTMLButtonElement>("reset-draft-btn")?.addEventListener("click", () => {
    void reset();
  });
  byID<HTMLButtonElement>("game-edit-save")?.addEventListener("click", () => {
    void saveEdit();
  });
  byID<HTMLButtonElement>("game-edit-cancel")?.addEventListener("click", () => {
    void cancelEdit();
  });

  const worldTarget = byID<HTMLElement>("game-world-card");
  worldTarget?.addEventListener("dragover", (event) => {
    event.preventDefault();
    event.dataTransfer!.dropEffect = "move";
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

  const editCanvas = byID<HTMLElement>("game-edit-canvas");
  editCanvas?.addEventListener("dragover", (event) => {
    event.preventDefault();
    editCanvas.dataset.dropActive = "true";
    event.dataTransfer!.dropEffect = "move";
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

  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      closeComponentOverlay(overlayRoot());
    }
  });
}

async function loadSession(): Promise<void> {
  setStatus("Loading scene...");
  try {
    renderSession(await fetchGameSession());
  } catch (error) {
    setStatus(errorMessage(error), true);
  }
}

async function reset(): Promise<void> {
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

async function cycle(direction: "next" | "previous"): Promise<void> {
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

async function collect(cardId: string): Promise<void> {
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

async function play(sourceCardId: string, targetCardId: string): Promise<void> {
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

async function startEdit(cardId: string): Promise<void> {
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

async function installEditComponent(componentCardId: string): Promise<void> {
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

async function saveEdit(): Promise<void> {
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

async function cancelEdit(): Promise<void> {
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

function renderSession(session: GameSessionSnapshot, options: RenderOptions = {}): void {
  latestSession = session;
  renderActiveCard(session.activeWorldCard);
  renderLibrary(session.library);
  renderEditMode(session);
  renderAction(session.activeWorldCard);
  setStatus(session.message || "");

  const progress = byID<HTMLElement>("game-progress");
  if (progress) {
    progress.textContent = `${session.library.length} cards in library`;
  }
  const count = byID<HTMLElement>("game-library-count");
  if (count) {
    count.textContent = session.library.length ? `${session.library.length} card${session.library.length === 1 ? "" : "s"}` : "Empty";
  }

  if (!session.editSession) {
    closeComponentOverlay(overlayRoot());
    return;
  }
  if (options.openOverlay) {
    openEditingOverlay(session.editSession.editingOverlay);
  }
}

function renderActiveCard(card: RenderedGameCard): void {
  const root = byID<HTMLElement>("game-world-card");
  if (!root) return;
  root.innerHTML = card.preview_html;
  root.dataset.activeCardId = card.id;
  root.dataset.cardKind = card.kind;
}

function renderEditMode(session: GameSessionSnapshot): void {
  const shell = byID<HTMLElement>("card-workspace");
  const mode = byID<HTMLElement>("game-edit-mode");
  const cardRoot = byID<HTMLElement>("game-edit-card");
  const title = byID<HTMLElement>("game-edit-title");
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

function renderAction(card: RenderedGameCard): void {
  const collect = byID<HTMLButtonElement>("game-collect-card");
  if (!collect) return;
  collect.hidden = !card.collectible || Boolean(card.collected);
  collect.disabled = !card.collectible || Boolean(card.collected);
}

function renderLibrary(cards: RenderedGameCard[]): void {
  const root = byID<HTMLElement>("game-library-list");
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
      event.dataTransfer!.effectAllowed = "move";
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
    preview.innerHTML = card.preview_html;

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

function libraryAction(card: RenderedGameCard): HTMLButtonElement | null {
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

function renderComponentTray(cards: RenderedGameCard[]): void {
  const root = byID<HTMLElement>("game-edit-component-tray");
  const count = byID<HTMLElement>("game-edit-component-count");
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
      event.dataTransfer!.effectAllowed = "move";
    });
    item.addEventListener("click", () => {
      if (!disabled) void installEditComponent(card.id);
    });

    const preview = document.createElement("div");
    preview.innerHTML = card.preview_html;

    const label = document.createElement("div");
    label.className = "game-library-card-name";
    label.textContent = card.name;

    item.append(preview, label);
    root.appendChild(item);
  });
}

async function handleLibraryClick(card: RenderedGameCard): Promise<void> {
  if (isEditableCard(card)) {
    await startEdit(card.id);
    return;
  }
  setStatus("Drag this card onto the active world card to use it.");
}

function openEditingOverlay(overlay = latestSession?.editSession?.editingOverlay): void {
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
    onClose: () => undefined,
    onControl: (control, value) => {
      void applyEditControl(overlay, control, value);
    },
  });
}

async function applyEditControl(overlay: ComponentOverlay, control: ControlDescriptor, value: unknown): Promise<void> {
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

function isEditableCard(card: RenderedGameCard): boolean {
  return Boolean(card.state?.editable);
}

function componentKindForCard(card: RenderedGameCard): string {
  const value = card.state?.componentKind;
  return typeof value === "string" ? value : "";
}

function pendingComponentIds(): Set<string> {
  return new Set(latestSession?.editSession?.pendingConsumedComponentIds || []);
}

function overlayRoot(): HTMLElement | null {
  return byID<HTMLElement>("stage-overlay-root");
}

function setStatus(message: string, isError = false): void {
  const status = byID<HTMLElement>("game-status");
  if (!status) return;
  status.textContent = message;
  status.dataset.tone = isError ? "error" : "info";
}

function setEditStatus(message: string, isError = false): void {
  const status = byID<HTMLElement>("game-edit-status");
  if (!status) return;
  status.textContent = message;
  status.dataset.tone = isError ? "error" : "info";
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : "Something went wrong.";
}
