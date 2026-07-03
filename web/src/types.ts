export interface Personality {
  tone?: string;
  style_rules?: string[];
}

export interface Card {
  card_id: string;
  name: string;
  domain?: string[];
  archetype?: string;
  personality?: Personality;
  components?: ComponentInstance[];
}

export interface ComponentInstance {
  id: string;
  type: string;
  props?: Record<string, unknown>;
}

export interface Memory {
  id?: string;
  user_id?: string;
  card_id?: string;
  timestamp?: string;
  user_input?: string;
  card_response?: string;
  summary?: string;
  tags?: string[];
  importance?: number;
  collection_name?: string;
}

export interface SearchResult {
  memory?: Memory;
  rank: number;
  score: number;
}

export interface ChatResult {
  user_id?: string;
  card?: Card;
  assistant_response?: string;
  retrieved_memories?: SearchResult[];
  stored_memory?: Memory;
}

export interface TranscriptItem {
  user: string;
  assistant: string;
}
