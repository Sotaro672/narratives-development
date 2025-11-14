// frontend/shell/src/shared/types/tokenOperation.ts

/**
 * 共通で利用する TokenOperation (トークン運用) の型定義。
 * backend/internal/domain/tokenOperation/entity.go の構造を Mirror。
 */

/**
 * TokenOperation
 * - トークン設計 (TokenBlueprint) と担当者 (Assignee) の紐づけを表す。
 * - ID・tokenBlueprintId・assigneeId は全て必須。
 */
export interface TokenOperation {
  id: string;
  tokenBlueprintId: string;
  assigneeId: string;
}

/**
 * TokenOperationExtended
 * - サーバー側で JOIN 済みの拡張構造。
 * - TokenOperation に加えて、ブランド・担当者・トークン名などの情報を含む。
 */
export interface TokenOperationExtended extends TokenOperation {
  tokenName: string;
  symbol: string;
  brandId: string;
  assigneeName: string;
  brandName: string;
}

/* =========================================================
 * バリデーション / 定数
 * =======================================================*/

/** 制約ポリシー（Go 側と同等の値） */
export const ENFORCE_ID_PREFIX = false;
export const TOKEN_OPERATION_ID_PREFIX = "";
export const MAX_ID_LENGTH = 128;

export const MAX_NAME_LENGTH = 200;
export const MAX_SYMBOL_LENGTH = 32;

/**
 * TokenOperation の簡易バリデーション。
 * 不正な場合はエラーメッセージ配列を返す。
 */
export function validateTokenOperation(op: TokenOperation): string[] {
  const errors: string[] = [];
  const id = op.id?.trim();
  const blueprint = op.tokenBlueprintId?.trim();
  const assignee = op.assigneeId?.trim();

  if (!id) {
    errors.push("id is required");
  } else {
    if (ENFORCE_ID_PREFIX && TOKEN_OPERATION_ID_PREFIX) {
      if (!id.startsWith(TOKEN_OPERATION_ID_PREFIX)) {
        errors.push("id must start with TOKEN_OPERATION_ID_PREFIX");
      }
    }
    if (MAX_ID_LENGTH > 0 && id.length > MAX_ID_LENGTH) {
      errors.push(`id length must be <= ${MAX_ID_LENGTH}`);
    }
  }

  if (!blueprint) errors.push("tokenBlueprintId is required");
  if (!assignee) errors.push("assigneeId is required");

  return errors;
}

/**
 * TokenOperationExtended の簡易バリデーション。
 * 基本構造 + 追加フィールドの整合性を確認。
 */
export function validateTokenOperationExtended(
  op: TokenOperationExtended,
): string[] {
  const errors = validateTokenOperation(op);

  const pushIf = (cond: boolean, msg: string) => {
    if (cond) errors.push(msg);
  };

  const tokenName = op.tokenName?.trim();
  const symbol = op.symbol?.trim();
  const brandId = op.brandId?.trim();
  const assigneeName = op.assigneeName?.trim();
  const brandName = op.brandName?.trim();

  pushIf(!tokenName, "tokenName is required");
  pushIf(
    !!tokenName && MAX_NAME_LENGTH > 0 && tokenName.length > MAX_NAME_LENGTH,
    `tokenName length must be <= ${MAX_NAME_LENGTH}`,
  );

  pushIf(!symbol, "symbol is required");
  pushIf(
    !!symbol && MAX_SYMBOL_LENGTH > 0 && symbol.length > MAX_SYMBOL_LENGTH,
    `symbol length must be <= ${MAX_SYMBOL_LENGTH}`,
  );

  pushIf(!brandId, "brandId is required");

  pushIf(!assigneeName, "assigneeName is required");
  pushIf(
    !!assigneeName &&
      MAX_NAME_LENGTH > 0 &&
      assigneeName.length > MAX_NAME_LENGTH,
    `assigneeName length must be <= ${MAX_NAME_LENGTH}`,
  );

  pushIf(!brandName, "brandName is required");
  pushIf(
    !!brandName &&
      MAX_NAME_LENGTH > 0 &&
      brandName.length > MAX_NAME_LENGTH,
    `brandName length must be <= ${MAX_NAME_LENGTH}`,
  );

  return errors;
}

/**
 * TokenOperation の正規化（trim のみ）
 */
export function normalizeTokenOperation(op: TokenOperation): TokenOperation {
  return {
    id: op.id.trim(),
    tokenBlueprintId: op.tokenBlueprintId.trim(),
    assigneeId: op.assigneeId.trim(),
  };
}

/**
 * TokenOperationExtended の正規化（trim のみ）
 */
export function normalizeTokenOperationExtended(
  op: TokenOperationExtended,
): TokenOperationExtended {
  return {
    ...normalizeTokenOperation(op),
    tokenName: op.tokenName.trim(),
    symbol: op.symbol.trim(),
    brandId: op.brandId.trim(),
    assigneeName: op.assigneeName.trim(),
    brandName: op.brandName.trim(),
  };
}
