// frontend/console/tokenBlueprint/src/infrastructure/storage/tokenBlueprintAssetStorage.ts
import { getDownloadURL, ref, uploadBytes } from "firebase/storage";

import {
  auth,
  storage,
} from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

export type TokenBlueprintAssetTarget = "tokenBlueprintIcon" | "tokenBlueprintContents";

export type TokenBlueprintContentKind = "image" | "video" | "pdf" | "document";

function getFileExtension(file: File): string {
  const fileName = String(file.name ?? "");
  const parts = fileName.split(".");
  const ext = parts.length > 1 ? parts.pop() : "";

  return ext ? `.${ext}` : "";
}

function safeFileName(file: File, fallback: string): string {
  const raw = String(file.name ?? "").trim();
  if (!raw) return fallback;

  return raw
    .replace(/[\\/:*?"<>|#%{}[\]^~`]/g, "_")
    .replace(/\s+/g, "_");
}

export function guessTokenBlueprintContentType(file: File): TokenBlueprintContentKind {
  const mime = String(file.type || "").toLowerCase();
  if (mime.startsWith("image/")) return "image";
  if (mime.startsWith("video/")) return "video";
  if (mime === "application/pdf") return "pdf";
  return "document";
}

function buildTokenBlueprintIconPath(params: {
  companyId: string;
  tokenBlueprintId: string;
  file: File;
}): string {
  const extension = getFileExtension(params.file);
  const timestamp = Date.now();

  return [
    "token-blueprints",
    params.companyId,
    params.tokenBlueprintId,
    "icon",
    `${timestamp}${extension}`,
  ].join("/");
}

function buildTokenBlueprintContentPath(params: {
  companyId: string;
  tokenBlueprintId: string;
  contentId: string;
  file: File;
}): string {
  const fileName = safeFileName(params.file, `${params.contentId}${getFileExtension(params.file)}`);

  return [
    "token-blueprints",
    params.companyId,
    params.tokenBlueprintId,
    "contents",
    params.contentId,
    fileName,
  ].join("/");
}

async function assertSignedIn(): Promise<void> {
  const user = auth.currentUser;
  if (!user) {
    throw new Error("Firebase Auth user is not signed in.");
  }

  await user.getIdToken();
}

export async function uploadTokenBlueprintIconToFirebaseStorage(params: {
  companyId: string;
  tokenBlueprintId: string;
  file: File;
}): Promise<{
  downloadUrl: string;
  objectPath: string;
}> {
  if (!params.companyId) {
    throw new Error("companyId is required before uploading token blueprint icon.");
  }

  if (!params.tokenBlueprintId) {
    throw new Error("tokenBlueprintId is required before uploading token blueprint icon.");
  }

  if (!params.file) {
    throw new Error("file is required before uploading token blueprint icon.");
  }

  await assertSignedIn();

  const objectPath = buildTokenBlueprintIconPath(params);
  const storageRef = ref(storage, objectPath);

  await uploadBytes(storageRef, params.file, {
    contentType: params.file.type || "application/octet-stream",
  });

  const downloadUrl = await getDownloadURL(storageRef);

  return {
    downloadUrl,
    objectPath,
  };
}

export async function uploadTokenBlueprintContentToFirebaseStorage(params: {
  companyId: string;
  tokenBlueprintId: string;
  contentId: string;
  file: File;
}): Promise<{
  downloadUrl: string;
  objectPath: string;
}> {
  if (!params.companyId) {
    throw new Error("companyId is required before uploading token blueprint content.");
  }

  if (!params.tokenBlueprintId) {
    throw new Error("tokenBlueprintId is required before uploading token blueprint content.");
  }

  if (!params.contentId) {
    throw new Error("contentId is required before uploading token blueprint content.");
  }

  if (!params.file) {
    throw new Error("file is required before uploading token blueprint content.");
  }

  await assertSignedIn();

  const objectPath = buildTokenBlueprintContentPath(params);
  const storageRef = ref(storage, objectPath);

  await uploadBytes(storageRef, params.file, {
    contentType: params.file.type || "application/octet-stream",
  });

  const downloadUrl = await getDownloadURL(storageRef);

  return {
    downloadUrl,
    objectPath,
  };
}