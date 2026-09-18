// frontend/console/shell/src/shared/util/cropIconImage.ts

import type { IconCropPosition } from "../types/iconCrop";

export type CropIconImageParams = {
  file: File;
  position: IconCropPosition;
  scale: number;
  viewportSize: number;
  outputSize?: number;
};

const DEFAULT_OUTPUT_SIZE = 512;
const OUTPUT_MIME_TYPE = "image/webp";
const OUTPUT_QUALITY = 0.92;

function loadImage(file: File): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const objectUrl = URL.createObjectURL(file);
    const image = new Image();

    image.onload = () => {
      URL.revokeObjectURL(objectUrl);
      resolve(image);
    };

    image.onerror = () => {
      URL.revokeObjectURL(objectUrl);
      reject(new Error("画像の読み込みに失敗しました。別の画像を選択してください。"));
    };

    image.src = objectUrl;
  });
}

function canvasToBlob(canvas: HTMLCanvasElement, type: string, quality?: number): Promise<Blob> {
  return new Promise((resolve, reject) => {
    canvas.toBlob(
      (blob) => {
        if (!blob) {
          reject(new Error("アイコン画像の切り抜きに失敗しました。"));
          return;
        }

        resolve(blob);
      },
      type,
      quality,
    );
  });
}

function createCroppedFileName(fileName: string): string {
  const trimmed = fileName.trim();
  if (!trimmed) return "icon.webp";

  const lastDotIndex = trimmed.lastIndexOf(".");
  const baseName = lastDotIndex > 0 ? trimmed.slice(0, lastDotIndex) : trimmed;

  return `${baseName}-cropped.webp`;
}

export async function cropIconImage({
  file,
  position,
  scale,
  viewportSize,
  outputSize = DEFAULT_OUTPUT_SIZE,
}: CropIconImageParams): Promise<File> {
  if (!file) {
    throw new Error("アイコン画像が選択されていません。");
  }

  if (scale <= 0 || !Number.isFinite(scale)) {
    throw new Error("画像の拡大率が不正です。");
  }

  if (viewportSize <= 0 || !Number.isFinite(viewportSize)) {
    throw new Error("切り抜き領域のサイズを取得できませんでした。画像を選択し直してください。");
  }

  if (outputSize <= 0 || !Number.isFinite(outputSize)) {
    throw new Error("出力画像サイズが不正です。");
  }

  const image = await loadImage(file);
  const sourceWidth = image.naturalWidth;
  const sourceHeight = image.naturalHeight;

  if (sourceWidth <= 0 || sourceHeight <= 0) {
    throw new Error("画像サイズを取得できませんでした。別の画像を選択してください。");
  }

  const coverScale = Math.max(viewportSize / sourceWidth, viewportSize / sourceHeight);
  const renderedScale = coverScale * scale;
  const renderedWidth = sourceWidth * renderedScale;
  const renderedHeight = sourceHeight * renderedScale;

  const renderedLeft = viewportSize / 2 + position.x - renderedWidth / 2;
  const renderedTop = viewportSize / 2 + position.y - renderedHeight / 2;

  const sourceX = -renderedLeft / renderedScale;
  const sourceY = -renderedTop / renderedScale;
  const sourceCropSize = viewportSize / renderedScale;

  const clampedSourceX = Math.min(Math.max(0, sourceX), Math.max(0, sourceWidth - sourceCropSize));
  const clampedSourceY = Math.min(Math.max(0, sourceY), Math.max(0, sourceHeight - sourceCropSize));

  const safeCropSize = Math.min(
    sourceCropSize,
    sourceWidth - clampedSourceX,
    sourceHeight - clampedSourceY,
  );

  if (safeCropSize <= 0 || !Number.isFinite(safeCropSize)) {
    throw new Error("切り抜き可能な画像領域を取得できませんでした。画像を選択し直してください。");
  }

  const canvas = document.createElement("canvas");
  canvas.width = Math.round(outputSize);
  canvas.height = Math.round(outputSize);

  const context = canvas.getContext("2d");

  if (!context) {
    throw new Error("このブラウザでは画像編集機能を利用できません。");
  }

  context.imageSmoothingEnabled = true;
  context.imageSmoothingQuality = "high";
  context.clearRect(0, 0, canvas.width, canvas.height);

  context.drawImage(
    image,
    clampedSourceX,
    clampedSourceY,
    safeCropSize,
    safeCropSize,
    0,
    0,
    canvas.width,
    canvas.height,
  );

  const blob = await canvasToBlob(canvas, OUTPUT_MIME_TYPE, OUTPUT_QUALITY);

  return new File([blob], createCroppedFileName(file.name), {
    type: OUTPUT_MIME_TYPE,
    lastModified: Date.now(),
  });
}

export default cropIconImage;