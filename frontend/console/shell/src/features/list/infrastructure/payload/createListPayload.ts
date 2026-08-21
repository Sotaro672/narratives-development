// frontend/console/shell/src/features/list/infrastructure/payload/createListPayload.ts

import type { CreateListInput } from "../dto/createListInput";

export function buildCreateListPayloadArray(
  input: CreateListInput,
): Record<string, unknown> {
  const inventoryId = input.inventoryId ?? "";

  const id = input.id || inventoryId;

  if (!id) {
    throw new Error("missing_id");
  }

  if (!input.title) {
    throw new Error("missing_title");
  }

  if (!input.transportationOption) {
    throw new Error("missing_transportation_option");
  }

  if (
    input.transportationOption === "custom" &&
    !input.transportationId
  ) {
    throw new Error("missing_transportation_id");
  }

  return {
    id,

    inventoryId,

    title: input.title,

    description: input.description,

    status: input.status,

    assigneeId: input.assigneeId || undefined,

    transportationOption:
      input.transportationOption,

    transportationId:
      input.transportationOption === "custom"
        ? input.transportationId
        : undefined,

    prices: input.priceRows,
  };
}