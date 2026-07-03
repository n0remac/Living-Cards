import type { CardEvent } from "../types";

export function renderEvents(root: HTMLElement | null, events: CardEvent[]): void {
  if (!root) return;
  events.forEach((event, index) => {
    window.setTimeout(() => showEvent(root, event), index * 140);
  });
}

export function showMessage(root: HTMLElement | null, message: string, tone: "info" | "error" = "info"): void {
  if (!root) return;
  const toast = document.createElement("div");
  toast.className = toastClass(tone);
  toast.textContent = message;
  root.appendChild(toast);
  window.setTimeout(() => {
    toast.classList.add("opacity-0", "-translate-y-2");
  }, 1700);
  window.setTimeout(() => {
    toast.remove();
  }, 2200);
}

function showEvent(root: HTMLElement, event: CardEvent): void {
  switch (event.type) {
    case "fragmentApplied":
      showMessage(root, labelForTarget(event.target) + " changed");
      return;
    case "xpGained":
      showMessage(root, "+" + event.amount + " XP");
      return;
    case "levelUp":
      showMessage(root, "Level " + event.level);
      return;
    case "targetUnlocked":
      showMessage(root, labelForTarget(event.target) + " unlocked");
      return;
    case "modeUnlocked":
      showMessage(root, labelForTarget(event.target) + " " + event.mode + " unlocked");
      return;
    case "invalidAction":
      showMessage(root, event.message || "Locked", "error");
      return;
  }
}

function toastClass(tone: "info" | "error"): string {
  const base = "stage-toast pointer-events-none rounded-md border px-4 py-2 text-sm font-semibold shadow-xl backdrop-blur transition duration-500";
  if (tone === "error") {
    return base + " border-amber-200/35 bg-amber-950/80 text-amber-100";
  }
  return base + " border-emerald-200/35 bg-zinc-950/78 text-emerald-50";
}

function labelForTarget(target: string): string {
  switch (target) {
    case "background":
      return "Background";
    case "border":
      return "Border";
    case "textarea":
      return "Text";
    default:
      return "Card";
  }
}
