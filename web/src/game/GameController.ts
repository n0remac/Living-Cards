import {
  applyGameCardComponentControl,
  applyGameEditControl,
  applyGameLibraryComponentControl,
  cancelGameEdit,
  collectGameCard,
  cycleGameCard,
  fetchGameSession,
  installGameEditComponent,
  playGameCard,
  resetGameSession,
  saveGameEdit,
  selectGameCardComponent,
  selectGameEditComponent,
  startGameEdit,
} from "../api";
import { byID } from "../dom";
import { closeComponentOverlay, openComponentOverlay } from "../stage/componentControls";
import type { ComponentOverlay, ControlDescriptor, GameSessionSnapshot, RenderedGameCard } from "../types";

let latestSession: GameSessionSnapshot | null = null;
let busy = false;
let controlBusy = false;
let activePressState: ActivePressState | null = null;
let editPressState: ActivePressState | null = null;

const longPressDelayMS = 560;
const dragThresholdPX = 6;

interface RenderOptions {
  openOverlay?: boolean;
  openActiveOverlay?: boolean;
}

interface ActiveComponentHit {
  cardId: string;
  componentId: string;
  componentKind: string;
  element: HTMLElement;
  preview: HTMLElement;
}

interface ActivePressState {
  hit: ActiveComponentHit;
  pointerId: number;
  startClientX: number;
  startClientY: number;
  startX: number;
  startY: number;
  nextX: number;
  nextY: number;
  dragging: boolean;
  longPressFired: boolean;
  longPressTimer: number;
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

  const editCard = byID<HTMLElement>("game-edit-card");
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
  bindSliderInputEvents(byID<HTMLElement>("game-library-list"), "library");

  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      closeComponentOverlay(overlayRoot());
    }
  });
}

type SliderInputScope = "world" | "edit" | "library";

function bindSliderInputEvents(root: HTMLElement | null, scope: SliderInputScope): void {
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

function stopSliderInputEvent(event: Event): void {
  if (sliderInputFromEvent(event)) {
    event.stopPropagation();
  }
}

function sliderInputFromEvent(event: Event): HTMLInputElement | null {
  const target = event.target;
  if (!(target instanceof Element)) return null;
  const input = target.closest<HTMLInputElement>("input[data-slider-input]");
  return input && input.type === "range" ? input : null;
}

function isSliderInputTarget(event: Event): boolean {
  return Boolean(sliderInputFromEvent(event));
}

function updateSliderValueDisplay(input: HTMLInputElement): void {
  const component = input.closest<HTMLElement>("[data-component-id][data-component-kind='slider']");
  const value = component?.querySelector<HTMLElement>("[data-slider-value]");
  if (value) value.textContent = input.value;
}

async function commitSliderInputValue(scope: SliderInputScope, input: HTMLInputElement): Promise<void> {
  if (controlBusy) return;
  const component = input.closest<HTMLElement>("[data-component-id][data-component-kind='slider']");
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
      const cardId = input.closest<HTMLElement>(".game-library-card")?.dataset.cardId || input.closest<HTMLElement>("[data-card-id]")?.dataset.cardId || "";
      if (!cardId) return;
      renderSession(await applyGameLibraryComponentControl(cardId, componentId, "slider", "value", value));
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

function onActivePointerDown(event: PointerEvent): void {
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
    longPressTimer,
  };

  try {
    (event.currentTarget as HTMLElement).setPointerCapture(event.pointerId);
  } catch {
    // Pointer capture can fail if the target is already gone after a rerender.
  }
  event.preventDefault();
  event.stopPropagation();
}

function onActivePointerMove(event: PointerEvent): void {
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
  state.nextX = clampDragPercent(state.startX + (dx / rect.width) * 100);
  state.nextY = clampDragPercent(state.startY + (dy / rect.height) * 100);
  state.hit.element.style.left = `${state.nextX.toFixed(2)}%`;
  state.hit.element.style.top = `${state.nextY.toFixed(2)}%`;
  event.preventDefault();
  event.stopPropagation();
}

function onActivePointerUp(event: PointerEvent): void {
  const state = activePressState;
  if (!state || state.pointerId !== event.pointerId) return;

  const shouldCommit = state.dragging;
  const hit = state.hit;
  const x = Math.round(state.nextX);
  const y = Math.round(state.nextY);
  cleanupActivePressState(event.currentTarget as HTMLElement);
  if (shouldCommit) {
    void applyActivePosition(hit, x, y);
  }
  event.preventDefault();
  event.stopPropagation();
}

function onActivePointerCancel(event: PointerEvent): void {
  const state = activePressState;
  if (!state || state.pointerId !== event.pointerId) return;
  cleanupActivePressState(event.currentTarget as HTMLElement);
  if (latestSession) {
    renderSession(latestSession);
  }
}

function cleanupActivePressState(target?: HTMLElement): void {
  const state = activePressState;
  if (!state) return;
  window.clearTimeout(state.longPressTimer);
  delete state.hit.element.dataset.dragging;
  if (target) {
    try {
      target.releasePointerCapture(state.pointerId);
    } catch {
      // The browser may already have released capture after a rerender.
    }
  }
  activePressState = null;
}

function onEditPointerDown(event: PointerEvent): void {
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
    longPressTimer: 0,
  };

  try {
    (event.currentTarget as HTMLElement).setPointerCapture(event.pointerId);
  } catch {
    // Pointer capture can fail if a rerender happens during interaction.
  }
  event.preventDefault();
  event.stopPropagation();
}

