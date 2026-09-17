// frontend/mall/src/features/avatar/utils/cropAvatarImage.ts

export type AvatarCropPosition = {
  x: number;
  y: number;
};

export type CropAvatarImageParams = {
  file: File;
  position: AvatarCropPosition;
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
      reject(new Error("画像の読み込みに失敗しました。"));
    };

    image.src = objectUrl;
  });
}

function canvasToBlob(
  canvas: HTMLCanvasElement,
  type: string,
  quality?: number,
): Promise<Blob> {
  return new Promise((resolve, reject) => {
    canvas.toBlob(
      (blob) => {
        if (!blob) {
          reject(new Error("画像の切り抜きに失敗しました。"));
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

  if (!trimmed) {
    return "avatar-icon.webp";
  }

  const lastDotIndex = trimmed.lastIndexOf(".");
  const baseName =
    lastDotIndex > 0
      ? trimmed.slice(0, lastDotIndex)
      : trimmed;

  return `${baseName}-cropped.webp`;
}

export async function cropAvatarImage({
  file,
  position,
  scale,
  viewportSize,
  outputSize = DEFAULT_OUTPUT_SIZE,
}: CropAvatarImageParams): Promise<File> {
  if (scale <= 0 || !Number.isFinite(scale)) {
    throw new Error("画像の拡大率が不正です。");
  }

  if (viewportSize <= 0 || !Number.isFinite(viewportSize)) {
    throw new Error("切り抜き領域のサイズが取得できませんでした。");
  }

  if (outputSize <= 0 || !Number.isFinite(outputSize)) {
    throw new Error("出力画像サイズが不正です。");
  }

  const image = await loadImage(file);
  const sourceWidth = image.naturalWidth;
  const sourceHeight = image.naturalHeight;

  if (sourceWidth <= 0 || sourceHeight <= 0) {
    throw new Error("画像サイズを取得できませんでした。");
  }

  /*
   * AvatarIconCropperでは、画像全体が円形viewportを覆うように
   * object-fit: cover相当の倍率をまず計算し、その上からscaleを適用している。
   */
  const coverScale = Math.max(
    viewportSize / sourceWidth,
    viewportSize / sourceHeight,
  );

  const renderedScale = coverScale * scale;
  const renderedWidth = sourceWidth * renderedScale;
  const renderedHeight = sourceHeight * renderedScale;

  /*
   * Cropper上では画像中心をviewport中心へ配置したあと、
   * position.x / position.yだけ画像を移動している。
   */
  const renderedLeft =
    viewportSize / 2 +
    position.x -
    renderedWidth / 2;

  const renderedTop =
    viewportSize / 2 +
    position.y -
    renderedHeight / 2;

  /*
   * viewportの左上(0, 0)と右下(viewportSize, viewportSize)を、
   * 元画像上の座標へ戻す。
   */
  const sourceX = -renderedLeft / renderedScale;
  const sourceY = -renderedTop / renderedScale;
  const sourceCropSize = viewportSize / renderedScale;

  /*
   * 浮動小数点誤差などで画像範囲からわずかにはみ出しても
   * drawImageが不自然にならないように補正する。
   */
  const clampedSourceX = Math.min(
    Math.max(0, sourceX),
    Math.max(0, sourceWidth - sourceCropSize),
  );

  const clampedSourceY = Math.min(
    Math.max(0, sourceY),
    Math.max(0, sourceHeight - sourceCropSize),
  );

  const safeCropSize = Math.min(
    sourceCropSize,
    sourceWidth - clampedSourceX,
    sourceHeight - clampedSourceY,
  );

  if (safeCropSize <= 0) {
    throw new Error("切り抜き可能な画像領域を取得できませんでした。");
  }

  const canvas = document.createElement("canvas");
  canvas.width = Math.round(outputSize);
  canvas.height = Math.round(outputSize);

  const context = canvas.getContext("2d");

  if (!context) {
    throw new Error("画像編集機能を利用できません。");
  }

  context.imageSmoothingEnabled = true;
  context.imageSmoothingQuality = "high";

  context.clearRect(
    0,
    0,
    canvas.width,
    canvas.height,
  );

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

  const blob = await canvasToBlob(
    canvas,
    OUTPUT_MIME_TYPE,
    OUTPUT_QUALITY,
  );

  return new File(
    [blob],
    createCroppedFileName(file.name),
    {
      type: OUTPUT_MIME_TYPE,
      lastModified: Date.now(),
    },
  );
}

export default cropAvatarImage;