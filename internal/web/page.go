package web

import . "github.com/n0remac/GoDom/html"

func Page() *Node {
	return Html(
		Attr("data-theme", "dark"),
		pageHead("Living Card", pageCSS()),
		Body(
			Class("min-h-screen bg-[var(--app-bg)] text-[var(--app-fg)]"),
			Div(
				Class("flex min-h-screen flex-col"),
				Header(
					Class("border-b border-[var(--app-border-strong)] bg-[var(--app-surface)]/95 backdrop-blur"),
					Div(
						Class("mx-auto flex w-full max-w-none flex-wrap items-center gap-3 px-4 py-4 md:px-6"),
						Div(
							Class("mr-auto flex min-w-[14rem] flex-col"),
							H1(Class("text-2xl font-semibold tracking-tight text-[var(--app-fg)]"), T("Living Card")),
							P(Class("text-sm text-[var(--app-fg-muted)]"), T("A persistent AI entity with memory, retrieval, and local inference.")),
						),
						Button(
							Id("reload-cards-btn"),
							Type("button"),
							Class(uiSecondaryButtonClass("sm")),
							T("Reload Cards"),
						),
					),
				),
				Main(
					Class("flex-1 px-4 pb-4 pt-3 md:px-6"),
					Div(
						Class("grid gap-4 xl:grid-cols-[18rem_minmax(0,1.2fr)_minmax(0,0.9fr)]"),
						Aside(
							Class("rounded-3xl border border-[var(--app-border-strong)] bg-[var(--app-surface)] p-4 shadow-sm backdrop-blur"),
							H2(Class("text-sm font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]"), T("Cards")),
							Div(Id("card-list"), Class("mt-4 flex max-h-[calc(100vh-12rem)] flex-col gap-2 overflow-y-auto pr-1")),
						),
						Section(
							Class("rounded-3xl border border-[var(--app-border-strong)] bg-[var(--app-surface)] p-4 shadow-sm backdrop-blur"),
							Div(
								Class("flex items-center justify-between gap-3"),
								H2(Class("text-sm font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]"), T("Conversation")),
								Div(Id("conversation-status"), Class("text-sm text-[var(--app-fg-soft)]"), T("Select a card to begin.")),
							),
							Div(Id("card-meta"), Class("mt-4 rounded-2xl border border-[var(--app-border)] bg-[var(--app-surface-muted)] p-4 text-sm text-[var(--app-fg-muted)]"), T("No card selected.")),
							Div(Id("transcript"), Class("mt-4 space-y-3")),
							Form(
								Id("chat-form"),
								Class("mt-4 space-y-3"),
								Div(
									Class("space-y-2"),
									Label(Class("block text-[0.72rem] font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]"), T("Message")),
									TextArea(
										Id("chat-input"),
										Name("message"),
										Class(uiInputClass()+" min-h-28 resize-y"),
										Placeholder("Ask the card a question..."),
									),
								),
								Button(
									Id("send-btn"),
									Type("submit"),
									Class(uiPrimaryButtonClass("sm")),
									T("Send"),
								),
							),
						),
						Section(
							Class("rounded-3xl border border-[var(--app-border-strong)] bg-[var(--app-surface)] p-4 shadow-sm backdrop-blur"),
							H2(Class("text-sm font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]"), T("Debug")),
							Div(
								Class("mt-4 space-y-4"),
								Div(
									Class("rounded-2xl border border-[var(--app-border)] bg-[var(--app-surface-muted)] p-4"),
									H3(Class("text-xs font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]"), T("Retrieved Memories")),
									Div(Id("retrieved-memories"), Class("mt-3 space-y-3 text-sm text-[var(--app-fg-muted)]"), T("No retrieval yet.")),
								),
								Div(
									Class("rounded-2xl border border-[var(--app-border)] bg-[var(--app-surface-muted)] p-4"),
									H3(Class("text-xs font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]"), T("Stored Summary")),
									Div(Id("stored-summary"), Class("mt-3 text-sm text-[var(--app-fg-muted)]"), T("No interaction stored yet.")),
								),
								Div(
									Class("rounded-2xl border border-[var(--app-border)] bg-[var(--app-surface-muted)] p-4"),
									H3(Class("text-xs font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]"), T("Recent Memories")),
									Div(Id("recent-memories"), Class("mt-3 space-y-3 text-sm text-[var(--app-fg-muted)]"), T("No card selected.")),
								),
							),
						),
					),
				),
			),
			Script(Raw(pageScript())),
		),
	)
}

