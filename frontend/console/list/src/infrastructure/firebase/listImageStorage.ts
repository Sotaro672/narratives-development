// frontend/console/list/src/infrastructure/firebase/listImageStorage.ts

import { getDownloadURL, ref, uploadBytes } from "firebase/storage";
import { storage } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

function toText(v: unknown): string {
  if (v === null || v === undefined) return "";
  return typeof v === "string" ? v.trim() : String(v).trim();
}

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

export function createListImageId(): string {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }

  return `${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

export function safeListImageFileName(file: File): string {
  const raw = toText(file?.name) || "image";

  return raw
    .replace(/[\\/:*?"<>|#%{}^~[\]`]/g, "_")
    .replace(/\s+/g, "_")
    .slice(0, 160);
}

export function buildListImageObjectPath(args: {
  listId: string;
  imageId: string;
  file: File;
}): string {
  const listId = toText(args.listId);
  const imageId = toText(args.imageId);
  const name = safeListImageFileName(args.file);

  if (!listId) throw new Error("invalid_list_id");
  if (!imageId) throw new Error("invalid_image_id");

  return `lists/${listId}/images/${imageId}/${name}`;
}

export async function uploadListImageToFirebaseStorage(
  input: UploadListImageToFirebaseStorageInput,
): Promise<UploadedListImageFromFirebaseStorage> {
  const listId = toText(input.listId);
  const imageId = toText(input.imageId) || createListImageId();
  const file = input.file;

  if (!listId) throw new Error("invalid_list_id");
  if (!imageId) throw new Error("invalid_image_id");
  if (!file) throw new Error("invalid_file");

  const objectPath = buildListImageObjectPath({
    listId,
    imageId,
    file,
  });

  const storageRef = ref(storage, objectPath);

  const snapshot = await uploadBytes(storageRef, file, {
    contentType: file.type || "application/octet-stream",
  });

  const url = await getDownloadURL(snapshot.ref);

  if (!url) {
    throw new Error("firebase_storage_download_url_empty");
  }

  return {
    imageId,
    objectPath,
    url,
  };
}