//frontend\console\model\src\application\modelUpdateService.tsx
import {
  updateModelVariation as updateModelVariationApi,
  deleteModelVariation as deleteModelVariationApi,
  type ModelVariationResponse,
} from "../infrastructure/api/modelUpdateApi";

export type {
  ModelVariationUpdateRequest,
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
 * ★ 差分ログを分かりやすく出力
 */
export async function deleteRemovedModelVariations(
  listed: ModelVariationResponse[],
  remainingIds: string[],
): Promise<void> {
  const trimmedRemaining = remainingIds.map((id) => id.trim()).filter(Boolean);
  const remainingSet = new Set(trimmedRemaining);

  // 削除対象: list に存在するが remainingIds にない variation
  const removed = listed.filter((v) => v.id && !remainingSet.has(v.id));

  // =======================================================
  // 🔍 差分ログ（非常にわかりやすく）
  // =======================================================
  console.group(
    "%c[ModelUpdateService] ModelVariation 差分チェック",
    "color:#0a84ff; font-weight:bold;"
  );

  console.log("📌 既存(listed) IDs:", listed.map((v) => v.id));
  console.log("📌 残す(remaining) IDs:", trimmedRemaining);

  console.log(
    "%c🗑 削除対象 IDs:",
    "color:#ff3b30; font-weight:bold;",
    removed.map((v) => v.id)
  );

  console.groupEnd();
  // =======================================================

  // DELETE /models/{id} を実行
  for (const v of removed) {
    console.log(
      `%c[ModelUpdateService] DELETE /models/${v.id}`,
      "color:#ff3b30;"
    );
    await deleteModelVariationApi(v.id);
  }
}
