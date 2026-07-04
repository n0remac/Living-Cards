import { applyColorControl, ColorControlPayload, fetchInteractiveDraftCard, tapCardZone } from "../api";
import { byID } from "../dom";
import { replacePreviewHTML } from "../designer/document";
import type { CardDocument, CardEvent, ComponentTarget, GameState, InteractiveDraftCardResponse, TapCardResponse } from "../types";
import { animateCardTap, animateInvalidTap } from "./cardMotion";
import { closeColorControls, openColorControls } from "./colorControls";
import { CardHit, hitTestCard } from "./hitTesting";
import { initNotifications, renderEvents, showMessage } from "./overlays";

interface StageDeps {
  resetDraft: () => Promise<void>;
}

let tapBusy = false;
let colorBusy = false;
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

function openLongPressControls(hit: CardHit): void {
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
    },
  });
}

async function applyColor(target: ComponentTarget, control: ColorControlPayload): Promise<void> {
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

function applyTapResponse(response: TapCardResponse): void {
  latestDocument = response.document;
  latestGameState = response.gameState;
  replacePreviewHTML(response.preview_html);
  updateHUD(response.gameState);
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
  if (level) level.textContent = "Lv " + gameState.level;
  if (xp) xp.textContent = gameState.xp + " XP";
  if (taps) taps.textContent = gameState.tapCount + " taps";
  if (bar) {
    const currentLevelStart = Math.max(0, gameState.level - 1) * 5;
    const progress = Math.max(0, Math.min(5, gameState.xp - currentLevelStart));
    bar.style.width = String((progress / 5) * 100) + "%";
  }
}

function colorControlsUnlocked(target: ComponentTarget): boolean {
  if (!latestGameState) return false;
  const modes = latestGameState.targetProgress[target]?.unlockedModes || [];
  return modes.includes("simpleControls");
}

function targetSupportsColorControls(target: ComponentTarget): boolean {
  return target === "background" || target === "border";
}

function lockedControlMessage(target: ComponentTarget): string {
  if (targetSupportsColorControls(target)) {
    return "Color controls unlock at level 5.";
  }
  return "Color controls are locked.";
}

function currentColorForTarget(target: ComponentTarget): string {
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

function findNode(node: CardDocument["root"], target: ComponentTarget): CardDocument["root"] | null {
  if (node.type === target) return node;
  for (const child of node.children || []) {
    const match = findNode(child, target);
    if (match) return match;
  }
  return null;
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
