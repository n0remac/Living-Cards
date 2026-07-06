export interface CardDocument {
  card_id: string;
  name: string;
  root: ComponentNode;
}

export interface ComponentNode {
  id: string;
  componentKind: string;
  config?: ConfigJSON;
  children?: ComponentNode[];
}

export type ConfigKind = "background" | "border" | "textarea" | "shape" | "image";
export type ComponentKind = "card" | "textarea" | "shape" | "image" | "slider";
export type ComponentTarget = ConfigKind | "slider" | "card" | "shadow" | "padding" | "textblock" | "button" | "layout";
export type CardHitZone = "border" | "background" | "textarea" | "shape" | "image" | "slider";
export type EditMode = "random" | "preset" | "simpleControls" | "advancedControls" | "aiPrompt" | "library";

export type ConfigJSON = Record<string, unknown>;

export interface GeneratedConfigEnvelope<T = ConfigJSON> {
  componentKind: string;
  description: string;
  config: T;
}

export type GeneratedConfig = GeneratedConfigEnvelope;

export interface ConfigIssue {
  path: string;
  code: string;
  message: string;
  actual?: unknown;
  allowed?: string[];
}

export interface DesignLibraryItem {
  id: string;
  name: string;
  componentKind: ConfigKind;
  description: string;
  config: ConfigJSON;
  saved?: boolean;
}

export interface RenderedDraftCard {
  document: CardDocument;
  preview_html: string;
  library: DesignLibraryItem[];
}

export interface ComponentKindProgress {
  taps: number;
  level: number;
  unlockedModes: EditMode[];
}

export interface ComponentProgress {
  componentId: string;
  componentKind: ComponentKind;
  xp: number;
  level: number;
  interactions: number;
  randomTapEnabled: boolean;
  preventRandomizing: boolean;
  overlayUnlocked: boolean;
  overlayOpened: boolean;
  unlockedTraits: string[];
  unlockedControls: string[];
}

export interface GameState {
  totalXp: number;
  globalLevel: number;
  totalInteractions: number;
  unlockedComponentKinds: ComponentKind[];
  unlockedConfigKinds: ConfigKind[];
  selectedComponentId?: string;
  componentProgress: Record<string, ComponentProgress>;
  tapCount: number;
  level: number;
  xp: number;
  unlockedModes: EditMode[];
  componentKindProgress: Record<string, ComponentKindProgress>;
}

export interface ComponentDescriptor {
  componentId: string;
  componentKind: ComponentKind;
  label: string;
  traits: string[];
}

export interface ControlOption {
  label: string;
  value: string;
}

export interface ControlDescriptor {
  trait: string;
  control: string;
  kind: "checkbox" | "color" | "range" | "select" | "text";
  label: string;
  value?: unknown;
  options?: ControlOption[];
  min?: number;
  max?: number;
  step?: number;
}

export interface ComponentOverlay {
  componentId: string;
  componentKind: ComponentKind;
  title: string;
  randomizeEnabled: boolean;
  controls: ControlDescriptor[];
}

export type CardEvent =
  | { type: "configApplied"; componentKind?: ComponentTarget; componentId?: string; trait?: string; control?: string }
  | { type: "controlChanged"; componentId: string; componentKind: ComponentKind; control: string }
  | { type: "componentAdded"; componentId: string; componentKind: ComponentKind; message?: string }
  | { type: "xpGained"; amount: number }
  | { type: "levelUp"; level: number }
  | { type: "componentLevelUp"; componentId: string; componentKind: ComponentKind; level: number }
  | { type: "componentUnlocked"; componentKind: ComponentKind; message?: string }
  | { type: "componentSelected"; componentId: string; componentKind: ComponentKind }
  | { type: "overlayOpened"; componentId: string; componentKind: ComponentKind }
  | { type: "configKindUnlocked"; componentKind: ConfigKind }
  | { type: "modeUnlocked"; componentKind: ComponentTarget; mode: EditMode }
  | { type: "invalidAction"; componentKind?: ComponentTarget; componentId?: string; message: string };

export interface InteractiveDraftCardResponse {
  document: CardDocument;
  gameState: GameState;
  preview_html: string;
  availableConfigKinds: ComponentTarget[];
  availableComponents: ComponentDescriptor[];
  overlay?: ComponentOverlay;
  library: DesignLibraryItem[];
}

export interface TapCardResponse {
  document: CardDocument;
  gameState: GameState;
  appliedConfig?: GeneratedConfig;
  preview_html: string;
  events: CardEvent[];
  overlay?: ComponentOverlay;
  library: DesignLibraryItem[];
}

export interface ApplyConfigResponse {
  document: CardDocument;
  normalized_config: GeneratedConfig;
  preview_html: string;
  library: DesignLibraryItem[];
}

export interface LibraryResponse {
  item?: DesignLibraryItem;
  library: DesignLibraryItem[];
}

export interface RenderedGameCard {
  id: string;
  name: string;
  kind: "world" | "item" | "clue" | string;
  tags?: string[];
  collectible: boolean;
  collected?: boolean;
  state?: Record<string, unknown>;
  document: CardDocument;
  preview_html: string;
}

export interface GameSessionSnapshot {
  worldDeck: RenderedGameCard[];
  activeWorldCard: RenderedGameCard;
  activeWorldCardId: string;
  activeIndex: number;
  library: RenderedGameCard[];
  solvedFlags: Record<string, boolean>;
  message?: string;
}
