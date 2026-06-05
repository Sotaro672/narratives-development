// frontend/console/productBlueprint/src/infrastructure/api/productBlueprintUpdateApi.ts

import type {
  ApparelModelNumberRow,
  ApparelSizeInput,
} from "../../domain/entity/apparel";

import type {
  CategoryFieldValues,
  ProductBlueprintCategorySnapshot,
} from "../../domain/entity/productBlueprintCategory";

export type UpdateProductBlueprintParams = {
  id: string;

  productName: string;
  brandId: string;

  productBlueprintCategoryId: string;
  productBlueprintCategory: ProductBlueprintCategorySnapshot;

  categoryFields?: CategoryFieldValues | null;

  productIdTagType: string | null;

  companyId: string;
  assigneeId: string;

  /**
   * TODO:
   * ProductBlueprint 本体更新 API からは将来的に分離したい。
   * 本来 colors / sizes / modelNumbers は model variation 側の責務。
   */
  colors: string[];
  colorRgbMap?: Record<string, string>;

  sizes?: ApparelSizeInput[];
  modelNumbers?: ApparelModelNumberRow[];

  updatedBy?: string | null;
};