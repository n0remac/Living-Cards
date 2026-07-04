import type { CardEvent } from "../types";

type NotificationTone = "info" | "error";

interface NotificationItem {
  message: string;
  tone: NotificationTone;
}

const visibleMS = 2600;
const notificationQueue: NotificationItem[] = [];
const notificationHistory: NotificationItem[] = [];
let notificationTimer = 0;
let activeNotification = false;

export function initNotifications(): void {
  const section = notificationSection();
  if (!section) return;
  stopCardTapEvents(section);
  const history = notificationHistoryPanel();
  if (history) {
    stopCardTapEvents(history);
  }
  section.addEventListener("click", (event) => {
    event.stopPropagation();
    toggleHistory();
  });
  document.addEventListener("click", (event) => {
    const history = notificationHistoryPanel();
    if (!history || history.classList.contains("hidden")) return;
    const target = event.target instanceof Node ? event.target : null;
    if (target && (history.contains(target) || section.contains(target))) return;
    history.classList.add("hidden");
  });
  renderHistory();
}

export function renderEvents(root: HTMLElement | null, events: CardEvent[]): void {
  if (!root) return;
  events.forEach((event) => showEvent(root, event));
}

export function showMessage(root: HTMLElement | null, message: string, tone: NotificationTone = "info"): void {
  if (!root) return;
  enqueueNotification({ message, tone });
}

function showEvent(root: HTMLElement | null, event: CardEvent): void {
  switch (event.type) {
    case "fragmentApplied":
      return;
    case "xpGained":
      return;
    case "levelUp":
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

function enqueueNotification(item: NotificationItem): void {
  notificationQueue.push(item);
  notificationHistory.unshift(item);
  if (notificationHistory.length > 30) {
    notificationHistory.length = 30;
  }
  renderHistory();
  showNextNotification();
}

function showNextNotification(): void {
  if (activeNotification) return;
  const current = notificationCurrent();
  const section = notificationSection();
  if (!current || !section) return;
  const item = notificationQueue.shift();
  if (!item) {
    current.textContent = "No notifications";
    section.dataset.tone = "empty";
    return;
  }
  activeNotification = true;
  current.textContent = item.message;
  section.dataset.tone = item.tone;
  window.clearTimeout(notificationTimer);
  notificationTimer = window.setTimeout(() => {
    activeNotification = false;
    showNextNotification();
  }, visibleMS);
}

function toggleHistory(): void {
  const history = notificationHistoryPanel();
  if (!history) return;
  history.classList.toggle("hidden");
  renderHistory();
}

function stopCardTapEvents(element: HTMLElement): void {
  for (const eventName of ["pointerdown", "pointermove", "pointerup", "pointercancel", "contextmenu"]) {
    element.addEventListener(eventName, (event) => {
      event.stopPropagation();
    });
  }
}

function renderHistory(): void {
  const list = notificationHistoryList();
  if (!list) return;
  list.innerHTML = "";
  if (!notificationHistory.length) {
    const empty = document.createElement("div");
    empty.className = "stage-notification-history-empty";
    empty.textContent = "No notifications yet.";
    list.appendChild(empty);
    return;
  }
  notificationHistory.forEach((item) => {
    const row = document.createElement("div");
    row.className = "stage-notification-history-item";
    row.dataset.tone = item.tone;
    row.textContent = item.message;
    list.appendChild(row);
  });
}

function notificationSection(): HTMLButtonElement | null {
  return document.getElementById("stage-notification-section") as HTMLButtonElement | null;
}

function notificationCurrent(): HTMLSpanElement | null {
  return document.getElementById("stage-notification-current") as HTMLSpanElement | null;
}

function notificationHistoryPanel(): HTMLDivElement | null {
  return document.getElementById("stage-notification-history") as HTMLDivElement | null;
}

function notificationHistoryList(): HTMLDivElement | null {
  return document.getElementById("stage-notification-history-list") as HTMLDivElement | null;
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
