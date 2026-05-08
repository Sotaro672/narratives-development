// frontend/console/mintRequest/src/infrastructure/dto/inspectionBatch.dto.ts

import type { MintInspectionView } from "../../domain/entity/inspections";

/**
 * Backend → Frontend DTO
 *
 * 方針:
 * - domain/entity/inspections.ts を正として型を再利用する
 * - 既存コードの互換のため、InspectionBatchDTO は MintInspectionView の alias とする
 */
export type InspectionBatchDTO = MintInspectionView;
