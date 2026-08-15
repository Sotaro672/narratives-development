// frontend/console/shell/src/features/tokenBlueprint/infrastructure/storage/tokenBlueprintAssetStorage.ts

import { getDownloadURL, ref, uploadBytes } from "firebase/storage";

import { auth, storage } from "../../../../auth/infrastructure/config/firebaseClient";
import type { ContentType } from "../../../../shared/types/tokenBlueprint";

const DEFAULT_CONTENT_TYPE = "application/octet-stream";

export type FirebaseStorageUploadResult = {
  downloadUrl: string;
  objectPath: string;
  fileName: string;
  contentType: string;
  size: number;
};

export type FirebaseStorageContentUploadResult = FirebaseStorageUploadResult & {
  kind: ContentType;
};

function safeFileName(file: File): string {
  const fileName = file.name
    .trim()
    .replace(/[\\/:*?"<>|#%{}[\]^~`]/g, "_")
    .replace(/\s+/g, "_")
    .replace(/^_+/, "")
    .replace(/_+$/, "");

  if (!fileName) {
    throw new Error("file.name is invalid.");
  }

  return fileName;
}

export function getTokenBlueprintContentType(file: File): string {
  return file.type.trim() || DEFAULT_CONTENT_TYPE;
}

export function guessTokenBlueprintContentType(file: File): ContentType {
  const contentType = getTokenBlueprintContentType(file).toLowerCase();

  if (contentType.startsWith("image/")) return "image";
  if (contentType.startsWith("video/")) return "video";
  if (contentType === "application/pdf") return "pdf";

  return "document";
}

function buildTokenBlueprintIconPath(params: {
  companyId: string;
  tokenBlueprintId: string;
  file: File;
}): string {
  return [
    "token-blueprints",
    params.companyId,
    params.tokenBlueprintId,
    "icon",
    `${Date.now()}_${safeFileName(params.file)}`,
  ].join("/");
}

function buildTokenBlueprintContentPath(params: {
  companyId: string;
  tokenBlueprintId: string;
  contentId: string;
  file: File;
}): string {
  return [
    "token-blueprints",
    params.companyId,
    params.tokenBlueprintId,
    "contents",
    params.contentId,
    safeFileName(params.file),
  ].join("/");
}

async function assertSignedIn(): Promise<void> {
  const user = auth.currentUser;

  if (!user) {
    throw new Error("Firebase Auth user is not signed in.");
  }

  await user.getIdToken();
}

function assertUploadRequiredParams(params: {
  companyId: string;
  tokenBlueprintId: string;
  file: File;
  targetLabel: string;
}): void {
  if (!params.companyId) {
    throw new Error(`companyId is required before uploading ${params.targetLabel}.`);
  }

  if (!params.tokenBlueprintId) {
    throw new Error(`tokenBlueprintId is required before uploading ${params.targetLabel}.`);
  }

  if (!params.file) {
    throw new Error(`file is required before uploading ${params.targetLabel}.`);
  }

  if (!params.file.name) {
    throw new Error(`file.name is required before uploading ${params.targetLabel}.`);
  }
}

export async function uploadTokenBlueprintIconToFirebaseStorage(params: {
  companyId: string;
  tokenBlueprintId: string;
  file: File;
}): Promise<FirebaseStorageUploadResult> {
  assertUploadRequiredParams({ ...params, targetLabel: "token blueprint icon" });
  await assertSignedIn();

  const objectPath = buildTokenBlueprintIconPath(params);
  const storageRef = ref(storage, objectPath);
  const contentType = getTokenBlueprintContentType(params.file);

  await uploadBytes(storageRef, params.file, {
    contentType,
    customMetadata: {
      companyId: params.companyId,
      tokenBlueprintId: params.tokenBlueprintId,
      target: "tokenBlueprintIcon",
      originalFileName: params.file.name,
    },
  });

  const downloadUrl = await getDownloadURL(storageRef);

  return {
    downloadUrl,
    objectPath,
    fileName: params.file.name,
    contentType,
    size: params.file.size,
  };
}

export async function uploadTokenBlueprintContentToFirebaseStorage(params: {
  companyId: string;
  tokenBlueprintId: string;
  contentId: string;
  file: File;
}): Promise<FirebaseStorageContentUploadResult> {
  assertUploadRequiredParams({ ...params, targetLabel: "token blueprint content" });

  if (!params.contentId) {
    throw new Error("contentId is required before uploading token blueprint content.");
  }

  await assertSignedIn();

  const objectPath = buildTokenBlueprintContentPath(params);
  const storageRef = ref(storage, objectPath);
  const contentType = getTokenBlueprintContentType(params.file);
  const kind = guessTokenBlueprintContentType(params.file);

  await uploadBytes(storageRef, params.file, {
    contentType,
    customMetadata: {
      companyId: params.companyId,
      tokenBlueprintId: params.tokenBlueprintId,
      contentId: params.contentId,
      target: "tokenBlueprintContents",
      kind,
      originalFileName: params.file.name,
    },
  });

  const downloadUrl = await getDownloadURL(storageRef);

  return {
    downloadUrl,
    objectPath,
    fileName: params.file.name,
    contentType,
    size: params.file.size,
    kind,
  };
}