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

export type FragmentTarget = "background" | "border" | "textarea" | "shape";
export type ComponentType = "card" | "textarea" | "shape";
export type ComponentTarget = FragmentTarget | "card" | "shadow" | "padding" | "textblock" | "image" | "button" | "layout";
export type CardHitZone = "border" | "background" | "textarea" | "shape";
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

export interface ComponentProgress {
  componentId: string;
  componentType: ComponentType;
  xp: number;
  level: number;
  interactions: number;
  randomTapEnabled: boolean;
  overlayUnlocked: boolean;
  overlayOpened: boolean;
  unlockedTraits: string[];
  unlockedControls: string[];
}

export interface GameState {
  totalXp: number;
  globalLevel: number;
  totalInteractions: number;
  unlockedComponentTypes: ComponentType[];
  selectedComponentId?: string;
  componentProgress: Record<string, ComponentProgress>;
  tapCount: number;
  level: number;
  xp: number;
  unlockedTargets: ComponentTarget[];
  unlockedModes: EditMode[];
  targetProgress: Record<string, TargetProgress>;
}

export interface ComponentDescriptor {
  componentId: string;
  componentType: ComponentType;
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
  kind: "color" | "range" | "select" | "text";
  label: string;
  value?: unknown;
  options?: ControlOption[];
  min?: number;
  max?: number;
  step?: number;
}

export interface ComponentOverlay {
  componentId: string;
  componentType: ComponentType;
  title: string;
  randomizeEnabled: boolean;
  controls: ControlDescriptor[];
}

export type CardEvent =
  | { type: "fragmentApplied"; target?: ComponentTarget; componentId?: string; componentType?: ComponentType; trait?: string; control?: string }
  | { type: "xpGained"; amount: number }
  | { type: "levelUp"; level: number }
  | { type: "componentLevelUp"; componentId: string; componentType: ComponentType; level: number }
  | { type: "componentUnlocked"; componentType: ComponentType; message?: string }
  | { type: "componentSelected"; componentId: string; componentType: ComponentType }
  | { type: "overlayOpened"; componentId: string; componentType: ComponentType }
  | { type: "targetUnlocked"; target: ComponentTarget }
  | { type: "modeUnlocked"; target: ComponentTarget; mode: EditMode }
  | { type: "invalidAction"; target?: ComponentTarget; componentId?: string; message: string };

export interface InteractiveDraftCardResponse {
  document: CardDocument;
  gameState: GameState;
  preview_html: string;
  availableTargets: ComponentTarget[];
  availableComponents: ComponentDescriptor[];
  overlay?: ComponentOverlay;
  library: DesignLibraryItem[];
}

export interface TapCardResponse {
  document: CardDocument;
  gameState: GameState;
  appliedFragment?: GeneratedStyleFragment;
  preview_html: string;
  events: CardEvent[];
  overlay?: ComponentOverlay;
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
