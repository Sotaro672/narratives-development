// frontend/console/shell/src/shared/types/productBlueprint.ts

/**
 * 商品へ付与する識別タグの種類。
 *
 * backend/internal/domain/productBlueprint/entity.goの
 * ProductIDTagTypeに対応する。
 */
export type ProductIDTagType =
  | "qr"
  | "nfc";

/**
 * 商品へ付与する識別タグ。
 *
 * backend/internal/domain/productBlueprint/entity.goの
 * ProductIDTagに対応する。
 */
export type ProductIDTag = {
  type: ProductIDTagType;
};

/**
 * ProductIDTagTypeとして有効な値か判定する。
 */
export function isValidProductIDTagType(
  value: string,
): value is ProductIDTagType {
  return (
    value === "qr" ||
    value === "nfc"
  );
}