//frontend\console\list\src\infrastructure\dto\updateListInput.ts
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

  decision?: "list" | "hold";

  assigneeId?: string;
  updatedBy?: string;
};