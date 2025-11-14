// frontend/operation/mockdata.tsx
// frontend/shell/src/shared/types/tokenOperation.ts を正として定義を統一

import type {
  TokenOperationExtended,
} from "../../../../shell/src/shared/types/tokenOperation";

/**
 * TokenOperationExtended モックデータ
 * - backend/internal/domain/tokenOperation/entity.go
 * - frontend/shell/src/shared/types/tokenOperation.ts
 * を Mirror した形で、画面表示に必要な情報を保持。
 *
 * ※ 旧モックで使用していた linkedProducts / planned / requested / issued /
 *    distributionRate などは TokenOperation ドメイン外のため削除。
 *    必要であれば別ドメイン or ViewModel 側で拡張してください。
 */

export const TOKEN_OPERATION_EXTENDED: TokenOperationExtended[] = [
  {
    id: "token_operation_001",
    tokenBlueprintId: "token_blueprint_003", // LUMINA VIP Token
    assigneeId: "member_001",
    tokenName: "LUMINA VIP Token",
    symbol: "LVIP",
    brandId: "brand_lumina",
    assigneeName: "member_001", // 表示名を使う場合は View 側で解決
    brandName: "LUMINA Fashion",
  },
  {
    id: "token_operation_002",
    tokenBlueprintId: "token_blueprint_001", // SILK Premium Token
    assigneeId: "member_002",
    tokenName: "SILK Premium Token",
    symbol: "SILK",
    brandId: "brand_lumina",
    assigneeName: "member_002",
    brandName: "LUMINA Fashion",
  },
  {
    id: "token_operation_003",
    tokenBlueprintId: "token_blueprint_004", // NEXUS Community Token
    assigneeId: "member_001",
    tokenName: "NEXUS Community Token",
    symbol: "NXCOM",
    brandId: "brand_nexus",
    assigneeName: "member_001",
    brandName: "NEXUS Street",
  },
  {
    id: "token_operation_004",
    tokenBlueprintId: "token_blueprint_002", // NEXUS Street Token
    assigneeId: "member_003",
    tokenName: "NEXUS Street Token",
    symbol: "NEXUS",
    brandId: "brand_nexus",
    assigneeName: "member_003",
    brandName: "NEXUS Street",
  },
  {
    id: "token_operation_005",
    tokenBlueprintId: "token_blueprint_005", // SILK Limited Edition
    assigneeId: "member_003",
    tokenName: "SILK Limited Edition",
    symbol: "SLKED",
    brandId: "brand_lumina",
    assigneeName: "member_003",
    brandName: "LUMINA Fashion",
  },
];

/**
 * ID から拡張トークン運用情報を取得
 */
export function findTokenOperationExtendedById(
  id: string,
): TokenOperationExtended | undefined {
  return TOKEN_OPERATION_EXTENDED.find((op) => op.id === id);
}
