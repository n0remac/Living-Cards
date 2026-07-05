import { initDesigner } from "./designer/controller";
import { initGameStage } from "./game/GameController";

document.addEventListener("DOMContentLoaded", () => {
  initDesigner();
  initGameStage();
});
