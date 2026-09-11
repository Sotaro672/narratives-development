// frontend/admin/shell/src/shared/type/contractDetail.ts

import type { Company } from "./company";

export type ContractBrandRow = {
  id: string;
  name: string;
  managerName: string;
  brandIcon: string;
  brandBackgroundImage: string;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
};

export type ContractAnnouncementRow = {
  id: string;
  title: string;
  tokenBlueprintId: string;
  tokenName: string;
  published: boolean;
  targetAvatarCount: number;
  createdAt: string;
  updatedAt: string;
};

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
  reportCount: number;
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
  reportCount: number;
  createdAt: string;
  updatedAt: string;
};

export type ContractDetailResponse = {
  company: Company;
  brands: ContractBrandRow[];
  announcements: ContractAnnouncementRow[];
  lists: ContractListRow[];
  tokenBlueprints: ContractTokenBlueprintRow[];
  productBlueprints: ContractProductBlueprintRow[];
};