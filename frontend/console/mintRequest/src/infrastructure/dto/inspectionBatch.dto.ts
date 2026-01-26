// frontend/console/mintRequest/src/infrastructure/dto/inspectionBatch.dto.ts

import type {
  InspectionItem,
  InspectionStatus,
  MintInspectionView,
} from "../../domain/entity/inspections";

/**
 * Backend → Frontend DTO
 *
 * 方針:
 * - domain/entity/inspections.ts を正として型を再利用する
 * - 既存コードの互換のため、InspectionBatchDTO は MintInspectionView の alias とする
 */

export type InspectionStatusDTO = InspectionStatus;
export type InspectionItemDTO = InspectionItem;

/**
 * MintUsecase が返す “検査バッチ 1 件” の DTO（既存の定義に合わせる）
 */
export type InspectionBatchDTO = MintInspectionView;
