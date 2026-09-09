// frontend/admin/shell/src/shared/type/contractProductBlueprintDetail.ts

import type { Company } from "./company";

export type ContractProductBlueprintModelRef = {
  modelId: string;
  displayOrder: number;
  kind: string;
  modelNumber: string;
  size?: string;
  color?: string;
  rgb?: number;
  volumeValue?: number;
  volumeUnit?: string;
};

export type ContractProductBlueprintDetail = {
  id: string;
  productName: string;
  description: string;
  brandId: string;
  brandName: string;
  companyId: string;
  productBlueprintCategoryPath: string[];
  categoryFields: Record<string, unknown>;
  productIdTagType: string;
  assigneeId: string;
  assigneeName: string;
  modelRefs: ContractProductBlueprintModelRef[];
  printed: boolean;
  createdAt: string;
  updatedAt: string;
};

export type ContractProductBlueprintDetailResponse = {
  company: Company;
  productBlueprint: ContractProductBlueprintDetail;
};