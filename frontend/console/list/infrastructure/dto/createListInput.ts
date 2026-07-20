// frontend/console/list/src/infrastructure/dto/createListInput.ts
import type { ListStatus } from "../../domain/list";

export type CreateListInput = {
  id?: string;
  inventoryId?: string;

  title: string;
  description: string;

  priceRows?: Array<{
    modelId: string;
    price: number | null;

    size: string;
    color: string;
    stock: number;
    rgb?: number | null;
  }>;

  status?: ListStatus;

  assigneeId?: string;
  createdBy?: string;
};