function onEditPointerMove(event: PointerEvent): void {
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
  state.nextX = clampDragPercent(state.startX + (dx / rect.width) * 100);
  state.nextY = clampDragPercent(state.startY + (dy / rect.height) * 100);
  state.hit.element.style.left = `${state.nextX.toFixed(2)}%`;
  state.hit.element.style.top = `${state.nextY.toFixed(2)}%`;
  event.preventDefault();
  event.stopPropagation();
}

function onEditPointerUp(event: PointerEvent): void {
  const state = editPressState;
  if (!state || state.pointerId !== event.pointerId) return;

  const shouldCommit = state.dragging;
  const hit = state.hit;
  const x = Math.round(state.nextX);
  const y = Math.round(state.nextY);
  cleanupEditPressState(event.currentTarget as HTMLElement);
  if (shouldCommit) {
    void applyEditPosition(hit, x, y);
  } else {
    void selectEditComponent(hit);
  }
  event.preventDefault();
  event.stopPropagation();
}

function onEditPointerCancel(event: PointerEvent): void {
  const state = editPressState;
  if (!state || state.pointerId !== event.pointerId) return;
  cleanupEditPressState(event.currentTarget as HTMLElement);
  if (latestSession) {
    renderSession(latestSession);
  }
}

function cleanupEditPressState(target?: HTMLElement): void {
  const state = editPressState;
  if (!state) return;
  delete state.hit.element.dataset.dragging;
  if (target) {
    try {
      target.releasePointerCapture(state.pointerId);
    } catch {
      // The browser may already have released capture after a rerender.
    }
  }
  editPressState = null;
}

