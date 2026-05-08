// frontend/console/mintRequest/src/infrastructure/dto/mintRequestManagementRaw.dto.ts

import type { InspectionStatus } from "../../domain/entity/inspections";

/**
 * MintRequestQueryService が返す “一覧用 Raw DTO” をそのまま表現。
 * - フィールド名の揺れ（top-level / subdoc / casing）を許容する
 * - この型は「受信データの受け皿」であり、画面で使う型（VM）ではない
 */
export type MintRequestManagementRawDTO = {
  id?: string;
  productionId?: string;
  inspectionId?: string;

  // list fields (preferred)
  tokenName?: string | null;
  productName?: string | null;

  mintQuantity?: number | null;
  productionQuantity?: number | null;

  inspectionStatus?: InspectionStatus | string | null;

  requestedBy?: string | null;
  requestedByName?: string | null;
  createdByName?: string | null;

  mintedAt?: string | null;

  tokenBlueprintId?: string | null;

  // raw sub docs (optional)
  mint?: {
    tokenBlueprintId?: string | null;
    tokenBlueprintID?: string | null; // casing fallback
    tokenBlueprint?: string | null; // older list DTO name

    tokenName?: string | null;
    mintedAt?: string | null;

    createdBy?: string | null;
  } | null;

  inspection?: {
    status?: InspectionStatus | string | null;
    productName?: string | null;
  } | null;
};

/**
 * レスポンスが `{ items: [...] }` の場合の wrapper
 */
export type MintRequestManagementRawResponseDTO = {
  items?: MintRequestManagementRawDTO[] | null;
};
