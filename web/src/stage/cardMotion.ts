export function animateCardTap(preview: HTMLElement, x: number, y: number): void {
  const rotateY = (x - 0.5) * 8;
  const rotateX = (0.5 - y) * 8;
  runAnimation(preview, [
    { transform: "translateY(0) scale(1) rotateX(0deg) rotateY(0deg)" },
    { transform: `translateY(-8px) scale(1.018) rotateX(${rotateX}deg) rotateY(${rotateY}deg)` },
    { transform: "translateY(0) scale(1) rotateX(0deg) rotateY(0deg)" },
  ], 260);
}

export function animateInvalidTap(preview: HTMLElement): void {
  runAnimation(preview, [
    { transform: "translateX(0)" },
    { transform: "translateX(-7px)" },
    { transform: "translateX(6px)" },
    { transform: "translateX(-4px)" },
    { transform: "translateX(0)" },
  ], 230);
}

function runAnimation(element: HTMLElement, keyframes: Keyframe[], duration: number): void {
  if (typeof element.animate !== "function") return;
  element.getAnimations().forEach((animation) => animation.cancel());
  element.animate(keyframes, {
    duration,
    easing: "cubic-bezier(.2,.8,.2,1)",
  });
}
