import { collectGameCard, cycleGameCard, fetchGameSession, playGameCard, resetGameSession, saveControllerCard } from "../api";
import { byID } from "../dom";
import type { CardDocument, ComponentNode, GameSessionSnapshot, RenderedGameCard } from "../types";

let latestSession: GameSessionSnapshot | null = null;
let busy = false;

export function initGameStage(): void {
  bindControls();
  bindControllerBuilder();
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

function bindControllerBuilder(): void {
  byID<HTMLButtonElement>("controller-builder-close")?.addEventListener("click", closeControllerBuilder);
  byID<HTMLButtonElement>("controller-builder-cancel")?.addEventListener("click", closeControllerBuilder);
  byID<HTMLButtonElement>("controller-builder-save")?.addEventListener("click", () => {
    void saveController();
  });
  byID<HTMLDivElement>("controller-builder-overlay")?.addEventListener("click", (event) => {
    if (event.target === event.currentTarget) {
      closeControllerBuilder();
    }
  });
  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape" && document.body.classList.contains("controller-builder-open")) {
      closeControllerBuilder();
    }
  });

  const range = byID<HTMLInputElement>("controller-slider-input");
  const number = byID<HTMLInputElement>("controller-slider-number");
  range?.addEventListener("input", () => syncControllerInputs(Number(range.value)));
  number?.addEventListener("input", () => syncControllerInputs(Number(number.value)));
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
    const item = document.createElement("div");
    item.className = "game-library-card";
    item.draggable = true;
    item.dataset.cardId = card.id;
    item.addEventListener("dragstart", (event) => {
      event.dataTransfer?.setData("text/plain", card.id);
      event.dataTransfer!.effectAllowed = "move";
    });

    const preview = document.createElement("div");
    preview.innerHTML = card.preview_html;

    const label = document.createElement("div");
    label.className = "game-library-card-name";
    label.textContent = card.name;

    item.append(preview, label);
    if (card.id === "blank-controller") {
      const build = document.createElement("button");
      build.type = "button";
      build.className = "game-library-build";
      build.textContent = "Build";
      build.disabled = !hasLibraryCard("slider-component");
      build.title = build.disabled ? "Collect Slider Component first." : "Build Regulator Controller";
      build.addEventListener("pointerdown", (event) => {
        event.stopPropagation();
      });
      build.addEventListener("click", (event) => {
        event.preventDefault();
        event.stopPropagation();
        openControllerBuilder();
      });
      item.appendChild(build);
    }
    root.appendChild(item);
  });
}

function openControllerBuilder(): void {
  if (!hasLibraryCard("blank-controller")) {
    setStatus("Collect the Blank Controller first.", true);
    return;
  }
  if (!hasLibraryCard("slider-component")) {
    setStatus("Collect the Slider Component first.", true);
    return;
  }
  syncControllerInputs(existingControllerValue() ?? 50);
  document.body.classList.add("controller-builder-open");
}

function closeControllerBuilder(): void {
  document.body.classList.remove("controller-builder-open");
}

async function saveController(): Promise<void> {
  if (busy) return;
  busy = true;
  setStatus("Saving controller...");
  try {
    const value = readControllerValue();
    const snapshot = await saveControllerCard("blank-controller", createControllerDocument(value));
    closeControllerBuilder();
    renderSession(snapshot);
  } catch (error) {
    setStatus(errorMessage(error), true);
  } finally {
    busy = false;
  }
}

function hasLibraryCard(cardId: string): boolean {
  return Boolean(latestSession?.library.some((card) => card.id === cardId));
}

function existingControllerValue(): number | null {
  const controller = latestSession?.library.find((card) => card.id === "generator-regulator-controller");
  return controller ? sliderValueFromNode(controller.document.root) : null;
}

function sliderValueFromNode(node: ComponentNode): number | null {
  if (node.componentKind === "slider" && node.config && typeof node.config.value === "number") {
    return clampControllerValue(node.config.value);
  }
  for (const child of node.children || []) {
    const value = sliderValueFromNode(child);
    if (value !== null) return value;
  }
  return null;
}

function syncControllerInputs(value: number): void {
  const normalized = clampControllerValue(value);
  const range = byID<HTMLInputElement>("controller-slider-input");
  const number = byID<HTMLInputElement>("controller-slider-number");
  if (range) range.value = String(normalized);
  if (number) number.value = String(normalized);
}

function readControllerValue(): number {
  const number = byID<HTMLInputElement>("controller-slider-number");
  return clampControllerValue(Number(number?.value ?? 50));
}

function clampControllerValue(value: number): number {
  if (!Number.isFinite(value)) return 50;
  return Math.max(0, Math.min(100, Math.round(value)));
}

function createControllerDocument(value: number): CardDocument {
  return {
    card_id: "generator-regulator-controller",
    name: "Regulator Controller",
    root: {
      id: "generator-regulator-controller-root",
      componentKind: "card",
      config: {
        padding_px: 18,
        shadow: "0 24px 60px rgba(8,47,73,0.34)",
      },
      children: [{
        id: "regulator-output-slider",
        componentKind: "slider",
        config: {
          label: "Output",
          min: 0,
          max: 100,
          step: 1,
          value: clampControllerValue(value),
        },
      }],
    },
  };
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
