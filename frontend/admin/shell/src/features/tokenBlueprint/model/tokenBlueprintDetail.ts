// frontend\admin\shell\src\features\tokenBlueprint\model\tokenBlueprintDetail.ts

import type { Company } from "../../../shared/type/company";

export type ContractTokenBlueprintContentFile = {
  id: string;
  name: string;
  type: string;
  contentType: string;
  url: string;
  isPublic: boolean;
  size: number;
  createdAt: string;
  updatedAt: string;
};

export type ContractTokenBlueprintDetail = {
  id: string;
  name: string;
  symbol: string;
  brandId: string;
  brandName: string;
  companyId: string;
  description: string;
  assigneeId: string;
  assigneeName: string;
  minted: boolean;
  moderationStatus: string;
  metadataUri: string;
  iconUrl: string;
  contentFiles: ContractTokenBlueprintContentFile[];
  likeCount: number;
  dislikeCount: number;
  commentCount: number;
  reportCount: number;
  createdAt: string;
  updatedAt: string;
};

export type ContractTokenBlueprintDetailResponse = {
  company: Company;
  tokenBlueprint: ContractTokenBlueprintDetail;
};