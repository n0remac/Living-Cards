import type { CardHitZone, ComponentTarget } from "../types";

const borderBandPX = 24;

export interface CardHit {
  target: ComponentTarget;
  zone: CardHitZone;
  x: number;
  y: number;
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
    return { target: "border", zone: "border", x, y };
  }

  const target = event.target instanceof Element ? event.target : null;
  if (target?.closest('[data-component-type="textarea"]')) {
    return { target: "textarea", zone: "textarea", x, y };
  }

  return { target: "background", zone: "background", x, y };
}

function clamp(value: number): number {
  if (!Number.isFinite(value)) return 0;
  return Math.max(0, Math.min(1, value));
}
