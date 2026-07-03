// web/src/dom.ts
function escapeHtml(value) {
  return String(value || "").replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;").replace(/'/g, "&#39;");
}
function byID(id) {
  return document.getElementById(id);
}

// internal/web/components/appheader/client.ts
function initAppHeader(deps) {
  const reload = byID("reload-cards-btn");
  if (reload) {
    reload.addEventListener("click", () => {
      void deps.loadCards();
    });
  }
}

// internal/web/components/chatform/client.ts
var chatDeps = null;
function initChatForm(deps) {
  chatDeps = deps;
  const form = byID("chat-form");
  if (form) {
    form.addEventListener("submit", (event) => {
      void submitChat(event);
    });
  }
}
function refreshSelectedCardChatView(card) {
  renderCardMeta(card);
  renderTranscript();
}
function setChatStatus(message, isError) {
  const el = byID("conversation-status");
  if (!el) return;
  el.textContent = message;
  el.className = isError ? "text-sm text-red-300" : "text-sm text-[var(--app-fg-soft)]";
}
function renderCardMeta(card) {
  const meta = byID("card-meta");
  if (!meta) return;
  meta.innerHTML = '<div class="space-y-2"><div><span class="text-[0.72rem] font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]">Name</span><p class="mt-1 text-base font-semibold text-[var(--app-fg)]">' + escapeHtml(card.name) + '</p></div><div><span class="text-[0.72rem] font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]">Tone</span><p class="mt-1 text-sm text-[var(--app-fg-muted)]">' + escapeHtml((card.personality || {}).tone || "") + '</p></div><div><span class="text-[0.72rem] font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]">Style Rules</span><ul class="mt-2 list-disc space-y-1 pl-5 text-sm text-[var(--app-fg-muted)]">' + ((card.personality || {}).style_rules || []).map((rule) => "<li>" + escapeHtml(rule) + "</li>").join("") + "</ul></div></div>";
}
function renderTranscript() {
  const transcriptEl = byID("transcript");
  if (!transcriptEl || !chatDeps) return;
  const cardId = chatDeps.getSelectedCardId();
  const transcript = chatDeps.getTranscript(cardId);
  if (!transcript.length) {
    transcriptEl.innerHTML = '<div class="rounded-2xl border border-dashed border-[var(--app-border)] px-4 py-8 text-center text-sm text-[var(--app-fg-soft)]">No local transcript yet for this card.</div>';
    return;
  }
  transcriptEl.innerHTML = transcript.map((item) => {
    return '<div class="space-y-3 rounded-2xl border border-[var(--app-border)] bg-[var(--app-surface-muted)] p-4"><div class="rounded-2xl border border-[var(--app-border)] bg-[var(--app-panel)] px-4 py-3"><p class="text-xs font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]">User</p><p class="mt-2 text-sm text-[var(--app-fg)]">' + escapeHtml(item.user) + '</p></div><div class="rounded-2xl border border-[var(--app-border)] bg-[var(--app-panel)] px-4 py-3"><p class="text-xs font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]">Card</p><p class="mt-2 text-sm text-[var(--app-fg)]">' + escapeHtml(item.assistant) + "</p></div></div>";
  }).join("");
}
async function submitChat(event) {
  event.preventDefault();
  if (!chatDeps) return;
  const cardId = chatDeps.getSelectedCardId();
  if (!cardId) {
    setChatStatus("Select a card first.", true);
    return;
  }
  const input = byID("chat-input");
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
      assistant: payload.assistant_response || ""
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

// web/src/api.ts
async function fetchCards() {
  const response = await fetch("/api/cards", { cache: "no-store" });
  if (!response.ok) {
    throw new Error("Failed to load cards.");
  }
  return await response.json();
}
async function fetchCard(cardId) {
  const response = await fetch("/api/cards/" + encodeURIComponent(cardId), { cache: "no-store" });
  if (!response.ok) {
    throw new Error("Failed to load card.");
  }
  return await response.json();
}
async function fetchCardCanvas(cardId) {
  const response = await fetch("/api/cards/" + encodeURIComponent(cardId) + "/canvas", { cache: "no-store" });
  if (!response.ok) {
    throw new Error("Failed to load card canvas.");
  }
  return await response.text();
}
async function fetchRecentMemories(cardId, userId) {
  const response = await fetch("/api/cards/" + encodeURIComponent(cardId) + "/memories?user_id=" + encodeURIComponent(userId), { cache: "no-store" });
  if (!response.ok) {
    throw new Error("Failed to load memories.");
  }
  return await response.json();
}
async function sendChatMessage(cardId, userId, message) {
  const response = await fetch("/api/cards/" + encodeURIComponent(cardId) + "/components/chat-form/actions/send", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ user_id: userId, message })
  });
  if (!response.ok) {
    throw new Error(await response.text() || "Chat request failed.");
  }
  return await response.json();
}

