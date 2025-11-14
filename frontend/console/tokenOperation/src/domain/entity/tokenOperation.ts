// frontend/tokenOperation/src/presentation/domain/entity/tokenOperation.ts

/**
 * TokenOperation
 * backend/internal/domain/tokenOperation/entity.go の TokenOperation に対応。
 *
 * - camelCase 命名
 * - ID/各種 ID は string
 */
export interface TokenOperation {
  id: string;
  tokenBlueprintId: string;
  assigneeId: string;
}

/**
 * TokenOperationExtended
 * backend/internal/domain/tokenOperation/entity.go の TokenOperationExtended に対応。
 *
 * サーバー側で TokenBlueprint / Brand / Member を JOIN 済みの拡張ビュー。
 */
export interface TokenOperationExtended extends TokenOperation {
  tokenName: string;
  symbol: string;
  brandId: string;
  assigneeName: string;
  brandName: string;
}

/* =========================================================
 * バリデーション / ヘルパ
 * =======================================================*/

/** ID prefix 制約（Go 側の EnforceIDPrefix / TokenOperationIDPrefix と対応） */
export const ENFORCE_ID_PREFIX = false;
export const TOKEN_OPERATION_ID_PREFIX = "";
export const MAX_ID_LENGTH = 128;

export const MAX_NAME_LENGTH = 200;
export const MAX_SYMBOL_LENGTH = 32;

/** TokenOperation の妥当性チェック */
export function validateTokenOperation(op: TokenOperation): string[] {
  const errors: string[] = [];

  const id = op.id?.trim();
  const tokenBlueprintId = op.tokenBlueprintId?.trim();
  const assigneeId = op.assigneeId?.trim();

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

  if (!tokenBlueprintId) {
    errors.push("tokenBlueprintId is required");
  }

  if (!assigneeId) {
    errors.push("assigneeId is required");
  }

  return errors;
}

/** TokenOperationExtended の妥当性チェック */
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
    !!tokenName &&
      MAX_NAME_LENGTH > 0 &&
      [...tokenName].length > MAX_NAME_LENGTH,
    `tokenName length must be <= ${MAX_NAME_LENGTH}`,
  );

  pushIf(!symbol, "symbol is required");
  pushIf(
    !!symbol &&
      MAX_SYMBOL_LENGTH > 0 &&
      [...symbol].length > MAX_SYMBOL_LENGTH,
    `symbol length must be <= ${MAX_SYMBOL_LENGTH}`,
  );

  pushIf(!brandId, "brandId is required");

  pushIf(!assigneeName, "assigneeName is required");
  pushIf(
    !!assigneeName &&
      MAX_NAME_LENGTH > 0 &&
      [...assigneeName].length > MAX_NAME_LENGTH,
    `assigneeName length must be <= ${MAX_NAME_LENGTH}`,
  );

  pushIf(!brandName, "brandName is required");
  pushIf(
    !!brandName &&
      MAX_NAME_LENGTH > 0 &&
      [...brandName].length > MAX_NAME_LENGTH,
    `brandName length must be <= ${MAX_NAME_LENGTH}`,
  );

  return errors;
}

/** 正規化（trim のみ、Go 側 New と整合） */
export function normalizeTokenOperation(
  input: TokenOperation,
): TokenOperation {
  return {
    id: input.id.trim(),
    tokenBlueprintId: input.tokenBlueprintId.trim(),
    assigneeId: input.assigneeId.trim(),
  };
}

/** Extended 正規化ヘルパ */
export function normalizeTokenOperationExtended(
  input: TokenOperationExtended,
): TokenOperationExtended {
  return {
    ...normalizeTokenOperation(input),
    tokenName: input.tokenName.trim(),
    symbol: input.symbol.trim(),
    brandId: input.brandId.trim(),
    assigneeName: input.assigneeName.trim(),
    brandName: input.brandName.trim(),
  };
}
