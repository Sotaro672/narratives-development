// frontend/console/model/src/application/buildMeasurements.ts 

import type { ItemType } from "../../../productBlueprint/src/domain/entity/catalog";
import type { SizeRow } from "../domain/entity/catalog";

/**
 * itemType に応じて ModelVariationPayload.measurements を組み立てるユーティリティ
 *
 * NewModelVariationPayload['measurements'] に合わせて
 * chest / waist / length / shoulder の4項目だけを返す。
 */
export function buildMeasurements(itemType: ItemType, size: SizeRow) {
  // ボトムスの場合: ウエスト / 丈 を優先して埋める
  if (itemType === "ボトムス") {
    return {
      // ボトムスでは胸囲・肩幅は使わないので null
      chest: null,
      shoulder: null,
      waist: size.waist ?? null,
      length: size.length ?? null,
    };
  }

  // デフォルト（トップス想定）
  return {
    chest: size.chest ?? null,
    shoulder: size.shoulder ?? null,
    waist: size.waist ?? null,
    length: size.length ?? null,
  };
}
