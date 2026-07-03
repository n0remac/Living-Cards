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
