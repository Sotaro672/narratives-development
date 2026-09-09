// frontend/admin/shell/src/shared/type/contractListDetail.ts

import type { Company } from "./company";

export type ContractListPriceRow = {
  modelId: string;
  kind: string;
  modelNumber: string;
  size?: string;
  color?: string;
  rgb?: number;
  volumeValue?: number;
  volumeUnit?: string;
  price: number;
};

export type ContractListImage = {
  id: string;
  url: string;
  displayOrder: number;
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
  images: ContractListImage[];
  prices: ContractListPriceRow[];
  productName: string;
  tokenName: string;
  productBrandId: string;
  productBrandName: string;
  tokenBrandId: string;
  tokenBrandName: string;
  totalOrderCount: number;
  reportCount: number;
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