import type {
  ComponentTemplateInput,
  TextConfig,
  TextConfigInput,
} from "../src/generated/card-types.generated";

const validTextInput: TextConfigInput = {
  "content": "Defaults fill the remaining fields.",
};

const validTemplateInput: ComponentTemplateInput<"text"> = {
  "component_kind": "text",
  "config": validTextInput,
};

// @ts-expect-error Canonical configs contain every field.
const incompleteCanonicalConfig: TextConfig = {
  "content": "Missing canonical fields.",
};

const nullTemplateInput: ComponentTemplateInput<"text"> = {
  "component_kind": "text",
  // @ts-expect-error Input config may be omitted, but it may not be null.
  "config": null,
};

// @ts-expect-error exactOptionalPropertyTypes rejects explicit undefined.
const undefinedTextInput: TextConfigInput = {
  "content": undefined,
};

const mismatchedTemplate: ComponentTemplateInput<"text"> = {
  // @ts-expect-error The discriminant remains correlated with its config kind.
  "component_kind": "shape",
};

void validTemplateInput;
void incompleteCanonicalConfig;
void nullTemplateInput;
void undefinedTextInput;
void mismatchedTemplate;
