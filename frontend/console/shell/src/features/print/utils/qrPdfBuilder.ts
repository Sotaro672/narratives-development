// frontend/console/product/src/utils/qrPdfBuilder.ts


import { PDFDocument } from "pdf-lib";
import { generateQrPngDataUrl } from "./qrImageConverter";


/**
 * 1 つの QR に対応する情報
 */
export type QrPdfItem = {
  /** QR に埋め込むペイロード（URL など） */
  payload: string;
  /** QR 下に表示するラベル（modelNumber など） */
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
 * 日本語を含むラベル文字列を PNG DataURL に変換する。
 *
 * pdf-lib の標準フォントでは日本語を直接描画できないため、
 * ブラウザの Canvas で文字列を描画し、
 * その結果を画像として PDF に埋め込む。
 *
 * PDF 側で文字画像の縦横比を維持できるよう、
 * Canvas の論理サイズも返す。
 */
function generateLabelPngDataUrl(
  label: string,
): {
  dataUrl: string;
  width: number;
  height: number;
} {
  const canvas = document.createElement("canvas");


  const scale = 4;
  const fontSize = 32;
  const paddingX = 8;
  const paddingY = 6;


  const measureContext = canvas.getContext("2d");


  if (!measureContext) {
    throw new Error("Failed to create canvas context for PDF label");
  }


  measureContext.font =
    `${fontSize}px "Noto Sans JP", "Yu Gothic", "YuGothic", "Meiryo", sans-serif`;


  const measuredWidth =
    Math.ceil(
      measureContext.measureText(label).width,
    );


  const width =
    measuredWidth + paddingX * 2;


  const height =
    fontSize + paddingY * 2;


  canvas.width = width * scale;
  canvas.height = height * scale;


  const drawContext = canvas.getContext("2d");


  if (!drawContext) {
    throw new Error("Failed to create canvas context for PDF label");
  }


  drawContext.scale(
    scale,
    scale,
  );


  drawContext.clearRect(
    0,
    0,
    width,
    height,
  );


  drawContext.fillStyle = "#000000";
  drawContext.font =
    `${fontSize}px "Noto Sans JP", "Yu Gothic", "YuGothic", "Meiryo", sans-serif`;
  drawContext.textAlign = "center";
  drawContext.textBaseline = "middle";


  drawContext.fillText(
    label,
    width / 2,
    height / 2,
  );


  return {
    dataUrl: canvas.toDataURL("image/png"),
    width,
    height,
  };
}


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


    // QR DataURL → Uint8Array（PNG バイナリ）
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


    // ラベルがあれば Canvas で画像化して描画
    if (item.label) {
      const label =
        generateLabelPngDataUrl(item.label);


      const labelBase64 =
        label.dataUrl.split(",")[1] ?? "";


      const labelBytes = Uint8Array.from(
        atob(labelBase64),
        (c) => c.charCodeAt(0),
      );


      const labelImage =
        await pdfDoc.embedPng(labelBytes);


      const maxLabelWidth =
        cellWidth - 8;


      // PDF 上でのラベル文字サイズ。
      // 日本語でも十分読みやすい大きさにする。
      const targetLabelHeight = 20;


      let labelWidth =
        targetLabelHeight *
        (label.width / label.height);


      let labelHeight =
        targetLabelHeight;


      // セル幅を超える場合だけ縮小する。
      // 縦横比は維持する。
      if (labelWidth > maxLabelWidth) {
        const ratio =
          maxLabelWidth / labelWidth;


        labelWidth =
          maxLabelWidth;


        labelHeight *= ratio;
      }


      // ラベルをセル中央に配置する。
      const labelX =
        x +
        (cellWidth - labelWidth) / 2;


      const labelY =
        yOffset;


      currentPage.drawImage(labelImage, {
        x: labelX,
        y: labelY,
        width: labelWidth,
        height: labelHeight,
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


  const blob = new Blob(
    [arrayBuffer],
    {
      type: "application/pdf",
    },
  );


  return blob;
}


/**
 * 生成済み PDF Blob を新しいタブで開くヘルパー
 */
export function openQrPdfInNewTab(blob: Blob): void {
  const url = URL.createObjectURL(blob);
  window.open(url, "_blank", "noopener,noreferrer");
}