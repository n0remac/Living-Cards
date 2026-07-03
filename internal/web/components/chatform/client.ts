import { byID, escapeHtml } from "../../../../web/src/dom";
import type { Card, ChatResult, Memory, SearchResult, TranscriptItem } from "./types";

interface ChatFormDeps {
  getSelectedCardId: () => string;
  getUserId: () => string;
  getTranscript: (cardId: string) => TranscriptItem[];
  appendTranscript: (cardId: string, item: TranscriptItem) => void;
  sendChat: (cardId: string, userId: string, message: string) => Promise<ChatResult>;
  refreshRecentMemories: () => Promise<void>;
  renderRetrievedMemories: (items: SearchResult[]) => void;
  renderStoredSummary: (memory?: Memory) => void;
}

let chatDeps: ChatFormDeps | null = null;

export function initChatForm(deps: ChatFormDeps): void {
  chatDeps = deps;
  const form = byID<HTMLFormElement>("chat-form");
  if (form) {
    form.addEventListener("submit", (event) => {
      void submitChat(event);
    });
  }
}

export function refreshSelectedCardChatView(card: Card): void {
  renderCardMeta(card);
  renderTranscript();
}

export function setChatStatus(message: string, isError: boolean): void {
  const el = byID<HTMLDivElement>("conversation-status");
  if (!el) return;
  el.textContent = message;
  el.className = isError ? "text-sm text-red-300" : "text-sm text-[var(--app-fg-soft)]";
}

function renderCardMeta(card: Card): void {
  const meta = byID<HTMLDivElement>("card-meta");
  if (!meta) return;
  meta.innerHTML =
    '<div class="space-y-2">' +
      '<div><span class="text-[0.72rem] font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]">Name</span><p class="mt-1 text-base font-semibold text-[var(--app-fg)]">' + escapeHtml(card.name) + "</p></div>" +
      '<div><span class="text-[0.72rem] font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]">Tone</span><p class="mt-1 text-sm text-[var(--app-fg-muted)]">' + escapeHtml((card.personality || {}).tone || "") + "</p></div>" +
      '<div><span class="text-[0.72rem] font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]">Style Rules</span><ul class="mt-2 list-disc space-y-1 pl-5 text-sm text-[var(--app-fg-muted)]">' + ((card.personality || {}).style_rules || []).map((rule) => "<li>" + escapeHtml(rule) + "</li>").join("") + "</ul></div>" +
    "</div>";
}

function renderTranscript(): void {
  const transcriptEl = byID<HTMLDivElement>("transcript");
  if (!transcriptEl || !chatDeps) return;
  const cardId = chatDeps.getSelectedCardId();
  const transcript = chatDeps.getTranscript(cardId);
  if (!transcript.length) {
    transcriptEl.innerHTML = '<div class="rounded-2xl border border-dashed border-[var(--app-border)] px-4 py-8 text-center text-sm text-[var(--app-fg-soft)]">No local transcript yet for this card.</div>';
    return;
  }
  transcriptEl.innerHTML = transcript.map((item) => {
    return '<div class="space-y-3 rounded-2xl border border-[var(--app-border)] bg-[var(--app-surface-muted)] p-4">' +
      '<div class="rounded-2xl border border-[var(--app-border)] bg-[var(--app-panel)] px-4 py-3"><p class="text-xs font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]">User</p><p class="mt-2 text-sm text-[var(--app-fg)]">' + escapeHtml(item.user) + "</p></div>" +
      '<div class="rounded-2xl border border-[var(--app-border)] bg-[var(--app-panel)] px-4 py-3"><p class="text-xs font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]">Card</p><p class="mt-2 text-sm text-[var(--app-fg)]">' + escapeHtml(item.assistant) + "</p></div>" +
    "</div>";
  }).join("");
}

async function submitChat(event: SubmitEvent): Promise<void> {
  event.preventDefault();
  if (!chatDeps) return;
  const cardId = chatDeps.getSelectedCardId();
  if (!cardId) {
    setChatStatus("Select a card first.", true);
    return;
  }
  const input = byID<HTMLTextAreaElement>("chat-input");
  if (!input) return;
  const message = String(input.value || "").trim();
  if (!message) {
    setChatStatus("Message cannot be empty.", true);
    return;
  }
  setChatStatus("Generating response...", false);
  try {
    const payload = await chatDeps.sendChat(cardId, chatDeps.getUserId(), message);
    chatDeps.appendTranscript(cardId, {
      user: message,
      assistant: payload.assistant_response || "",
    });
    input.value = "";
    renderTranscript();
    chatDeps.renderRetrievedMemories(payload.retrieved_memories || []);
    chatDeps.renderStoredSummary(payload.stored_memory);
    await chatDeps.refreshRecentMemories();
    setChatStatus("Ready.", false);
  } catch (error) {
    setChatStatus(error instanceof Error ? error.message : "Chat request failed.", true);
  }
}
