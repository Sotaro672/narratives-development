// frontend/console/product/src/utils/qrImageConverter.ts

import QRCode from "qrcode";

/**
 * QR コード生成オプション
 */
export type QrImageOptions = {
  /**
   * 画像サイズ（px）
   * デフォルト: 256px
   */
  size?: number;

  /**
   * QR コードの余白（白枠）
   * デフォルト: 2px
   */
  margin?: number;

  /**
   * color.dark: QR の線の色
   * color.light: 背景色
   */
  color?: {
    dark?: string;
    light?: string;
  };
};

/**
 * QR ペイロード（文字列）から PNG Base64(DataURL) を生成する関数。
 *
 * @param payload URL や productId、任意の文字列
 * @param options QR コードのオプション
 * @returns data:image/png;base64,... の DataURL
 */
export async function generateQrPngDataUrl(
  payload: string,
  options?: QrImageOptions,
): Promise<string> {
  const size = options?.size ?? 256;
  const margin = options?.margin ?? 2;
  const dark = options?.color?.dark ?? "#000000";
  const light = options?.color?.light ?? "#ffffff";

  return QRCode.toDataURL(payload, {
    width: size,
    margin,
    color: { dark, light },
  });
}

/**
 * PNG DataURL → Blob 変換
 *
 * @param dataUrl data:image/png;base64,...
 */
export async function dataUrlToBlob(dataUrl: string): Promise<Blob> {
  const res = await fetch(dataUrl);
  return res.blob();
}

/**
 * PNG DataURL を直接ダウンロードさせるユーティリティ
 *
 * @param dataUrl
 * @param filename
 */
export function downloadPngDataUrl(dataUrl: string, filename: string): void {
  const a = document.createElement("a");
  a.href = dataUrl;
  a.download = filename;
  a.style.display = "none";

  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
}
