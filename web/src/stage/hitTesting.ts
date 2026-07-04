import type { CardHitZone, ComponentTarget, ComponentType } from "../types";

const borderBandPX = 24;

export interface CardHit {
  target: ComponentTarget;
  zone: CardHitZone;
  componentId: string;
  componentType: ComponentType;
  trait: string;
  x: number;
  y: number;
  clientX: number;
  clientY: number;
}

export function hitTestCard(event: PointerEvent, preview: HTMLElement): CardHit {
  const rect = preview.getBoundingClientRect();
  const x = clamp((event.clientX - rect.left) / rect.width);
  const y = clamp((event.clientY - rect.top) / rect.height);
  const localX = event.clientX - rect.left;
  const localY = event.clientY - rect.top;
  const inBorderBand =
    localX <= borderBandPX ||
    localY <= borderBandPX ||
    rect.width - localX <= borderBandPX ||
    rect.height - localY <= borderBandPX;

  if (inBorderBand) {
    return cardRootHit("border", "border", x, y, event);
  }

  const target = event.target instanceof Element ? event.target : null;
  const component = target?.closest<HTMLElement>("[data-component-id][data-component-type]");
  const componentType = component?.dataset.componentType as ComponentType | undefined;
  if (component && componentType === "shape") {
    return componentHit(component, "shape", "shape", "geometry", x, y, event);
  }
  if (component && componentType === "textarea") {
    return componentHit(component, "textarea", "textarea", "text", x, y, event);
  }

  return cardRootHit("background", "background", x, y, event);
}

function cardRootHit(target: ComponentTarget, zone: CardHitZone, x: number, y: number, event: PointerEvent): CardHit {
  return {
    target,
    zone,
    componentId: "card-root",
    componentType: "card",
    trait: target === "border" ? "border" : "background",
    x,
    y,
    clientX: event.clientX,
    clientY: event.clientY,
  };
}

function componentHit(
  element: HTMLElement,
  target: ComponentTarget,
  zone: CardHitZone,
  trait: string,
  x: number,
  y: number,
  event: PointerEvent,
): CardHit {
  return {
    target,
    zone,
    componentId: element.dataset.componentId || "",
    componentType: (element.dataset.componentType || target) as ComponentType,
    trait,
    x,
    y,
    clientX: event.clientX,
    clientY: event.clientY,
  };
}

function clamp(value: number): number {
  if (!Number.isFinite(value)) return 0;
  return Math.max(0, Math.min(1, value));
}
