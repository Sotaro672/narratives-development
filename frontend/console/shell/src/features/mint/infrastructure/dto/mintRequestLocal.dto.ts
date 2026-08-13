// frontend/console/shell/src/features/mint/infrastructure/dto/mintRequestLocal.dto.ts 
 
import type { 
  InspectionBatchDTO, 
} from "../../../../shared/types/inspections"; 
 
import type { 
  ProductIDTag, 
} from "../../../../shared/types/productBlueprint"; 
 
import type { 
  ProductBlueprintCategorySnapshot, 
  CategoryFieldValues, 
} from "../../../productBlueprint/domain/productBlueprintCategory"; 
 
/** 
 * ProductBlueprint.modelRefs取得用DTO。 
 * 
 * displayOrderはProductBlueprint側にのみ存在する前提のため、 
 * UIはこの値を正として扱う。 
 */ 
export type ProductBlueprintModelRefDTO = { 
  modelId: string; 
  displayOrder: number; 
}; 
 
export type ProductBlueprintPatchDTO = { 
  productName?: string | null; 
  description?: string | null; 
 
  brandId?: string | null; 
  brandName?: string | null; 
  companyId?: string | null; 
 
  /** 
   * 商品カテゴリ。 
   * 
   * ProductBlueprint側にdenormalize保存される 
   * カテゴリsnapshotを正とする。 
   * 
   * itemTypeは廃止し、カテゴリ判定には 
   * productBlueprintCategoryを使用する。 
   */ 
  productBlueprintCategory?: 
    | ProductBlueprintCategorySnapshot 
    | null; 
 
  /** 
   * カテゴリ別入力値。 
   */ 
  categoryFields?: 
    | CategoryFieldValues 
    | null; 
 
  /** 
   * 商品へ付与する識別タグ。 
   * 
   * shared/types/productBlueprint.tsの 
   * ProductIDTagを正規型として使用する。 
   * 
   * フィールド名はtypeのみを使用し、 
   * 旧形式のTypeは保持しない。 
   */ 
  productIdTag?: 
    | ProductIDTag 
    | null; 
 
  assigneeId?: string | null; 
 
  /** 
   * displayOrderの唯一のソース。 
   * 
   * ProductBlueprint.modelRefsを正とする。 
   */ 
  modelRefs?: 
    | ProductBlueprintModelRefDTO[] 
    | null; 
}; 
 
/** 
 * GET /mint/inspections/{productionId} が返す 
 * inspection部分のDTO。 
 * 
 * productBlueprintId / productName / modelMetaは 
 * responseのトップレベルを正とする。 
 */ 
export type MintRequestInspectionDTO = Pick< 
  InspectionBatchDTO, 
  | "productionId" 
  | "status" 
  | "quantity" 
  | "totalPassed" 
  | "inspections" 
>; 
 
/** 
 * GET /mint/inspections/{productionId} の 
 * Backend BFF response。 
 * 
 * Backend responseのfield名・型を正とし、 
 * 旧形式のfallback用fieldは保持しない。 
 */ 
export type MintRequestDetailDTO = { 
  productBlueprintId?: string; 
 
  productName: string; 
 
  modelMeta?: 
    InspectionBatchDTO["modelMeta"]; 
 
  inspection?: 
    | MintRequestInspectionDTO 
    | null; 
};