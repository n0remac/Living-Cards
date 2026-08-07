import type {
  CardDocument,
  ComponentConfigInput,
  ComponentKind,
  ComponentLibraryItem,
  ComponentTemplate,
  GeneratedComponentKind,
  GeneratedConfigEnvelope,
  InstallableComponentKind,
  LeafComponentKind,
  RootComponentKind,
} from "./generated/card-types.generated";

export type {
  AIGeneratableComponentKind,
  BackgroundConfig,
  BackgroundConfigInput,
  BorderConfig,
  BorderConfigInput,
  ButtonConfig,
  ButtonConfigInput,
  CardConfig,
  CardConfigInput,
  CardDocument,
  CardDocumentInput,
  ComponentConfig,
  ComponentConfigInput,
  ComponentConfigInputMap,
  ComponentConfigMap,
  ComponentKind,
  ComponentLibraryItem,
  ComponentNode,
  ComponentNodeInput,
  ComponentTemplate,
  ComponentTemplateInput,
  GeneratedComponentKind,
  GeneratedConfigEnvelope,
  GeneratedConfigEnvelopeInput,
  ImageConfig,
  ImageConfigInput,
  InstallableComponentKind,
  LeafComponentKind,
  PresetLibraryItem,
  RootComponentKind,
  ShapeConfig,
  ShapeConfigInput,
  SliderConfig,
  SliderConfigInput,
  TextConfig,
  TextConfigInput,
  TextInputConfig,
  TextInputConfigInput,
} from "./generated/card-types.generated";

export type ConfigKind = Exclude<GeneratedComponentKind, RootComponentKind>;
export type ComponentTarget = ComponentKind | "shadow" | "padding" | "textblock" | "layout";
export type CardHitZone = LeafComponentKind;
export type EditMode = "random" | "preset" | "simpleControls" | "advancedControls" | "aiPrompt" | "library";

export type ConfigJSON = ComponentConfigInput<InstallableComponentKind>;
export type GeneratedConfig = GeneratedConfigEnvelope;

export interface ConfigIssue {
  path: string;
  code: string;
  message: string;
  actual?: unknown;
  allowed?: string[];
}

export type DesignLibraryItem = ComponentLibraryItem<ConfigKind>;

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
  kind: "checkbox" | "color" | "position" | "range" | "select" | "text";
  label: string;
  property?: string;
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
  state?: Record<string, unknown> & {
    component_template?: ComponentTemplate;
  };
  document: CardDocument;
  preview_html: string;
}

export interface GameEditSession {
  targetCardId: string;
  draftCard: RenderedGameCard;
  pendingConsumedComponentIds?: string[];
  selectedComponentId?: string;
  editingOverlay?: ComponentOverlay;
}

export interface GameSessionSnapshot {
  worldDeck: RenderedGameCard[];
  activeWorldCard: RenderedGameCard;
  activeWorldCardId: string;
  activeIndex: number;
  activeEditingComponentId?: string;
  activeEditingOverlay?: ComponentOverlay;
  library: RenderedGameCard[];
  editSession?: GameEditSession;
  solvedFlags: Record<string, boolean>;
  message?: string;
}

export type GameEvent =
  | { sequence: number; type: "sessionReset"; payload?: Record<string, never> }
  | { sequence: number; type: "cardCycled"; payload: { direction: string; previousCardId: string; activeCardId: string } }
  | { sequence: number; type: "cardCollected"; payload: { cardId: string; previousWorldIndex: number; activeCardId: string } }
  | { sequence: number; type: "cardPlayed"; payload: { sourceCardId: string; targetCardId: string; outcome: "resolved" | "conditionsFailed" | "noMatchingRule" } }
  | { sequence: number; type: "cardConsumed"; payload: { cardId: string } }
  | { sequence: number; type: "formSubmitted"; payload: { cardId: string; formId: string } }
  | { sequence: number; type: "componentSelected"; payload: GameComponentEventPayload }
  | { sequence: number; type: "componentChanged"; payload: GameComponentEventPayload }
  | { sequence: number; type: "editStarted"; payload: { cardId: string } }
  | { sequence: number; type: "editComponentInstalled"; payload: { cardId: string; componentCardId: string; componentId: string; componentKind: ComponentKind } }
  | { sequence: number; type: "editSaved"; payload: { cardId: string } }
  | { sequence: number; type: "editCanceled"; payload: { cardId: string } }
  | { sequence: number; type: "flagChanged"; payload: { flag: string; value: boolean } }
  | { sequence: number; type: "cardStateChanged"; payload: { cardId: string; key: string; value: unknown } }
  | { sequence: number; type: "cardTagsRemoved"; payload: { cardId: string; tags: string[] } }
  | { sequence: number; type: "cardVariantChanged"; payload: { cardId: string; variant: string } }
  | { sequence: number; type: "componentMounted"; payload: { sourceCardId: string; targetCardId: string; componentId: string; componentKind: ComponentKind } }
  | { sequence: number; type: "creatureAttacked"; payload: { sourceCardId: string; targetCardId: string; attack: number; previousHealth: number; health: number } }
  | { sequence: number; type: "deckLoaded"; payload: { deckId: string } }
  | { sequence: number; type: "ruleResolved"; payload: { ruleId: string; triggerKind: "cardPlayed" | "formSubmitted" | "componentUpdated"; outcome: "success" | "conditionsFailed" } }
  | { sequence: number; type: "actionRejected"; payload: { action: string; outcome: string } }
  | { sequence: number; type: "message"; message: string };

export interface GameComponentEventPayload {
  cardId: string;
  componentId: string;
  componentKind: ComponentKind;
  control?: string;
  scope: "world" | "library" | "edit";
}

export interface GameResult {
  revision: number;
  snapshot: GameSessionSnapshot;
  events: GameEvent[];
}
