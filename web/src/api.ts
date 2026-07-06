import type {
  ApplyFragmentResponse,
  CardDocument,
  CardHitZone,
  ComponentTarget,
  DesignLibraryItem,
  FragmentIssue,
  FragmentJSON,
  GameSessionSnapshot,
  GeneratedStyleFragment,
  InteractiveDraftCardResponse,
  LibraryResponse,
  RenderedDraftCard,
  TapCardResponse,
} from "./types";

export class FragmentGenerationError extends Error {
  rawResponse: string;
  issues: FragmentIssue[];

  constructor(message: string, rawResponse: string, issues: FragmentIssue[] = []) {
    super(message);
    this.name = "FragmentGenerationError";
    this.rawResponse = rawResponse;
    this.issues = issues;
  }
}

export async function fetchRenderedDraftCard(): Promise<RenderedDraftCard> {
  const response = await fetch("/api/draft-card/rendered", { cache: "no-store" });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to load draft card."));
  }
  return await response.json() as RenderedDraftCard;
}

export async function fetchInteractiveDraftCard(): Promise<InteractiveDraftCardResponse> {
  const response = await fetch("/api/draft-card/interactive", { cache: "no-store" });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to load interactive card."));
  }
  return await response.json() as InteractiveDraftCardResponse;
}

export async function resetDraftCard(): Promise<RenderedDraftCard> {
  const response = await fetch("/api/draft-card/reset", { method: "POST" });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to reset draft card."));
  }
  return await response.json() as RenderedDraftCard;
}

export async function tapCardZone(target: ComponentTarget, zone: CardHitZone, x: number, y: number): Promise<TapCardResponse> {
  const response = await fetch("/api/draft-card/tap", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ target, zone, x, y }),
  });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to apply card tap."));
  }
  return await response.json() as TapCardResponse;
}

export async function interactWithComponent(
  componentId: string,
  trait: string,
  interaction: "shortTap" | "longPress",
  x: number,
  y: number,
): Promise<TapCardResponse> {
  const response = await fetch("/api/draft-card/interact", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ componentId, trait, interaction, x, y }),
  });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to apply interaction."));
  }
  return await response.json() as TapCardResponse;
}

export interface ColorControlPayload {
  color: string;
  secondaryColor?: string;
  gradient?: boolean;
  angle?: number;
}

export async function applyComponentControl(componentId: string, trait: string, control: string, value: unknown): Promise<TapCardResponse> {
  const response = await fetch("/api/draft-card/control-change", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ componentId, trait, control, value }),
  });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to apply control."));
  }
  return await response.json() as TapCardResponse;
}

export async function randomizeComponent(componentId: string, trait = "", scope = "unlockedTraits"): Promise<TapCardResponse> {
  const response = await fetch("/api/draft-card/randomize-component", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ componentId, trait, scope }),
  });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to randomize component."));
  }
  return await response.json() as TapCardResponse;
}

export async function applyColorControl(target: ComponentTarget, control: ColorControlPayload): Promise<TapCardResponse> {
  const response = await fetch("/api/draft-card/control-change", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ target, ...control }),
  });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to apply color."));
  }
  return await response.json() as TapCardResponse;
}

export async function generateFragment(target: string, instruction: string, update = false): Promise<GeneratedStyleFragment> {
  const response = await fetch("/api/draft-card/fragments/" + encodeURIComponent(target), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ instruction, update }),
  });
  if (!response.ok) {
    throw await readFragmentError(response, "Failed to generate fragment.");
  }
  return await response.json() as GeneratedStyleFragment;
}

export async function applyDraftFragment(generatedFragment: GeneratedStyleFragment): Promise<ApplyFragmentResponse> {
  const response = await fetch("/api/draft-card/apply-fragment", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ generated_fragment: generatedFragment }),
  });
  if (!response.ok) {
    throw await readFragmentError(response, "Failed to apply fragment.");
  }
  return await response.json() as ApplyFragmentResponse;
}

export async function fetchDesignLibrary(target = ""): Promise<DesignLibraryItem[]> {
  const suffix = target ? "?target=" + encodeURIComponent(target) : "";
  const response = await fetch("/api/draft-card/library" + suffix, { cache: "no-store" });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to load design library."));
  }
  const payload = await response.json() as LibraryResponse;
  return payload.library || [];
}

export async function saveAppliedDesign(): Promise<LibraryResponse> {
  const response = await fetch("/api/draft-card/library/save-applied", { method: "POST" });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to save applied design."));
  }
  return await response.json() as LibraryResponse;
}

export async function applyLibraryDesign(itemID: string): Promise<ApplyFragmentResponse> {
  const response = await fetch("/api/draft-card/library/apply", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ item_id: itemID }),
  });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to apply library design."));
  }
  return await response.json() as ApplyFragmentResponse;
}

export async function fetchGameSession(): Promise<GameSessionSnapshot> {
  const response = await fetch("/api/game/session", { cache: "no-store" });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to load game session."));
  }
  return await response.json() as GameSessionSnapshot;
}

export async function resetGameSession(): Promise<GameSessionSnapshot> {
  const response = await fetch("/api/game/reset", { method: "POST" });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to reset game session."));
  }
  return await response.json() as GameSessionSnapshot;
}

export async function cycleGameCard(direction: "next" | "previous"): Promise<GameSessionSnapshot> {
  const response = await fetch("/api/game/cycle", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ direction }),
  });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to cycle card."));
  }
  return await response.json() as GameSessionSnapshot;
}

export async function collectGameCard(cardId: string): Promise<GameSessionSnapshot> {
  const response = await fetch("/api/game/collect", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ cardId }),
  });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to collect card."));
  }
  return await response.json() as GameSessionSnapshot;
}

export async function playGameCard(sourceCardId: string, targetCardId: string): Promise<GameSessionSnapshot> {
  const response = await fetch("/api/game/play-card", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ sourceCardId, targetCardId }),
  });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to play card."));
  }
  return await response.json() as GameSessionSnapshot;
}

export async function saveControllerCard(templateCardId: string, document: CardDocument): Promise<GameSessionSnapshot> {
  const response = await fetch("/api/game/save-controller", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ templateCardId, document }),
  });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to save controller."));
  }
  return await response.json() as GameSessionSnapshot;
}

export async function addDraftComponent(componentType: "textarea" | "shape" | "image", fragment?: FragmentJSON): Promise<TapCardResponse> {
  const response = await fetch("/api/draft-card/components", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ componentType, fragment }),
  });
  if (!response.ok) {
    throw new Error(await readError(response, "Failed to add component."));
  }
  return await response.json() as TapCardResponse;
}

async function readError(response: Response, fallback: string): Promise<string> {
  const text = String(await response.text() || "").trim();
  return text || fallback;
}

async function readFragmentError(response: Response, fallback: string): Promise<Error> {
  const contentType = response.headers.get("Content-Type") || "";
  if (contentType.includes("application/json")) {
    try {
      const payload = await response.json() as { message?: string; raw_response?: string; issues?: FragmentIssue[] };
      const message = String(payload.message || fallback).trim();
      const rawResponse = String(payload.raw_response || "").trim();
      const issues = Array.isArray(payload.issues) ? payload.issues : [];
      if (rawResponse || issues.length) {
        return new FragmentGenerationError(message, rawResponse, issues);
      }
      return new Error(message);
    } catch {
      return new Error(fallback);
    }
  }
  return new Error(await readError(response, fallback));
}
