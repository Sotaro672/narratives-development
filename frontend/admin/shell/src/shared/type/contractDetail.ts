// frontend/admin/shell/src/shared/type/contractDetail.ts

import type { Company } from "./company";

export type ContractListRow = {
  id: string;
  readableId: string;
  inventoryId: string;
  title: string;
  productName: string;
  tokenName: string;
  brandName: string;
  assigneeName: string;
  status: string;
  createdAt: string;
  updatedAt: string;
};

export type ContractTokenBlueprintRow = {
  id: string;
  name: string;
  symbol: string;
  brandName: string;
  assigneeName: string;
  minted: boolean;
  reportCount: number;
  createdAt: string;
  updatedAt: string;
};

export type ContractProductBlueprintRow = {
  id: string;
  productName: string;
  brandName: string;
  assigneeName: string;
  printed: boolean;
  createdAt: string;
  updatedAt: string;
};

export type ContractDetailResponse = {
  company: Company;
  lists: ContractListRow[];
  tokenBlueprints: ContractTokenBlueprintRow[];
  productBlueprints: ContractProductBlueprintRow[];
};