function activeComponentHit(event: MouseEvent | PointerEvent): ActiveComponentHit | null {
  const preview = activeCardPreview();
  const target = event.target;
  if (!preview || !(target instanceof Node) || !preview.contains(target)) return null;
  const cardId = preview.dataset.cardId || latestSession?.activeWorldCardId || "";
  if (!cardId) return null;

  const elementTarget = target instanceof Element ? target : null;
  const component = elementTarget?.closest<HTMLElement>("[data-component-id][data-component-kind]");
  if (component && preview.contains(component)) {
    const componentKind = component.dataset.componentKind || "";
    if (componentKind && componentKind !== "card") {
      return {
        cardId,
        componentId: component.dataset.componentId || "",
        componentKind,
        element: component,
        preview,
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
    preview,
  };
}

function activeCardPreview(): HTMLElement | null {
  const root = byID<HTMLElement>("game-world-card");
  const preview = root?.firstElementChild;
  return preview instanceof HTMLElement ? preview : null;
}

function editComponentHit(event: MouseEvent | PointerEvent): ActiveComponentHit | null {
  const preview = editCardPreview();
  const target = event.target;
  if (!preview || !(target instanceof Node) || !preview.contains(target)) return null;
  if (isSliderInputTarget(event)) return null;
  const cardId = latestSession?.editSession?.draftCard.id || preview.dataset.cardId || "";
  if (!cardId) return null;

  const elementTarget = target instanceof Element ? target : null;
  const component = elementTarget?.closest<HTMLElement>("[data-component-id][data-component-kind]");
  if (component && preview.contains(component)) {
    const componentKind = component.dataset.componentKind || "";
    if (componentKind === "slider") {
      return {
        cardId,
        componentId: component.dataset.componentId || "",
        componentKind,
        element: component,
        preview,
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
      preview,
    };
  }
  return null;
}

function editCardPreview(): HTMLElement | null {
  const root = byID<HTMLElement>("game-edit-card");
  const preview = root?.firstElementChild;
  return preview instanceof HTMLElement ? preview : null;
}

function componentPercentPosition(element: HTMLElement, preview: HTMLElement, componentKind: string): { x: number; y: number } {
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
    x: ((anchorX - previewRect.left) / previewRect.width) * 100,
    y: ((anchorY - previewRect.top) / previewRect.height) * 100,
  };
}

function parsePercent(value: string): number | null {
  const match = String(value || "").trim().match(/^(-?\d+(?:\.\d+)?)%$/);
  if (!match) return null;
  const parsed = Number(match[1]);
  return Number.isFinite(parsed) ? parsed : null;
}

function canDragActiveComponent(componentKind: string): boolean {
  return componentKind === "textarea" || componentKind === "shape" || componentKind === "image" || componentKind === "slider";
}

function canDragEditComponent(componentKind: string): boolean {
  return componentKind === "slider";
}

function componentTitle(componentKind: string): string {
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

function clampDragPercent(value: number): number {
  if (value < 1) return 1;
  if (value > 99) return 99;
  return value;
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
  closeComponentOverlay(overlayRoot());
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
  closeComponentOverlay(overlayRoot());
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
  closeComponentOverlay(overlayRoot());
  try {
    renderSession(await playGameCard(sourceCardId, targetCardId));
  } catch (error) {
    setStatus(errorMessage(error), true);
  } finally {
    busy = false;
  }
}

async function selectActiveComponent(hit: ActiveComponentHit): Promise<void> {
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

async function applyActivePosition(hit: ActiveComponentHit, x: number, y: number): Promise<void> {
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

async function selectEditComponent(hit: ActiveComponentHit): Promise<void> {
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

async function applyEditPosition(hit: ActiveComponentHit, x: number, y: number): Promise<void> {
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

function stopSliderInputEventsInPreview(preview: HTMLElement): void {
  for (const eventName of ["pointerdown", "click"]) {
    preview.addEventListener(eventName, (event) => {
      if (sliderInputFromEvent(event)) {
        event.stopPropagation();
      }
    });
  }
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
    preview.className = "game-card-thumbnail";
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

function openActiveEditingOverlay(overlay = latestSession?.activeEditingOverlay): void {
  if (!overlay) {
    closeComponentOverlay(overlayRoot());
    setStatus("Long press a card component to edit it.", true);
    return;
  }
  openComponentOverlay({
    root: overlayRoot(),
    overlay,
    onClose: () => undefined,
    onControl: (control, value) => {
      void applyActiveControl(overlay, control, value);
    },
  });
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

async function applyActiveControl(overlay: ComponentOverlay, control: ControlDescriptor, value: unknown): Promise<void> {
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
