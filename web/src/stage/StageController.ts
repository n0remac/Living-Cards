import { applyComponentControl, fetchInteractiveDraftCard, interactWithComponent, randomizeComponent } from "../api";
import { byID } from "../dom";
import { replacePreviewHTML } from "../designer/document";
import type { CardDocument, CardEvent, ComponentOverlay, GameState, InteractiveDraftCardResponse, TapCardResponse } from "../types";
import { animateCardTap, animateInvalidTap } from "./cardMotion";
import { closeComponentOverlay, openComponentOverlay } from "./componentControls";
import { CardHit, hitTestCard } from "./hitTesting";
import { initNotifications, renderEvents, showMessage } from "./overlays";

interface StageDeps {
  resetDraft: () => Promise<void>;
}

let tapBusy = false;
let controlBusy = false;
let latestDocument: CardDocument | null = null;
let latestGameState: GameState | null = null;
let pressState: {
  pointerID: number;
  hit: CardHit;
  startX: number;
  startY: number;
  timer: number;
  longPressed: boolean;
} | null = null;

const longPressMS = 540;
const moveCancelPX = 12;

export function initStage(deps: StageDeps): void {
  initNotifications();
  bindTapLayer();
  bindDesignerOverlay();
  bindReset(deps);
  document.addEventListener("living-card:interactive-refresh", () => {
    void loadInteractive(false);
  });
  void loadInteractive(true);
}

async function loadInteractive(showErrors: boolean): Promise<void> {
  try {
    const response = await fetchInteractiveDraftCard();
    applyInteractiveResponse(response);
  } catch (error) {
    if (showErrors) {
      showMessage(notificationRoot(), error instanceof Error ? error.message : "Failed to load card.", "error");
    }
  }
}

function applyInteractiveResponse(response: InteractiveDraftCardResponse): void {
  latestDocument = response.document;
  latestGameState = response.gameState;
  replacePreviewHTML(response.preview_html);
  updateHUD(response.gameState);
  renderSelection(response.gameState);
}

function bindTapLayer(): void {
  const workspace = byID<HTMLElement>("card-workspace");
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

function startPress(event: PointerEvent, workspace: HTMLElement): void {
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
    longPressed: false,
    timer: window.setTimeout(() => {
      if (!pressState || pressState.pointerID !== event.pointerId) return;
      pressState.longPressed = true;
      void openLongPressControls(hit);
    }, longPressMS),
  };
}

function maybeCancelPressMove(event: PointerEvent): void {
  if (!pressState || pressState.pointerID !== event.pointerId || pressState.longPressed) return;
  const dx = event.clientX - pressState.startX;
  const dy = event.clientY - pressState.startY;
  if (Math.hypot(dx, dy) > moveCancelPX) {
    cancelPress();
  }
}

async function finishPress(event: PointerEvent, workspace: HTMLElement): Promise<void> {
  if (!pressState || pressState.pointerID !== event.pointerId) return;
  const state = pressState;
  cancelPress();
  workspace.releasePointerCapture?.(event.pointerId);
  if (state.longPressed) return;
  await handleCardTap(state.hit);
}

function cancelPress(): void {
  if (pressState) {
    window.clearTimeout(pressState.timer);
  }
  pressState = null;
}

async function handleCardTap(hit: CardHit): Promise<void> {
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

async function openLongPressControls(hit: CardHit): Promise<void> {
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

function openOverlay(overlay: ComponentOverlay, anchorX: number, anchorY: number): void {
  openComponentOverlay({
    root: overlayRoot(),
    overlay,
    anchorX,
    anchorY,
    onControl: (control, value) => {
      void applyControl(overlay.componentId, control.trait, control.control, value);
    },
    onRandomize: () => {
      void applyRandomize(overlay.componentId);
    },
  });
}

async function applyControl(componentId: string, trait: string, control: string, value: unknown): Promise<void> {
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

async function applyRandomize(componentId: string): Promise<void> {
  if (controlBusy) return;
  controlBusy = true;
  try {
    const response = await randomizeComponent(componentId);
    applyTapResponse(response);
    renderEvents(notificationRoot(), response.events);
    if (response.overlay) {
      openOverlay(response.overlay, window.innerWidth - 320, 110);
    }
  } catch (error) {
    showMessage(notificationRoot(), error instanceof Error ? error.message : "Randomize failed.", "error");
  } finally {
    controlBusy = false;
  }
}

function applyTapResponse(response: TapCardResponse): void {
  latestDocument = response.document;
  latestGameState = response.gameState;
  replacePreviewHTML(response.preview_html);
  updateHUD(response.gameState);
  renderSelection(response.gameState);
}

function bindDesignerOverlay(): void {
  const open = byID<HTMLButtonElement>("designer-toggle-btn");
  const close = byID<HTMLButtonElement>("designer-close-btn");
  const overlay = byID<HTMLDivElement>("designer-overlay");
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

function bindReset(deps: StageDeps): void {
  const reset = byID<HTMLButtonElement>("reset-draft-btn");
  reset?.addEventListener("click", () => {
    void deps.resetDraft();
  });
}

function updateHUD(gameState: GameState): void {
  const level = byID<HTMLSpanElement>("card-level");
  const xp = byID<HTMLSpanElement>("card-xp");
  const taps = byID<HTMLSpanElement>("card-taps");
  const bar = byID<HTMLDivElement>("card-xp-bar");
  const globalLevel = gameState.globalLevel || gameState.level || 1;
  const totalXP = gameState.totalXp ?? gameState.xp ?? 0;
  const totalInteractions = gameState.totalInteractions ?? gameState.tapCount ?? 0;
  if (level) level.textContent = "Lv " + globalLevel;
  if (xp) xp.textContent = totalXP + " XP";
  if (taps) taps.textContent = totalInteractions + " actions";
  if (bar) {
    const currentLevelStart = Math.max(0, globalLevel - 1) * 5;
    const progress = Math.max(0, Math.min(5, totalXP - currentLevelStart));
    bar.style.width = String((progress / 5) * 100) + "%";
  }
}

function renderSelection(gameState: GameState): void {
  const preview = currentPreview();
  if (!preview) return;
  preview.querySelectorAll<HTMLElement>("[data-selected-component]").forEach((element) => {
    element.removeAttribute("data-selected-component");
    element.style.outline = "";
    element.style.outlineOffset = "";
  });
  const selected = gameState.selectedComponentId;
  if (!selected) return;
  const selector = '[data-component-id="' + escapeSelectorValue(selected) + '"]';
  const element = preview.matches(selector) ? preview : preview.querySelector<HTMLElement>(selector);
  if (!element) return;
  element.dataset.selectedComponent = "true";
  element.style.outline = "2px solid rgba(52, 211, 153, 0.86)";
  element.style.outlineOffset = "3px";
}

function hasInvalidAction(events: CardEvent[]): boolean {
  return events.some((event) => event.type === "invalidAction");
}

function currentPreview(): HTMLElement | null {
  return byID<HTMLElement>("draft-card-preview");
}

function overlayRoot(): HTMLElement | null {
  return byID<HTMLElement>("stage-overlay-root");
}

function notificationRoot(): HTMLElement | null {
  return byID<HTMLElement>("stage-notification-section");
}

function escapeSelectorValue(value: string): string {
  return value.replace(/\\/g, "\\\\").replace(/"/g, '\\"');
}
