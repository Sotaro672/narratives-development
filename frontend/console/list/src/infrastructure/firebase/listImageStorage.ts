// frontend/console/list/src/infrastructure/firebase/listImageStorage.ts

import { getDownloadURL, ref, uploadBytes } from "firebase/storage";
import { storage } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

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
  const raw = String(file?.name ?? "").trim() || "image";

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
  const listId = String(args.listId ?? "").trim();
  const imageId = String(args.imageId ?? "").trim();
  const name = safeListImageFileName(args.file);

  if (!listId) throw new Error("invalid_list_id");
  if (!imageId) throw new Error("invalid_image_id");

  return `lists/${listId}/images/${imageId}/${name}`;
}

export async function uploadListImageToFirebaseStorage(
  input: UploadListImageToFirebaseStorageInput,
): Promise<UploadedListImageFromFirebaseStorage> {
  const listId = String(input.listId ?? "").trim();
  const imageId = String(input.imageId ?? "").trim() || createListImageId();
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