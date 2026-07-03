export interface CardDocument {
  card_id: string;
  name: string;
  root: ComponentNode;
}

export interface ComponentNode {
  id: string;
  type: string;
  fragment?: FragmentJSON;
  children?: ComponentNode[];
}

export type FragmentTarget = "background" | "border" | "textarea";
export type ComponentTarget = FragmentTarget | "shadow" | "padding" | "textblock" | "image" | "button" | "layout";
export type CardHitZone = "border" | "background" | "textarea";
export type EditMode = "random" | "preset" | "simpleControls" | "advancedControls" | "aiPrompt" | "library";

export type FragmentJSON = Record<string, unknown>;

export interface GeneratedFragment<T = FragmentJSON> {
  target: string;
  description: string;
  fragment: T;
}

export type GeneratedStyleFragment = GeneratedFragment;

export interface FragmentIssue {
  path: string;
  code: string;
  message: string;
  actual?: unknown;
  allowed?: string[];
}

export interface DesignLibraryItem {
  id: string;
  name: string;
  target: FragmentTarget;
  description: string;
  fragment: FragmentJSON;
  saved?: boolean;
}

export interface RenderedDraftCard {
  document: CardDocument;
  preview_html: string;
  library: DesignLibraryItem[];
}

export interface TargetProgress {
  taps: number;
  level: number;
  unlockedModes: EditMode[];
}

export interface GameState {
  tapCount: number;
  level: number;
  xp: number;
  unlockedTargets: ComponentTarget[];
  unlockedModes: EditMode[];
  targetProgress: Record<string, TargetProgress>;
}

export type CardEvent =
  | { type: "fragmentApplied"; target: ComponentTarget }
  | { type: "xpGained"; amount: number }
  | { type: "levelUp"; level: number }
  | { type: "targetUnlocked"; target: ComponentTarget }
  | { type: "modeUnlocked"; target: ComponentTarget; mode: EditMode }
  | { type: "invalidAction"; target?: ComponentTarget; message: string };

export interface InteractiveDraftCardResponse {
  document: CardDocument;
  gameState: GameState;
  preview_html: string;
  availableTargets: ComponentTarget[];
  library: DesignLibraryItem[];
}

export interface TapCardResponse {
  document: CardDocument;
  gameState: GameState;
  appliedFragment?: GeneratedStyleFragment;
  preview_html: string;
  events: CardEvent[];
  library: DesignLibraryItem[];
}

export interface ApplyFragmentResponse {
  document: CardDocument;
  normalized_fragment: GeneratedStyleFragment;
  preview_html: string;
  library: DesignLibraryItem[];
}

export interface LibraryResponse {
  item?: DesignLibraryItem;
  library: DesignLibraryItem[];
}
