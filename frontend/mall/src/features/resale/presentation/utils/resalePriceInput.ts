// frontend/amol/src/features/resale/presentation/utils/resalePriceInput.ts

/**
 * 販売価格入力から半角数字以外を除去する。
 *
 * stateにはカンマを含まない数字文字列を保持する。
 */
export function normalizeResalePriceInput(
  value: string,
): string {
  return value.replace(
    /[^\d]/g,
    "",
  );
}

/**
 * 販売価格入力を3桁区切りの表示文字列へ変換する。
 *
 * Numberへ変換せず文字列のまま処理するため、
 * 大きな値でも表示時に丸められない。
 */
export function formatResalePriceInput(
  value: string,
): string {
  const digits =
    normalizeResalePriceInput(
      value,
    );

  if (!digits) {
    return "";
  }

  const normalizedDigits =
    digits.replace(
      /^0+(?=\d)/,
      "",
    );

  return normalizedDigits.replace(
    /\B(?=(\d{3})+(?!\d))/g,
    ",",
  );
}

/**
 * 販売価格が入力されているか判定する。
 */
export function hasResalePriceInput(
  value: string,
): boolean {
  return (
    normalizeResalePriceInput(
      value,
    ).length > 0
  );
}

/**
 * 販売価格入力をAPI送信用のnumberへ変換する。
 *
 * 未入力、負数、整数でない値、JavaScriptで安全に扱えない
 * 大きさの場合はnullを返す。
 */
export function parseResalePriceInput(
  value: string,
): number | null {
  const digits =
    normalizeResalePriceInput(
      value,
    );

  if (!digits) {
    return null;
  }

  const price =
    Number(digits);

  if (
    !Number.isSafeInteger(
      price,
    ) ||
    price < 0
  ) {
    return null;
  }

  return price;
}