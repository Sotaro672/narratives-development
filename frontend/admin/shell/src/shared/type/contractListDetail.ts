// frontend/admin/shell/src/shared/type/contractListDetail.ts

import type { Company } from "./company";

export type ContractListPriceRow = {
  modelId: string;
  price: number;
};

export type ContractListDetail = {
  id: string;
  readableId: string;
  inventoryId: string;
  productBlueprintId: string;
  tokenBlueprintId: string;
  title: string;
  description: string;
  imageId: string;
  prices: ContractListPriceRow[];
  productName: string;
  tokenName: string;
  brandId: string;
  brandName: string;
  assigneeId: string;
  assigneeName: string;
  status: string;
  createdAt: string;
  updatedAt: string;
};

export type ContractListDetailResponse = {
  company: Company;
  list: ContractListDetail;
};