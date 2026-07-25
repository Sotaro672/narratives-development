// frontend/console/tokenBlueprint/src/infrastructure/storage/tokenBlueprintAssetStorage.ts
import {
  deleteObject,
  getDownloadURL,
  ref,
  uploadBytes,
} from "firebase/storage";

import {
  auth,
  storage,
} from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

export type TokenBlueprintAssetTarget =
  | "tokenBlueprintIcon"
  | "tokenBlueprintContents";

export type TokenBlueprintContentKind = "image" | "video" | "pdf" | "document";

export type FirebaseStorageUploadResult = {
  downloadUrl: string;
  objectPath: string;
  fileName: string;
  contentType: string;
  size: number;
};

export type FirebaseStorageContentUploadResult =
  FirebaseStorageUploadResult & {
    kind: TokenBlueprintContentKind;
  };

const DEFAULT_CONTENT_TYPE = "application/octet-stream";

function getFileExtension(file: File): string {
  const fileName = String(file.name ?? "");
  const parts = fileName.split(".");
  const ext = parts.length > 1 ? parts.pop() : "";

  return ext ? `.${ext.toLowerCase()}` : "";
}

function safeFileName(file: File, fallback: string): string {
  const raw = String(file.name ?? "").trim();
  if (!raw) return fallback;

  return raw
    .replace(/[\\/:*?"<>|#%{}[\]^~`]/g, "_")
    .replace(/\s+/g, "_")
    .replace(/^_+/, "")
    .replace(/_+$/, "");
}

function getContentType(file: File): string {
  return String(file.type || "").trim() || DEFAULT_CONTENT_TYPE;
}

export function guessTokenBlueprintContentType(
  file: File,
): TokenBlueprintContentKind {
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
  const extension = getFileExtension(params.file);
  const timestamp = Date.now();
  const fallbackName = `icon_${timestamp}${extension}`;
  const fileName = safeFileName(params.file, fallbackName);

  return [
    "token-blueprints",
    params.companyId,
    params.tokenBlueprintId,
    "icon",
    `${timestamp}_${fileName}`,
  ].join("/");
}

function buildTokenBlueprintContentPath(params: {
  companyId: string;
  tokenBlueprintId: string;
  contentId: string;
  file: File;
}): string {
  const extension = getFileExtension(params.file);
  const fallbackName = `${params.contentId}${extension}`;
  const fileName = safeFileName(params.file, fallbackName);

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

function assertUploadRequiredParams(params: {
  companyId: string;
  tokenBlueprintId: string;
  file: File;
  targetLabel: string;
}): void {
  if (!params.companyId?.trim()) {
    throw new Error(
      `companyId is required before uploading ${params.targetLabel}.`,
    );
  }

  if (!params.tokenBlueprintId?.trim()) {
    throw new Error(
      `tokenBlueprintId is required before uploading ${params.targetLabel}.`,
    );
  }

  if (!params.file) {
    throw new Error(`file is required before uploading ${params.targetLabel}.`);
  }
}

export async function uploadTokenBlueprintIconToFirebaseStorage(params: {
  companyId: string;
  tokenBlueprintId: string;
  file: File;
}): Promise<FirebaseStorageUploadResult> {
  assertUploadRequiredParams({
    ...params,
    targetLabel: "token blueprint icon",
  });

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
      originalFileName: params.file.name || "",
    },
  });

  const downloadUrl = await getDownloadURL(storageRef);

  return {
    downloadUrl,
    objectPath,
    fileName: params.file.name || objectPath.split("/").pop() || "icon",
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
  assertUploadRequiredParams({
    ...params,
    targetLabel: "token blueprint content",
  });

  if (!params.contentId?.trim()) {
    throw new Error(
      "contentId is required before uploading token blueprint content.",
    );
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
      originalFileName: params.file.name || "",
    },
  });

  const downloadUrl = await getDownloadURL(storageRef);

  return {
    downloadUrl,
    objectPath,
    fileName:
      params.file.name || objectPath.split("/").pop() || params.contentId,
    contentType,
    size: params.file.size,
    kind,
  };
}

export async function deleteTokenBlueprintAssetFromFirebaseStorage(params: {
  objectPath: string;
}): Promise<void> {
  const objectPath = params.objectPath?.trim();

  if (!objectPath) {
    throw new Error(
      "objectPath is required before deleting Firebase Storage asset.",
    );
  }

  await assertSignedIn();

  await deleteObject(ref(storage, objectPath));
}