// frontend/console/list/src/infrastructure/firebase/listImageStorage.ts
import {
  deleteObject,
  getDownloadURL,
  ref,
  uploadBytes,
} from "firebase/storage";
import { storage } from "../../../shell/src/auth/infrastructure/config/firebaseClient";
export type UploadListImageToFirebaseStorageInput = {
  listId: string;
  file: File;
  imageId?: string;
};
export type UploadedListImageFromFirebaseStorage = {
  imageId: string;
  objectPath: string;
  url: string;
};
export type DeleteListImageFromFirebaseStorageInput = {
  storagePath: string;
};
export function createListImageId(): string {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2)}`;
}
export function safeListImageFileName(file: File): string {
  const raw = String(file?.name ?? "").trim() || "image";
  const normalized = raw
    .replace(/[\\/:*?"<>|#%{}^~[\]`]/g, "_")
    .replace(/\s+/g, "_")
    .replace(/^\.+/, "")
    .slice(0, 160);
  return normalized || "image";
}
export function buildListImageObjectPath(args: {
  listId: string;
  imageId: string;
  file: File;
}): string {
  const listId = normalizeListImagePathSegment(
    args.listId,
    "invalid_list_id",
  );
  const imageId = normalizeListImagePathSegment(
    args.imageId,
    "invalid_image_id",
  );
  const name = safeListImageFileName(args.file);
  return `lists/${listId}/images/${imageId}/${name}`;
}
export async function uploadListImageToFirebaseStorage(
  input: UploadListImageToFirebaseStorageInput,
): Promise<UploadedListImageFromFirebaseStorage> {
  const listId = normalizeListImagePathSegment(
    input.listId,
    "invalid_list_id",
  );
  const imageId = normalizeListImagePathSegment(
    String(input.imageId ?? "").trim() || createListImageId(),
    "invalid_image_id",
  );
  const file = input.file;
  if (!file) {
    throw new Error("invalid_file");
  }
  const objectPath = buildListImageObjectPath({
    listId,
    imageId,
    file,
  });
  const storageRef = ref(storage, objectPath);
  const snapshot = await uploadBytes(storageRef, file, {
    contentType: file.type || "application/octet-stream",
  });
  const url = String(await getDownloadURL(snapshot.ref)).trim();
  if (!url) {
    try {
      await deleteObject(snapshot.ref);
    } catch {
      // Download URL取得失敗を優先して返す。
    }
    throw new Error("firebase_storage_download_url_empty");
  }
  return {
    imageId,
    objectPath,
    url,
  };
}
export async function deleteListImageFromFirebaseStorage(
  input: DeleteListImageFromFirebaseStorageInput,
): Promise<void> {
  const storagePath = normalizeListImageObjectPath(input?.storagePath);
  try {
    await deleteObject(ref(storage, storagePath));
  } catch (error) {
    if (isFirebaseStorageObjectNotFound(error)) {
      return;
    }
    throw error;
  }
}
function normalizeListImageObjectPath(value: unknown): string {
  const storagePath = String(value ?? "").trim().replace(/^\/+/, "");
  if (!storagePath) {
    throw new Error("invalid_storage_path");
  }
  const lowerPath = storagePath.toLowerCase();
  if (
    lowerPath.startsWith("gs://") ||
    lowerPath.startsWith("http://") ||
    lowerPath.startsWith("https://") ||
    /[\r\n\u0000]/u.test(storagePath)
  ) {
    throw new Error("invalid_storage_path");
  }
  const parts = storagePath.split("/");
  if (
    parts.length !== 5 ||
    parts[0] !== "lists" ||
    parts[2] !== "images"
  ) {
    throw new Error("invalid_storage_path");
  }
  normalizeListImagePathSegment(parts[1], "invalid_storage_path_list_id");
  normalizeListImagePathSegment(parts[3], "invalid_storage_path_image_id");
  const fileName = String(parts[4] ?? "").trim();
  if (
    !fileName ||
    fileName === "." ||
    fileName === ".." ||
    fileName.includes("\\")
  ) {
    throw new Error("invalid_storage_path_file_name");
  }
  return storagePath;
}
function normalizeListImagePathSegment(
  value: unknown,
  errorCode: string,
): string {
  const normalized = String(value ?? "").trim();
  if (
    !normalized ||
    normalized === "." ||
    normalized === ".." ||
    normalized.includes("/") ||
    normalized.includes("\\") ||
    normalized.includes("://") ||
    /[\r\n\u0000]/u.test(normalized)
  ) {
    throw new Error(errorCode);
  }
  return normalized;
}
function isFirebaseStorageObjectNotFound(error: unknown): boolean {
  if (!error || typeof error !== "object") {
    return false;
  }
  const code = String(
    (error as { code?: unknown }).code ?? "",
  ).trim();
  return code === "storage/object-not-found";
}