// web/src/components.ts
var initializers = {
  "chat-form": initChatForm
};
function hydrateCardCanvas(root, deps) {
  root.querySelectorAll("[data-component-type]").forEach((node) => {
    const componentType = node.dataset.componentType || "";
    const initializer = initializers[componentType];
    if (initializer) {
      initializer(deps);
    }
  });
}

// web/src/state.ts
var livingCardState = {
  cards: [],
  selectedCardId: "",
  userId: "local-user",
  transcripts: {}
};

// web/src/app.ts
async function loadCards() {
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
function renderCards() {
  const list = byID("card-list");
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
    button.innerHTML = '<div class="flex items-center justify-between gap-2"><span class="font-semibold text-[var(--app-fg)]">' + escapeHtml(card.name) + '</span><span class="rounded-full border border-cyan-400/30 bg-cyan-400/12 px-2 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.18em] text-cyan-200">' + escapeHtml(card.card_id) + '</span></div><p class="mt-2 text-xs text-[var(--app-fg-muted)]">' + escapeHtml((card.domain || []).join(", ") || card.archetype || "") + "</p>";
    button.addEventListener("click", async () => {
      livingCardState.selectedCardId = card.card_id;
      renderCards();
      await refreshSelectedCard();
    });
    list.appendChild(button);
  });
}
async function refreshSelectedCard() {
  const cardId = livingCardState.selectedCardId;
  if (!cardId) {
    return;
  }
  let card;
  try {
    card = await fetchCard(cardId);
  } catch {
    setChatStatus("Failed to load card.", true);
    return;
  }
  if (!await refreshCardCanvas(card.card_id)) {
    return;
  }
  refreshSelectedCardChatView(card);
  await refreshRecentMemories();
  setChatStatus("Ready.", false);
}
async function refreshCardCanvas(cardId) {
  const canvas = byID("card-canvas");
  if (!canvas) return false;
  try {
    canvas.innerHTML = await fetchCardCanvas(cardId);
  } catch {
    setChatStatus("Failed to load card canvas.", true);
    return false;
  }
  hydrateCardCanvas(canvas, hydrationDeps());
  return true;
}
function renderRetrievedMemories(items) {
  const el = byID("retrieved-memories");
  if (!el) return;
  if (!items.length) {
    el.innerHTML = "<p>No retrieved memories for this turn.</p>";
    return;
  }
  el.innerHTML = items.map((item) => {
    return '<div class="rounded-2xl border border-[var(--app-border)] bg-[var(--app-panel)] px-4 py-3"><p class="text-xs font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]">Rank ' + escapeHtml(item.rank) + " &middot; Score " + escapeHtml(Number(item.score || 0).toFixed(3)) + '</p><p class="mt-2 text-sm text-[var(--app-fg)]">' + escapeHtml((item.memory || {}).summary || "") + "</p></div>";
  }).join("");
}
function renderStoredSummary(memory) {
  const el = byID("stored-summary");
  if (!el) return;
  if (!memory || !memory.summary) {
    el.textContent = "No interaction stored yet.";
    return;
  }
  el.innerHTML = '<p class="text-sm text-[var(--app-fg)]">' + escapeHtml(memory.summary) + '</p><p class="mt-2 text-xs text-[var(--app-fg-soft)]">' + escapeHtml(memory.timestamp || "") + "</p>";
}
async function refreshRecentMemories() {
  const el = byID("recent-memories");
  if (!el) return;
  if (!livingCardState.selectedCardId) {
    el.textContent = "No card selected.";
    return;
  }
  let memories;
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
    return '<div class="rounded-2xl border border-[var(--app-border)] bg-[var(--app-panel)] px-4 py-3"><p class="text-xs font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]">' + escapeHtml(item.timestamp || "") + '</p><p class="mt-2 text-sm text-[var(--app-fg)]">' + escapeHtml(item.summary || "") + "</p></div>";
  }).join("");
}
function getTranscript(cardId) {
  return livingCardState.transcripts[cardId] || [];
}
function appendTranscript(cardId, item) {
  if (!livingCardState.transcripts[cardId]) {
    livingCardState.transcripts[cardId] = [];
  }
  livingCardState.transcripts[cardId].push(item);
}
document.addEventListener("DOMContentLoaded", () => {
  initAppHeader({ loadCards });
  void loadCards();
});
function hydrationDeps() {
  return {
    getSelectedCardId: () => livingCardState.selectedCardId,
    getUserId: () => livingCardState.userId,
    getTranscript,
    appendTranscript,
    sendChat: sendChatMessage,
    refreshRecentMemories,
    renderRetrievedMemories,
    renderStoredSummary
  };
}
//# sourceMappingURL=app.js.map
