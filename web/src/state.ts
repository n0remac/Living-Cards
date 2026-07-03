import type { Card, TranscriptItem } from "./types";

export interface LivingCardState {
  cards: Card[];
  selectedCardId: string;
  userId: string;
  transcripts: Record<string, TranscriptItem[]>;
}

export const livingCardState: LivingCardState = {
  cards: [],
  selectedCardId: "",
  userId: "local-user",
  transcripts: {},
};
