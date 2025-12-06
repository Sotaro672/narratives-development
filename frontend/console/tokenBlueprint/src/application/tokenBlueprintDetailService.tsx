// frontend/console/tokenBlueprint/src/application/tokenBlueprintDetailService.tsx

import type { TokenBlueprint } from "../domain/entity/tokenBlueprint";
import {
  fetchTokenBlueprintById,
  updateTokenBlueprint,
  type UpdateTokenBlueprintPayload,
} from "../infrastructure/repository/tokenBlueprintRepositoryHTTP";

/**
 * 詳細取得（リポジトリのラッパー）
 */
export async function fetchTokenBlueprintDetail(
  id: string,
): Promise<TokenBlueprint> {
  const trimmed = id.trim();
  if (!trimmed) {
    throw new Error("id is empty");
  }
  return fetchTokenBlueprintById(trimmed);
}

/**
 * createdAt を yyyy/mm/dd にフォーマット
 */
export function formatCreatedAt(raw: unknown): string {
  if (!raw) return "";

  let d: Date;
  if (raw instanceof Date) {
    d = raw;
  } else {
    d = new Date(raw as any);
  }

  if (isNaN(d.getTime())) return "";

  const yyyy = d.getFullYear();
  const mm = String(d.getMonth() + 1).padStart(2, "0");
  const dd = String(d.getDate()).padStart(2, "0");
  return `${yyyy}/${mm}/${dd}`;
}

/**
 * TokenBlueprintCard の VM から UpdateTokenBlueprintPayload を組み立てる
 */
export function buildUpdatePayloadFromCardVm(
  blueprint: TokenBlueprint,
  cardVm: any,
): UpdateTokenBlueprintPayload {
  const vmAny: any = cardVm || {};
  const fields: any = vmAny.fields ?? vmAny ?? {};

  const trimOrUndefined = (v: unknown): string | undefined =>
    typeof v === "string" ? v.trim() : undefined;

  const payload: UpdateTokenBlueprintPayload = {
    name: trimOrUndefined(fields.name ?? blueprint.name),
    symbol: trimOrUndefined(fields.symbol ?? blueprint.symbol),
    brandId: trimOrUndefined(fields.brandId ?? blueprint.brandId),
    description: trimOrUndefined(
      fields.description ?? blueprint.description,
    ),
    assigneeId: trimOrUndefined(
      fields.assigneeId ?? blueprint.assigneeId,
    ),
    iconId:
      typeof fields.iconId === "string"
        ? fields.iconId
        : (blueprint as any)?.iconId ?? null,
    contentFiles:
      (fields.contentFiles as string[] | undefined) ??
      blueprint.contentFiles ??
      [],
  };

  return payload;
}

/**
 * TokenBlueprintCard の VM から update API を呼び出し、更新後の TokenBlueprint を返す
 */
export async function updateTokenBlueprintFromCard(
  blueprint: TokenBlueprint,
  cardVm: any,
): Promise<TokenBlueprint> {
  const payload = buildUpdatePayloadFromCardVm(blueprint, cardVm);

  // デバッグ用: 更新リクエストペイロードを確認
  // eslint-disable-next-line no-console
  console.log(
    "[tokenBlueprintDetailService.updateTokenBlueprintFromCard] request payload:",
    { id: blueprint.id, payload },
  );

  const updated = await updateTokenBlueprint(blueprint.id, payload);
  return updated;
}