func pageCSS() string {
	return `
.card-item.is-active {
  border-color: rgba(34, 211, 238, 0.68);
  background: rgba(8, 145, 178, 0.14);
}
`
}

func pageScript() string {
	return `
var livingCardState = {
  cards: [],
  selectedCardId: '',
  transcripts: {}
};

function escapeHtml(value) {
  return String(value || '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function setStatus(message, isError) {
  var el = document.getElementById('conversation-status');
  if (!el) return;
  el.textContent = message;
  el.className = isError ? 'text-sm text-red-300' : 'text-sm text-[var(--app-fg-soft)]';
}

async function loadCards() {
  setStatus('Loading cards...', false);
  var response = await fetch('/api/cards', { cache: 'no-store' });
  if (!response.ok) {
    setStatus('Failed to load cards.', true);
    return;
  }
  livingCardState.cards = await response.json();
  if (!livingCardState.selectedCardId && livingCardState.cards.length) {
    livingCardState.selectedCardId = livingCardState.cards[0].card_id;
  }
  renderCards();
  await refreshSelectedCard();
}

function renderCards() {
  var list = document.getElementById('card-list');
  if (!list) return;
  list.innerHTML = '';
  if (!livingCardState.cards.length) {
    list.innerHTML = '<div class="rounded-2xl border border-dashed border-[var(--app-border)] px-4 py-8 text-center text-sm text-[var(--app-fg-soft)]">No cards available.</div>';
    return;
  }
  livingCardState.cards.forEach(function(card) {
    var button = document.createElement('button');
    button.type = 'button';
    button.className = 'card-item rounded-2xl border border-[var(--app-border)] bg-[var(--app-surface-muted)] px-4 py-3 text-left text-[var(--app-fg)] shadow-sm transition hover:-translate-y-0.5 hover:border-cyan-400/60 hover:bg-cyan-500/10';
    if (card.card_id === livingCardState.selectedCardId) {
      button.classList.add('is-active');
    }
    button.innerHTML =
      '<div class="flex items-center justify-between gap-2">' +
        '<span class="font-semibold text-[var(--app-fg)]">' + escapeHtml(card.name) + '</span>' +
        '<span class="rounded-full border border-cyan-400/30 bg-cyan-400/12 px-2 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.18em] text-cyan-200">' + escapeHtml(card.card_id) + '</span>' +
      '</div>' +
      '<p class="mt-2 text-xs text-[var(--app-fg-muted)]">' + escapeHtml((card.domain || []).join(', ') || card.archetype || '') + '</p>';
    button.addEventListener('click', async function() {
      livingCardState.selectedCardId = card.card_id;
      renderCards();
      await refreshSelectedCard();
    });
    list.appendChild(button);
  });
}

async function refreshSelectedCard() {
  var cardId = livingCardState.selectedCardId;
  if (!cardId) {
    return;
  }
  var cardResponse = await fetch('/api/cards/' + encodeURIComponent(cardId), { cache: 'no-store' });
  if (!cardResponse.ok) {
    setStatus('Failed to load card.', true);
    return;
  }
  var card = await cardResponse.json();
  renderCardMeta(card);
  renderTranscript();
  await refreshRecentMemories();
  setStatus('Ready.', false);
}

function renderCardMeta(card) {
  var meta = document.getElementById('card-meta');
  if (!meta) return;
  meta.innerHTML =
    '<div class="space-y-2">' +
      '<div><span class="text-[0.72rem] font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]">Name</span><p class="mt-1 text-base font-semibold text-[var(--app-fg)]">' + escapeHtml(card.name) + '</p></div>' +
      '<div><span class="text-[0.72rem] font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]">Tone</span><p class="mt-1 text-sm text-[var(--app-fg-muted)]">' + escapeHtml((card.personality || {}).tone || '') + '</p></div>' +
      '<div><span class="text-[0.72rem] font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]">Style Rules</span><ul class="mt-2 list-disc space-y-1 pl-5 text-sm text-[var(--app-fg-muted)]">' + ((card.personality || {}).style_rules || []).map(function(rule) { return '<li>' + escapeHtml(rule) + '</li>'; }).join('') + '</ul></div>' +
    '</div>';
}

function renderTranscript() {
  var transcriptEl = document.getElementById('transcript');
  if (!transcriptEl) return;
  var cardId = livingCardState.selectedCardId;
  var transcript = livingCardState.transcripts[cardId] || [];
  if (!transcript.length) {
    transcriptEl.innerHTML = '<div class="rounded-2xl border border-dashed border-[var(--app-border)] px-4 py-8 text-center text-sm text-[var(--app-fg-soft)]">No local transcript yet for this card.</div>';
    return;
  }
  transcriptEl.innerHTML = transcript.map(function(item) {
    return '<div class="space-y-3 rounded-2xl border border-[var(--app-border)] bg-[var(--app-surface-muted)] p-4">' +
      '<div class="rounded-2xl border border-[var(--app-border)] bg-[var(--app-panel)] px-4 py-3"><p class="text-xs font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]">User</p><p class="mt-2 text-sm text-[var(--app-fg)]">' + escapeHtml(item.user) + '</p></div>' +
      '<div class="rounded-2xl border border-[var(--app-border)] bg-[var(--app-panel)] px-4 py-3"><p class="text-xs font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]">Card</p><p class="mt-2 text-sm text-[var(--app-fg)]">' + escapeHtml(item.assistant) + '</p></div>' +
    '</div>';
  }).join('');
}

function renderRetrievedMemories(items) {
  var el = document.getElementById('retrieved-memories');
  if (!el) return;
  if (!items.length) {
    el.innerHTML = '<p>No retrieved memories for this turn.</p>';
    return;
  }
  el.innerHTML = items.map(function(item) {
    return '<div class="rounded-2xl border border-[var(--app-border)] bg-[var(--app-panel)] px-4 py-3">' +
      '<p class="text-xs font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]">Rank ' + escapeHtml(item.rank) + ' · Score ' + escapeHtml(Number(item.score || 0).toFixed(3)) + '</p>' +
      '<p class="mt-2 text-sm text-[var(--app-fg)]">' + escapeHtml((item.memory || {}).summary || '') + '</p>' +
    '</div>';
  }).join('');
}

function renderStoredSummary(memory) {
  var el = document.getElementById('stored-summary');
  if (!el) return;
  if (!memory || !memory.summary) {
    el.textContent = 'No interaction stored yet.';
    return;
  }
  el.innerHTML = '<p class="text-sm text-[var(--app-fg)]">' + escapeHtml(memory.summary) + '</p><p class="mt-2 text-xs text-[var(--app-fg-soft)]">' + escapeHtml(memory.timestamp || '') + '</p>';
}

async function refreshRecentMemories() {
  var el = document.getElementById('recent-memories');
  if (!el) return;
  if (!livingCardState.selectedCardId) {
    el.textContent = 'No card selected.';
    return;
  }
  var response = await fetch('/api/cards/' + encodeURIComponent(livingCardState.selectedCardId) + '/memories', { cache: 'no-store' });
  if (!response.ok) {
    el.innerHTML = '<p>Failed to load memories.</p>';
    return;
  }
  var memories = await response.json();
  if (!memories.length) {
    el.innerHTML = '<p>No stored memories for this card yet.</p>';
    return;
  }
  el.innerHTML = memories.map(function(item) {
    return '<div class="rounded-2xl border border-[var(--app-border)] bg-[var(--app-panel)] px-4 py-3">' +
      '<p class="text-xs font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]">' + escapeHtml(item.timestamp || '') + '</p>' +
      '<p class="mt-2 text-sm text-[var(--app-fg)]">' + escapeHtml(item.summary || '') + '</p>' +
    '</div>';
  }).join('');
}

async function submitChat(event) {
  event.preventDefault();
  var cardId = livingCardState.selectedCardId;
  if (!cardId) {
    setStatus('Select a card first.', true);
    return;
  }
  var input = document.getElementById('chat-input');
  if (!input) return;
  var message = String(input.value || '').trim();
  if (!message) {
    setStatus('Message cannot be empty.', true);
    return;
  }
  setStatus('Generating response...', false);
  var response = await fetch('/api/cards/' + encodeURIComponent(cardId) + '/chat', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ message: message })
  });
  if (!response.ok) {
    var errorText = await response.text();
    setStatus(errorText || 'Chat request failed.', true);
    return;
  }
  var payload = await response.json();
  if (!livingCardState.transcripts[cardId]) {
    livingCardState.transcripts[cardId] = [];
  }
  livingCardState.transcripts[cardId].push({
    user: message,
    assistant: payload.assistant_response || ''
  });
  input.value = '';
  renderTranscript();
  renderRetrievedMemories(payload.retrieved_memories || []);
  renderStoredSummary(payload.stored_memory || {});
  await refreshRecentMemories();
  setStatus('Ready.', false);
}

document.addEventListener('DOMContentLoaded', function() {
  var reload = document.getElementById('reload-cards-btn');
  if (reload) reload.addEventListener('click', loadCards);
  var form = document.getElementById('chat-form');
  if (form) form.addEventListener('submit', submitChat);
  loadCards();
});
`
}
