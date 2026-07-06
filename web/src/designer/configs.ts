import { ConfigGenerationError, generateConfig } from "../api";
import type { ConfigIssue, GeneratedConfig } from "../types";

export async function generateComponentConfig(componentKind: string, instruction: string, update = false): Promise<GeneratedConfig> {
  return await generateConfig(componentKind, instruction, update);
}

export function parseGeneratedConfigEnvelope(raw: string): GeneratedConfig {
  const trimmed = String(raw || "").trim();
  if (!trimmed || trimmed === "{}") {
    throw new Error("Generate or paste a config before applying.");
  }
  let parsed: unknown;
  try {
    parsed = JSON.parse(trimmed);
  } catch {
    throw new Error("Generated config is not valid JSON.");
  }
  return normalizeGeneratedConfigEnvelope(parsed);
}

export function normalizeGeneratedConfigEnvelope(value: unknown): GeneratedConfig {
  if (!value || typeof value !== "object") {
    throw new Error("Generated config must be a JSON object.");
  }
  const record = value as Record<string, unknown>;
  const componentKind = String(record.componentKind || "").trim();
  const config = record.config;
  if (!config || typeof config !== "object") {
    throw new Error("Generated config must include a config object.");
  }
  return {
    componentKind,
    description: String(record.description || savedDesignFallbackName(componentKind)).trim(),
    config: cloneJSON(config as Record<string, unknown>),
  };
}

export function isConfigGenerationError(error: unknown): error is ConfigGenerationError {
  return error instanceof ConfigGenerationError;
}

function cloneJSON<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T;
}

export function formatIssues(issues: ConfigIssue[]): string {
  return issues.slice(0, 3).map((issue) => {
    const path = String(issue.path || "$");
    const message = String(issue.message || issue.code || "invalid value");
    return path + " " + message;
  }).join("; ");
}

export function savedDesignFallbackName(componentKind: string): string {
  switch (componentKind) {
    case "background":
      return "Saved Background";
    case "border":
      return "Saved Border";
    case "textarea":
      return "Saved Text Area";
    default:
      return "Saved Design";
  }
}
