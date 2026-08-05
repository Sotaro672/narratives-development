// frontend/amol/src/features/scan/utils/extractProductIdFromQr.ts

const PRODUCT_QR_ORIGIN =
  "https://amol.jp";

const PRODUCT_QR_PATH_PATTERN =
  /^\/([A-Za-z0-9_-]+)$/;

/**
 * Product QR:
 *   https://amol.jp/{productId}
 */
export function extractProductIdFromQr(
  rawText: string,
): string | null {
  const trimmed =
    rawText.trim();

  if (!trimmed) {
    return null;
  }

  try {
    const url =
      new URL(trimmed);

    if (
      url.origin !==
        PRODUCT_QR_ORIGIN ||
      url.username !== "" ||
      url.password !== "" ||
      url.search !== "" ||
      url.hash !== ""
    ) {
      return null;
    }

    const pathMatch =
      url.pathname.match(
        PRODUCT_QR_PATH_PATTERN,
      );

    return (
      pathMatch?.[1] ??
      null
    );
  } catch {
    return null;
  }
}