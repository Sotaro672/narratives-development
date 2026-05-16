//frontend\console\list\src\infrastructure\dto\createListInput.ts
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

  decision?: "list" | "hold";

  assigneeId?: string;
  createdBy?: string;
};