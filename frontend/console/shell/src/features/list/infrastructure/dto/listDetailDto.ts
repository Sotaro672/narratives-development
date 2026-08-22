// frontend/console/shell/src/features/list/infrastructure/dto/listDetailDto.ts

import type { ListStatus } from "../../../../shared/types/list";
import type { ListDetailPriceRowDTO } from "./listPriceRowDto";

export type ListDetailImageDTO = {
  id: string;
  url: string;
  displayOrder: number;
};

export type ListDetailDTO = {
  id: string;
  readableId: string;
  inventoryId: string;
  status: ListStatus;
  title: string;
  description: string;
  assigneeId: string;
  assigneeName: string;
  createdBy: string;
  createdByName: string;
  createdAt: string;
  updatedBy?: string;
  updatedByName?: string;
  updatedAt?: string;
  productBlueprintId: string;
  productBrandId: string;
  productBrandName: string;
  productName: string;
  tokenBlueprintId: string;
  tokenBrandId: string;
  tokenBrandName: string;
  tokenName: string;
  primaryImageId?: string;
  images: ListDetailImageDTO[];
  priceRows: ListDetailPriceRowDTO[];
};