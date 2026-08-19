// frontend/console/shell/src/features/productBlueprint/infrastructure/api/productBlueprintUpdateApi.ts

import type {
  ApparelModelNumberRow,
  ApparelSizeInput,
  Fit,
} from "../../../../shared/types/apparel";
import type {
  CategoryFieldValues,
  ProductBlueprintCategoryPath,
} from "../../domain/productBlueprintCategory";

// ------------------------------------------------------
// Update ProductBlueprint
// ------------------------------------------------------

export type UpdateProductBlueprintParams = {
  id: string;
  productName: string;
  brandId: string;
  productBlueprintCategoryPath: ProductBlueprintCategoryPath;
  categoryFields?: CategoryFieldValues | null;

  /**
   * Apparel category fields.
   *
   * ProductBlueprint.categoryFieldsへ集約する方針を維持しつつ、
   * presentation/application層からは従来どおり
   * 個別フィールドとしても渡せるようにする。
   */
  fit?: Fit | null;
  material?: string | null;
  weight?: number | null;
  qualityAssurance?: string[] | null;

  productIdTagType: string | null;
  companyId: string;
  assigneeId: string;

  /**
   * TODO:
   * ProductBlueprint本体更新APIからは将来的に分離する。
   *
   * 本来、colors / sizes / modelNumbersは
   * ModelVariation側の責務。
   */
  colors: string[];
  colorRgbMap?: Record<string, string>;
  sizes?: ApparelSizeInput[];
  modelNumbers?: ApparelModelNumberRow[];
  updatedBy?: string | null;
};