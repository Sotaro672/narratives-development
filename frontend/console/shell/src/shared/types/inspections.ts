// frontend/console/shell/src/shared/types/inspections.ts

/**
 * InspectionResult
 *
 * Backend BFF / inspection domain が返す値をそのまま扱う。
 *
 * - "notYet"          : 未検査
 * - "passed"          : 合格
 * - "failed"          : 不合格
 * - "notManufactured" : 生産されていない
 */
export type InspectionResult =
  | "notYet"
  | "passed"
  | "failed"
  | "notManufactured";

/**
 * InspectionStatus
 *
 * Backend BFF / inspection domain が返す値をそのまま扱う。
 *
 * - "inspecting" : 検査中
 * - "completed"  : 検査完了
 */
export type InspectionStatus =
  | "inspecting"
  | "completed";

/**
 * InspectionItem
 *
 * GET /mint/inspections/{productionId} の
 * inspection.inspections に対応する。
 *
 * modelNumberなどのモデル表示情報は保持せず、
 * Backend BFF のmodelMetaを正とする。
 */
export interface InspectionItem {
  productId: string;
  modelId: string;
  inspectionResult?: InspectionResult | null;
  inspectedBy?: string | null;
  inspectedAt?: string | null;
}

/**
 * InspectionBatch
 *
 * GET /mint/inspections/{productionId} の
 * inspectionフィールドに対応する。
 *
 * Frontend側では再構築・normalization・domain validationを行わず、
 * Backend BFF responseを正とする。
 */
export interface InspectionBatch {
  productionId: string;
  status: InspectionStatus;
  quantity: number;
  totalPassed: number;
  inspections: InspectionItem[];
}

/**
 * modelId -> モデル表示情報。
 *
 * GET /mint/inspections/{productionId} の
 * modelMetaに対応する。
 *
 * apparel:
 * - modelNumber
 * - size
 * - colorName
 * - rgb
 *
 * alcohol:
 * - modelNumber
 * - volume
 * - volumeUnit
 */
export interface MintModelMeta {
  modelId?: string;
  kind?: string;
  modelNumber?: string;
  size?: string;
  colorName?: string;
  rgb?: number;
  volume?: number;
  volumeUnit?: string;
}