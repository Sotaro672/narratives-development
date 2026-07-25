// frontend/console/model/src/application/modelUpdateService.tsx

import {
  updateModelVariation as updateModelVariationApi,
  deleteModelVariation as deleteModelVariationApi,
  type ModelVariationResponse,
} from "../infrastructure/api/modelUpdateApi";

export type {
  ModelVariationKind,
  Volume,
  ApparelModelVariationUpdateRequest,
  AlcoholModelVariationUpdateRequest,
  ModelVariationUpdateRequest,
  ApparelModelVariationResponse,
  AlcoholModelVariationResponse,
  ModelVariationResponse,
} from "../infrastructure/api/modelUpdateApi";

export {
  updateModelVariationApi as updateModelVariation,
  deleteModelVariationApi as deleteModelVariation,
};

/**
 * list 結果と「更新後に残る id」一覧を比較し、
 * 減った差分（= list にはあるが remainingIds には存在しないもの）を物理削除する。
 *
 * NOTE:
 * - ModelVariationResponse は apparel / alcohol の union 型。
 * - 差分削除では kind に依存せず、id のみを正として扱う。
 */
export async function deleteRemovedModelVariations(
  listed: ModelVariationResponse[],
  remainingIds: string[],
): Promise<void> {
  const remainingSet = new Set(
    remainingIds.map((id) => id.trim()).filter(Boolean),
  );

  const removed = listed.filter((variation) => {
    const id = variation.id.trim();
    return id && !remainingSet.has(id);
  });

  for (const variation of removed) {
    await deleteModelVariationApi(variation.id);
  }
}