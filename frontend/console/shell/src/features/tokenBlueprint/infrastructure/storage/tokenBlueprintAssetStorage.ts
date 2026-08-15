// frontend/console/shell/src/features/tokenBlueprint/infrastructure/storage/tokenBlueprintAssetStorage.ts

import { getDownloadURL, ref, uploadBytes } from "firebase/storage";

import { auth, storage } from "../../../../auth/infrastructure/config/firebaseClient";
import {
  TOKEN_BLUEPRINT_DEFAULT_CONTENT_TYPE,
  type ContentType,
} from "../../../../shared/types/tokenBlueprint";

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

function getContentType(file: File): string {
  return file.type.trim() || TOKEN_BLUEPRINT_DEFAULT_CONTENT_TYPE;
}

export function guessTokenBlueprintContentType(file: File): ContentType {
  const mime = getContentType(file).toLowerCase();

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
  const timestamp = Date.now();
  return [
    "token-blueprints",
    params.companyId,
    params.tokenBlueprintId,
    "icon",
    `${timestamp}_${safeFileName(params.file)}`,
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
  const contentType = getContentType(params.file);

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
  const contentType = getContentType(params.file);
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