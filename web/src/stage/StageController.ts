import { fetchInteractiveDraftCard, tapCardZone } from "../api";
import { byID } from "../dom";
import { replacePreviewHTML } from "../designer/document";
import type { CardEvent, GameState, InteractiveDraftCardResponse } from "../types";
import { animateCardTap, animateInvalidTap } from "./cardMotion";
import { hitTestCard } from "./hitTesting";
import { renderEvents, showMessage } from "./overlays";

interface StageDeps {
  resetDraft: () => Promise<void>;
}

let tapBusy = false;

export function initStage(deps: StageDeps): void {
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
      showMessage(overlayRoot(), error instanceof Error ? error.message : "Failed to load card.", "error");
    }
  }
}

function applyInteractiveResponse(response: InteractiveDraftCardResponse): void {
  replacePreviewHTML(response.preview_html);
  updateHUD(response.gameState);
}

function bindTapLayer(): void {
  const workspace = byID<HTMLElement>("card-workspace");
  if (!workspace) return;
  workspace.addEventListener("pointerdown", (event) => {
    void handleCardTap(event);
  });
}

async function handleCardTap(event: PointerEvent): Promise<void> {
  if (tapBusy || document.body.classList.contains("designer-open")) return;
  const preview = currentPreview();
  if (!preview) return;
  const hit = hitTestCard(event, preview);
  animateCardTap(preview, hit.x, hit.y);
  tapBusy = true;
  try {
    const response = await tapCardZone(hit.target, hit.zone, hit.x, hit.y);
    replacePreviewHTML(response.preview_html);
    updateHUD(response.gameState);
    const nextPreview = currentPreview();
    if (hasInvalidAction(response.events)) {
      if (nextPreview) animateInvalidTap(nextPreview);
    } else if (nextPreview) {
      animateCardTap(nextPreview, hit.x, hit.y);
    }
    renderEvents(overlayRoot(), response.events);
  } catch (error) {
    const nextPreview = currentPreview();
    if (nextPreview) animateInvalidTap(nextPreview);
    showMessage(overlayRoot(), error instanceof Error ? error.message : "Tap failed.", "error");
  } finally {
    tapBusy = false;
  }
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

function hasInvalidAction(events: CardEvent[]): boolean {
  return events.some((event) => event.type === "invalidAction");
}

function currentPreview(): HTMLElement | null {
  return byID<HTMLElement>("draft-card-preview");
}

function overlayRoot(): HTMLElement | null {
  return byID<HTMLElement>("stage-overlay-root");
}
