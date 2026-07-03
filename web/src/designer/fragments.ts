import { FragmentGenerationError, generateFragment } from "../api";
import type { FragmentIssue, GeneratedStyleFragment } from "../types";

export async function generateTargetFragment(target: string, instruction: string, update = false): Promise<GeneratedStyleFragment> {
  return await generateFragment(target, instruction, update);
}

export function parseGeneratedFragment(raw: string): GeneratedStyleFragment {
  const trimmed = String(raw || "").trim();
  if (!trimmed || trimmed === "{}") {
    throw new Error("Generate or paste a fragment before applying.");
  }
  let parsed: unknown;
  try {
    parsed = JSON.parse(trimmed);
  } catch {
    throw new Error("Generated fragment is not valid JSON.");
  }
  return normalizeGeneratedFragment(parsed);
}

export function normalizeGeneratedFragment(value: unknown): GeneratedStyleFragment {
  if (!value || typeof value !== "object") {
    throw new Error("Generated fragment must be a JSON object.");
  }
  const record = value as Record<string, unknown>;
  const target = String(record.target || "").trim();
  const fragment = record.fragment;
  if (!fragment || typeof fragment !== "object") {
    throw new Error("Generated fragment must include a fragment object.");
  }
  return {
    target,
    description: String(record.description || savedDesignFallbackName(target)).trim(),
    fragment: cloneJSON(fragment as Record<string, unknown>),
  };
}

export function isFragmentGenerationError(error: unknown): error is FragmentGenerationError {
  return error instanceof FragmentGenerationError;
}

function cloneJSON<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T;
}

export function formatIssues(issues: FragmentIssue[]): string {
  return issues.slice(0, 3).map((issue) => {
    const path = String(issue.path || "$");
    const message = String(issue.message || issue.code || "invalid value");
    return path + " " + message;
  }).join("; ");
}

export function savedDesignFallbackName(target: string): string {
  switch (target) {
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
