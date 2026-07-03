import { initAppHeader } from "../../internal/web/components/appheader/client";
import { initChatForm, refreshSelectedCardChatView, setChatStatus } from "../../internal/web/components/chatform/client";
import { fetchCard, fetchCards, fetchRecentMemories, sendChatMessage } from "./api";
import { byID, escapeHtml } from "./dom";
import { livingCardState } from "./state";
import type { Card, Memory, SearchResult, TranscriptItem } from "./types";

async function loadCards(): Promise<void> {
  setChatStatus("Loading cards...", false);
  try {
    livingCardState.cards = await fetchCards();
  } catch {
    setChatStatus("Failed to load cards.", true);
    return;
  }
  if (!livingCardState.selectedCardId && livingCardState.cards.length) {
    livingCardState.selectedCardId = livingCardState.cards[0].card_id;
  }
  renderCards();
  await refreshSelectedCard();
}

function renderCards(): void {
  const list = byID<HTMLDivElement>("card-list");
  if (!list) return;
  list.innerHTML = "";
  if (!livingCardState.cards.length) {
    list.innerHTML = '<div class="rounded-2xl border border-dashed border-[var(--app-border)] px-4 py-8 text-center text-sm text-[var(--app-fg-soft)]">No cards available.</div>';
    return;
  }
  livingCardState.cards.forEach((card) => {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "card-item rounded-2xl border border-[var(--app-border)] bg-[var(--app-surface-muted)] px-4 py-3 text-left text-[var(--app-fg)] shadow-sm transition hover:-translate-y-0.5 hover:border-cyan-400/60 hover:bg-cyan-500/10";
    if (card.card_id === livingCardState.selectedCardId) {
      button.classList.add("is-active");
    }
    button.innerHTML =
      '<div class="flex items-center justify-between gap-2">' +
        '<span class="font-semibold text-[var(--app-fg)]">' + escapeHtml(card.name) + "</span>" +
        '<span class="rounded-full border border-cyan-400/30 bg-cyan-400/12 px-2 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.18em] text-cyan-200">' + escapeHtml(card.card_id) + "</span>" +
      "</div>" +
      '<p class="mt-2 text-xs text-[var(--app-fg-muted)]">' + escapeHtml((card.domain || []).join(", ") || card.archetype || "") + "</p>";
    button.addEventListener("click", async () => {
      livingCardState.selectedCardId = card.card_id;
      renderCards();
      await refreshSelectedCard();
    });
    list.appendChild(button);
  });
}

async function refreshSelectedCard(): Promise<void> {
  const cardId = livingCardState.selectedCardId;
  if (!cardId) {
    return;
  }
  let card: Card;
  try {
    card = await fetchCard(cardId);
  } catch {
    setChatStatus("Failed to load card.", true);
    return;
  }
  refreshSelectedCardChatView(card);
  await refreshRecentMemories();
  setChatStatus("Ready.", false);
}

function renderRetrievedMemories(items: SearchResult[]): void {
  const el = byID<HTMLDivElement>("retrieved-memories");
  if (!el) return;
  if (!items.length) {
    el.innerHTML = "<p>No retrieved memories for this turn.</p>";
    return;
  }
  el.innerHTML = items.map((item) => {
    return '<div class="rounded-2xl border border-[var(--app-border)] bg-[var(--app-panel)] px-4 py-3">' +
      '<p class="text-xs font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]">Rank ' + escapeHtml(item.rank) + " &middot; Score " + escapeHtml(Number(item.score || 0).toFixed(3)) + "</p>" +
      '<p class="mt-2 text-sm text-[var(--app-fg)]">' + escapeHtml((item.memory || {}).summary || "") + "</p>" +
    "</div>";
  }).join("");
}

function renderStoredSummary(memory?: Memory): void {
  const el = byID<HTMLDivElement>("stored-summary");
  if (!el) return;
  if (!memory || !memory.summary) {
    el.textContent = "No interaction stored yet.";
    return;
  }
  el.innerHTML = '<p class="text-sm text-[var(--app-fg)]">' + escapeHtml(memory.summary) + '</p><p class="mt-2 text-xs text-[var(--app-fg-soft)]">' + escapeHtml(memory.timestamp || "") + "</p>";
}

async function refreshRecentMemories(): Promise<void> {
  const el = byID<HTMLDivElement>("recent-memories");
  if (!el) return;
  if (!livingCardState.selectedCardId) {
    el.textContent = "No card selected.";
    return;
  }
  let memories: Memory[];
  try {
    memories = await fetchRecentMemories(livingCardState.selectedCardId, livingCardState.userId);
  } catch {
    el.innerHTML = "<p>Failed to load memories.</p>";
    return;
  }
  if (!memories.length) {
    el.innerHTML = "<p>No stored memories for this card yet.</p>";
    return;
  }
  el.innerHTML = memories.map((item) => {
    return '<div class="rounded-2xl border border-[var(--app-border)] bg-[var(--app-panel)] px-4 py-3">' +
      '<p class="text-xs font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]">' + escapeHtml(item.timestamp || "") + "</p>" +
      '<p class="mt-2 text-sm text-[var(--app-fg)]">' + escapeHtml(item.summary || "") + "</p>" +
    "</div>";
  }).join("");
}

function getTranscript(cardId: string): TranscriptItem[] {
  return livingCardState.transcripts[cardId] || [];
}

function appendTranscript(cardId: string, item: TranscriptItem): void {
  if (!livingCardState.transcripts[cardId]) {
    livingCardState.transcripts[cardId] = [];
  }
  livingCardState.transcripts[cardId].push(item);
}

document.addEventListener("DOMContentLoaded", () => {
  initAppHeader({ loadCards });
  initChatForm({
    getSelectedCardId: () => livingCardState.selectedCardId,
    getUserId: () => livingCardState.userId,
    getTranscript,
    appendTranscript,
    sendChat: sendChatMessage,
    refreshRecentMemories,
    renderRetrievedMemories,
    renderStoredSummary,
  });
  void loadCards();
});
