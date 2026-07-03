import { initChatForm } from "../../internal/web/components/chatform/client";
import type { ChatResult, Memory, SearchResult, TranscriptItem } from "./types";

export interface HydrationDeps {
  getSelectedCardId: () => string;
  getUserId: () => string;
  getTranscript: (cardId: string) => TranscriptItem[];
  appendTranscript: (cardId: string, item: TranscriptItem) => void;
  sendChat: (cardId: string, userId: string, message: string) => Promise<ChatResult>;
  refreshRecentMemories: () => Promise<void>;
  renderRetrievedMemories: (items: SearchResult[]) => void;
  renderStoredSummary: (memory?: Memory) => void;
}

type Initializer = (deps: HydrationDeps) => void;

const initializers: Record<string, Initializer> = {
  "chat-form": initChatForm,
};

export function hydrateCardCanvas(root: HTMLElement, deps: HydrationDeps): void {
  root.querySelectorAll<HTMLElement>("[data-component-type]").forEach((node) => {
    const componentType = node.dataset.componentType || "";
    const initializer = initializers[componentType];
    if (initializer) {
      initializer(deps);
    }
  });
}
