//frontend\console\production\src\application\productBlueprint\types.ts
/**
 * ProductBlueprint 詳細用
 * ※ Dateに寄せたい場合は Production 同様に toDate を導入可能
 */
export type ProductBlueprintDetail = {
  id: string;

  productName: string;
  companyId: string;
  brandId: string;
  itemType: string;
  fit: string;
  material: string;
  weight: number;

  qualityAssurance: string[];
  productIdTag: string;
  assigneeId: string;

  createdBy?: string | null;
  createdAt: string;
  updatedBy?: string | null;
  updatedAt: string;

  deletedBy?: string | null;
  deletedAt?: string | null;

  expireAt?: string | null;
};
