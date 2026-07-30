// frontend/console/shell/src/features/list/infrastructure/dto/createListInput.ts

import type {
  ListPriceRow,
  ListStatus,
} from "../../../../shared/types/list";

export type CreateListInput = {
  id?: string;
  inventoryId?: string;

  title: string;
  description: string;

  priceRows?: ListPriceRow[];

  status?: ListStatus;

  assigneeId?: string;
};