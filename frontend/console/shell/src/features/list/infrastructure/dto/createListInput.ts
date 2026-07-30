// frontend/console/shell/src/features/list/infrastructure/dto/createListInput.ts

import type { ListStatus } from "../../../../shared/types/list";

export type CreateListInput = {
  id?: string;
  inventoryId?: string;

  title: string;
  description: string;

  priceRows?: Array<{
    modelId: string;
    price: number;

    size: string;
    color: string;
    stock: number;
    rgb?: number | null;
  }>;

  status?: ListStatus;

  assigneeId?: string;
  createdBy?: string;
};