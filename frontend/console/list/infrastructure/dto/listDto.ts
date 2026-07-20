// frontend/console/list/src/infrastructure/dto/listDto.ts
import type { ListStatus } from "../../domain/list";
import type { ListDetailPriceRowDTO } from "./listPriceRowDto";

export type ListDTO = {
  id: string;
  inventoryId?: string;

  status?: ListStatus;

  title?: string;
  description?: string;

  assigneeId?: string;
  assigneeName?: string;

  createdBy?: string;
  createdByName?: string;
  createdAt?: string;

  updatedBy?: string;
  updatedByName?: string;
  updatedAt?: string;

  productBlueprintId?: string;
  tokenBlueprintId?: string;

  productBrandId?: string;
  productBrandName?: string;
  productName?: string;

  tokenBrandId?: string;
  tokenBrandName?: string;
  tokenName?: string;

  imageId?: string;
  imageUrls?: string[];

  priceRows?: ListDetailPriceRowDTO[];
  totalStock?: number;
  currencyJpy?: boolean;
};