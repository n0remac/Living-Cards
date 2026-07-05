import { collectGameCard, cycleGameCard, fetchGameSession, playGameCard, resetGameSession } from "../api";
import { byID } from "../dom";
import type { GameSessionSnapshot, RenderedGameCard } from "../types";

let latestSession: GameSessionSnapshot | null = null;
let busy = false;

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

  const dropTarget = byID<HTMLElement>("game-world-card");
  dropTarget?.addEventListener("dragover", (event) => {
    event.preventDefault();
    event.dataTransfer!.dropEffect = "move";
  });
  dropTarget?.addEventListener("drop", (event) => {
    event.preventDefault();
    const sourceCardId = event.dataTransfer?.getData("text/plain") || "";
    const targetCardId = latestSession?.activeWorldCardId || "";
    if (sourceCardId && targetCardId) {
      void play(sourceCardId, targetCardId);
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

function renderSession(session: GameSessionSnapshot): void {
  latestSession = session;
  renderActiveCard(session.activeWorldCard);
  renderLibrary(session.library);
  renderAction(session.activeWorldCard);
  setStatus(session.message || "");

  const progress = byID<HTMLElement>("game-progress");
  if (progress) {
    progress.textContent = `${session.library.length} cards collected`;
  }
  const count = byID<HTMLElement>("game-library-count");
  if (count) {
    count.textContent = session.library.length ? `${session.library.length} card${session.library.length === 1 ? "" : "s"}` : "Empty";
  }
}

function renderActiveCard(card: RenderedGameCard): void {
  const root = byID<HTMLElement>("game-world-card");
  if (!root) return;
  root.innerHTML = card.preview_html;
  root.dataset.activeCardId = card.id;
  root.dataset.cardKind = card.kind;
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
    const button = document.createElement("button");
    button.type = "button";
    button.className = "game-library-card";
    button.draggable = true;
    button.dataset.cardId = card.id;
    button.addEventListener("dragstart", (event) => {
      event.dataTransfer?.setData("text/plain", card.id);
      event.dataTransfer!.effectAllowed = "move";
    });

    const preview = document.createElement("div");
    preview.innerHTML = card.preview_html;

    const label = document.createElement("div");
    label.className = "game-library-card-name";
    label.textContent = card.name;

    button.append(preview, label);
    root.appendChild(button);
  });
}

function setStatus(message: string, isError = false): void {
  const status = byID<HTMLElement>("game-status");
  if (!status) return;
  status.textContent = message;
  status.dataset.tone = isError ? "error" : "info";
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : "Something went wrong.";
}
