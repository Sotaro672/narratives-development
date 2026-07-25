// frontend/console/list/src/infrastructure/dto/updateListInput.ts
import type { ListStatus } from "../../domain/list";

export type UpdateListInput = {
  listId: string;

  title?: string;
  description?: string;

  priceRows?: Array<{
    modelId: string;
    price: number | null;

    size?: string;
    color?: string;
    stock?: number;
    rgb?: number | null;
  }>;

  status?: ListStatus;

  assigneeId?: string;
  updatedBy?: string;
};