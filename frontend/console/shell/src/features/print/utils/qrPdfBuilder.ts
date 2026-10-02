// frontend/console/shell/src/features/print/utils/qrPdfBuilder.ts

import {
  PDFDocument,
  type PDFImage,
  type PDFPage,
} from "pdf-lib";
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

export type QrPdfProgress =
  | {
      stage: "generating";
      completed: number;
      total: number;
    }
  | {
      stage: "saving";
      completed: number;
      total: number;
    }
  | {
      stage: "completed";
      completed: number;
      total: number;
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
  /** QR / PDF 生成進捗 */
  onProgress?: (progress: QrPdfProgress) => void;
};

type LabelImage = {
  image: PDFImage;
  width: number;
  height: number;
};

const A4_WIDTH = 595.28;
const A4_HEIGHT = 841.89;

function dataUrlToUint8Array(
  dataUrl: string,
  errorMessage: string,
): Uint8Array {
  const base64 = dataUrl.split(",")[1] ?? "";

  if (!base64) {
    throw new Error(errorMessage);
  }

  return Uint8Array.from(
    atob(base64),
    (c) => c.charCodeAt(0),
  );
}

/**
 * 日本語を含むラベル文字列を PNG DataURL に変換する。
 *
 * pdf-lib の標準フォントでは日本語を直接描画できないため、
 * ブラウザの Canvas で文字列を描画し、その結果を画像として PDF に埋め込む。
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

  measureContext.font = `${fontSize}px "Noto Sans JP", "Yu Gothic", "YuGothic", "Meiryo", sans-serif`;

  const measuredWidth = Math.ceil(
    measureContext.measureText(label).width,
  );

  const width = measuredWidth + paddingX * 2;
  const height = fontSize + paddingY * 2;

  canvas.width = width * scale;
  canvas.height = height * scale;

  const drawContext = canvas.getContext("2d");

  if (!drawContext) {
    throw new Error("Failed to create canvas context for PDF label");
  }

  drawContext.scale(scale, scale);
  drawContext.clearRect(0, 0, width, height);

  drawContext.fillStyle = "#000000";
  drawContext.font = `${fontSize}px "Noto Sans JP", "Yu Gothic", "YuGothic", "Meiryo", sans-serif`;
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

async function getOrCreateLabelImage(
  pdfDoc: PDFDocument,
  cache: Map<string, LabelImage>,
  label: string,
): Promise<LabelImage> {
  const cached = cache.get(label);

  if (cached) {
    return cached;
  }

  const generated = generateLabelPngDataUrl(label);

  const labelBytes = dataUrlToUint8Array(
    generated.dataUrl,
    "Failed to generate PDF label PNG data",
  );

  const image = await pdfDoc.embedPng(labelBytes);

  const result: LabelImage = {
    image,
    width: generated.width,
    height: generated.height,
  };

  cache.set(label, result);

  return result;
}

/**
 * QR 一覧を A4 縦で並べた PDF を生成し、Blob を返す。
 *
 * - 単位は PDF の pt（1pt ≒ 1/72 inch）
 * - A4: 595.28 x 841.89 pt（縦）
 * - この関数は PDF の生成のみを担当し、画面遷移や window.open は行わない。
 */
export async function buildQrPdfBlobA4(
  items: QrPdfItem[],
  options?: QrPdfOptions,
): Promise<Blob> {
  if (items.length === 0) {
    throw new Error("QR PDF items are empty");
  }

  const cols = Math.max(
    1,
    Math.floor(options?.cols ?? 4),
  );

  const cellHeight = options?.cellHeight ?? 140;

  if (!Number.isFinite(cellHeight) || cellHeight <= 0) {
    throw new Error("Invalid QR PDF cell height");
  }

  const pdfDoc = await PDFDocument.create();

  let currentPage: PDFPage = pdfDoc.addPage([
    A4_WIDTH,
    A4_HEIGHT,
  ]);

  const marginX = 36;
  const marginY = 36;
  const cellWidth = (A4_WIDTH - marginX * 2) / cols;

  let xIndex = 0;
  let yOffset = A4_HEIGHT - marginY - cellHeight;

  const labelImageCache = new Map<string, LabelImage>();
  const total = items.length;

  options?.onProgress?.({
    stage: "generating",
    completed: 0,
    total,
  });

  for (let index = 0; index < items.length; index += 1) {
    const item = items[index];

    if (xIndex >= cols) {
      xIndex = 0;
      yOffset -= cellHeight;

      if (yOffset < marginY) {
        currentPage = pdfDoc.addPage([
          A4_WIDTH,
          A4_HEIGHT,
        ]);

        yOffset = A4_HEIGHT - marginY - cellHeight;
      }
    }

    const x = marginX + cellWidth * xIndex;

    const dataUrl = await generateQrPngDataUrl(
      item.payload,
      {
        size: 256,
        margin: 1,
      },
    );

    const pngBytes = dataUrlToUint8Array(
      dataUrl,
      `Failed to generate QR PNG data at item ${index + 1}`,
    );

    const pngImage = await pdfDoc.embedPng(pngBytes);

    const qrSize = Math.min(
      cellWidth - 10,
      cellHeight - 30,
    );

    if (qrSize <= 0) {
      throw new Error("QR PDF cell is too small");
    }

    const qrX = x + (cellWidth - qrSize) / 2;
    const qrY = yOffset + 20;

    currentPage.drawImage(pngImage, {
      x: qrX,
      y: qrY,
      width: qrSize,
      height: qrSize,
    });

    const labelText = item.label?.trim();

    if (labelText) {
      const label = await getOrCreateLabelImage(
        pdfDoc,
        labelImageCache,
        labelText,
      );

      const maxLabelWidth = cellWidth - 8;
      const targetLabelHeight = 20;

      let labelWidth =
        targetLabelHeight *
        (label.width / label.height);

      let labelHeight = targetLabelHeight;

      if (labelWidth > maxLabelWidth) {
        const ratio =
          maxLabelWidth / labelWidth;

        labelWidth = maxLabelWidth;
        labelHeight *= ratio;
      }

      const labelX =
        x +
        (cellWidth - labelWidth) / 2;

      currentPage.drawImage(label.image, {
        x: labelX,
        y: yOffset,
        width: labelWidth,
        height: labelHeight,
      });
    }

    xIndex += 1;

    options?.onProgress?.({
      stage: "generating",
      completed: index + 1,
      total,
    });
  }

  options?.onProgress?.({
    stage: "saving",
    completed: total,
    total,
  });

  const pdfBytes = await pdfDoc.save({
    useObjectStreams: false,
  });

  const ab = pdfBytes.buffer.slice(
    pdfBytes.byteOffset,
    pdfBytes.byteOffset + pdfBytes.byteLength,
  );

  const arrayBuffer = ab as ArrayBuffer;

  const blob = new Blob(
    [arrayBuffer],
    {
      type: "application/pdf",
    },
  );

  options?.onProgress?.({
    stage: "completed",
    completed: total,
    total,
  });

  return blob;
}