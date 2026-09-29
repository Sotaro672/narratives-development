// frontend/mall/src/features/trade/infrastructure/tradeMessageImageApi.ts

import {
  getDownloadURL,
  ref,
  uploadBytes,
} from "firebase/storage";

import { storage } from "../../../lib/firebase";
import type { TradeMessageImage } from "../../shared/types/trade";

export type UploadTradeMessageImageParams = {
  tradeId: string;
  file: File;
};

function createUploadImageId(file: File): string {
  if (
    typeof crypto !== "undefined" &&
    "randomUUID" in crypto
  ) {
    return crypto.randomUUID();
  }

  return `${file.name}-${file.lastModified}-${Math.random()
    .toString(36)
    .slice(2)}`;
}

function sanitizeStorageFileName(
  fileName: string,
): string {
  const trimmed = fileName.trim();

  if (!trimmed) {
    return "image";
  }

  return trimmed.replace(
    /[^\w.\-()]/g,
    "_",
  );
}

function normalizeTradeId(tradeId: string): string {
  const normalized = tradeId.trim();

  if (!normalized) {
    throw new Error("取引IDが見つかりません。");
  }

  return normalized;
}

function validateImageFile(file: File): void {
  if (!file.type.startsWith("image/")) {
    throw new Error("画像ファイルを選択してください。");
  }

  if (file.size <= 0) {
    throw new Error("空の画像ファイルはアップロードできません。");
  }
}

export async function uploadTradeMessageImage({
  tradeId,
  file,
}: UploadTradeMessageImageParams): Promise<TradeMessageImage> {
  const normalizedTradeId = normalizeTradeId(tradeId);

  validateImageFile(file);

  const imageId = createUploadImageId(file);
  const safeFileName = sanitizeStorageFileName(file.name);
  const objectPath =
    `trade-message-images/${normalizedTradeId}` +
    `/${imageId}/${safeFileName}`;

  const storageRef = ref(
    storage,
    objectPath,
  );

  const mimeType =
    file.type ||
    "application/octet-stream";

  await uploadBytes(
    storageRef,
    file,
    {
      contentType: mimeType,
    },
  );

  const fileUrl = await getDownloadURL(
    storageRef,
  );

  return {
    fileName: file.name,
    fileUrl,
    objectPath,
    fileSize: file.size,
    mimeType,
  };
}