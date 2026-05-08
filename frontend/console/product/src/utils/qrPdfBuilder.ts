// frontend/console/product/src/utils/qrPdfBuilder.ts

import { PDFDocument, StandardFonts, rgb } from "pdf-lib";
import { generateQrPngDataUrl } from "./qrImageConverter";

/**
 * 1 つの QR に対応する情報
 */
export type QrPdfItem = {
  /** QR に埋め込むペイロード（URL など） */
  payload: string;
  /** QR 下に表示するラベル（productId など） */
  label?: string;
};

/**
 * PDF 生成時のオプション
 */
export type QrPdfOptions = {
  /** タイトル（未使用なら省略可） */
  title?: string;
  /** 横方向の列数（デフォルト 4 列） */
  cols?: number;
  /** 1 セルの高さ（pt） */
  cellHeight?: number;
};

/**
 * QR 一覧を A4 縦・1 行 4 つで並べた PDF を生成し、Blob を返す。
 *
 * - 単位は PDF の pt（1pt ≒ 1/72 inch）
 * - A4: 595.28 x 841.89 pt（縦）
 */
export async function buildQrPdfBlobA4(
  items: QrPdfItem[],
  options?: QrPdfOptions,
): Promise<Blob> {
  const pdfDoc = await PDFDocument.create();
  const page = pdfDoc.addPage([595.28, 841.89]); // A4 縦
  const font = await pdfDoc.embedFont(StandardFonts.Helvetica);

  const cols = options?.cols ?? 4;
  const marginX = 36; // 左右マージン
  const marginY = 36; // 上下マージン
  const cellWidth = (page.getWidth() - marginX * 2) / cols;
  const cellHeight = options?.cellHeight ?? 140;

  let xIndex = 0;
  let yOffset = page.getHeight() - marginY - cellHeight;

  for (const item of items) {
    // 列が埋まったら次の行へ
    if (xIndex >= cols) {
      xIndex = 0;
      yOffset -= cellHeight;

      // ページの下まで来たら新しいページを追加
      if (yOffset < marginY) {
        const newPage = pdfDoc.addPage([595.28, 841.89]);
        yOffset = newPage.getHeight() - marginY - cellHeight;
      }
    }

    const currentPage = pdfDoc.getPages()[pdfDoc.getPageCount() - 1];
    const x = marginX + cellWidth * xIndex;

    // QR PNG DataURL を生成
    const dataUrl = await generateQrPngDataUrl(item.payload, {
      size: 256,
      margin: 1,
    });

    // DataURL → Uint8Array（PNG バイナリ）
    const base64 = dataUrl.split(",")[1] ?? "";
    const pngBytes = Uint8Array.from(
      atob(base64),
      (c) => c.charCodeAt(0),
    );

    const pngImage = await pdfDoc.embedPng(pngBytes);

    const qrSize = Math.min(cellWidth - 10, cellHeight - 30);
    const qrX = x + (cellWidth - qrSize) / 2;
    const qrY = yOffset + 20;

    // QR 画像を描画
    currentPage.drawImage(pngImage, {
      x: qrX,
      y: qrY,
      width: qrSize,
      height: qrSize,
    });

    // ラベルがあれば下に描画
    if (item.label) {
      currentPage.drawText(item.label, {
        x: x + 4,
        y: yOffset + 4,
        size: 8,
        font,
        color: rgb(0, 0, 0),
        maxWidth: cellWidth - 8,
      });
    }

    xIndex += 1;
  }

  // pdf-lib の戻り値: Uint8Array<ArrayBufferLike>
  const pdfBytes = await pdfDoc.save();

  // Uint8Array<ArrayBufferLike> → 純粋な ArrayBuffer に変換して Blob に渡す
  const ab = pdfBytes.buffer.slice(
    pdfBytes.byteOffset,
    pdfBytes.byteOffset + pdfBytes.byteLength,
  );

  // TS 的には ArrayBuffer | SharedArrayBuffer なので、ここで ArrayBuffer に絞る
  const arrayBuffer = ab as ArrayBuffer;

  const blob = new Blob([arrayBuffer], { type: "application/pdf" });
  return blob;
}

/**
 * 生成済み PDF Blob を新しいタブで開くヘルパー
 */
export function openQrPdfInNewTab(blob: Blob): void {
  const url = URL.createObjectURL(blob);
  window.open(url, "_blank", "noopener,noreferrer");
}
