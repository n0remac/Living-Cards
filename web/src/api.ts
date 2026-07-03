import type { Card, ChatResult, Memory } from "./types";

export async function fetchCards(): Promise<Card[]> {
  const response = await fetch("/api/cards", { cache: "no-store" });
  if (!response.ok) {
    throw new Error("Failed to load cards.");
  }
  return await response.json() as Card[];
}

export async function fetchCard(cardId: string): Promise<Card> {
  const response = await fetch("/api/cards/" + encodeURIComponent(cardId), { cache: "no-store" });
  if (!response.ok) {
    throw new Error("Failed to load card.");
  }
  return await response.json() as Card;
}

export async function fetchCardCanvas(cardId: string): Promise<string> {
  const response = await fetch("/api/cards/" + encodeURIComponent(cardId) + "/canvas", { cache: "no-store" });
  if (!response.ok) {
    throw new Error("Failed to load card canvas.");
  }
  return await response.text();
}

export async function fetchRecentMemories(cardId: string, userId: string): Promise<Memory[]> {
  const response = await fetch("/api/cards/" + encodeURIComponent(cardId) + "/memories?user_id=" + encodeURIComponent(userId), { cache: "no-store" });
  if (!response.ok) {
    throw new Error("Failed to load memories.");
  }
  return await response.json() as Memory[];
}

export async function sendChatMessage(cardId: string, userId: string, message: string): Promise<ChatResult> {
  const response = await fetch("/api/cards/" + encodeURIComponent(cardId) + "/components/chat-form/actions/send", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ user_id: userId, message }),
  });
  if (!response.ok) {
    throw new Error(await response.text() || "Chat request failed.");
  }
  return await response.json() as ChatResult;
}
