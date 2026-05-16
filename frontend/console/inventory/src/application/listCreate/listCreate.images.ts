// frontend/console/inventory/src/application/listCreate/listCreate.images.ts

import { getDownloadURL, ref, uploadBytes } from "firebase/storage";

import {
  auth,
  storage,
} from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

import {
  saveListImageFromFirebaseStorageHTTP,
  setListPrimaryImageHTTP,
} from "../../../../list/src/infrastructure/repository";

import type { ListDTO } from "../../../../list/src/infrastructure/dto";

export function dedupeFiles(prev: File[], add: File[]): File[] {
  const exists = new Set(
    prev.map((f) => `${f.name}__${f.size}__${f.lastModified}`),
  );

  const filtered = add.filter(
    (f) => !exists.has(`${f.name}__${f.size}__${f.lastModified}`),
  );

  return [...prev, ...filtered];
}

function getListIdFromListDTO(dto: ListDTO, fallback = ""): string {
  return (dto as any).id || fallback;
}

function createImageId(): string {
  if (
    typeof crypto !== "undefined" &&
    typeof crypto.randomUUID === "function"
  ) {
    return crypto.randomUUID();
  }

  return `${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

function safeFileName(file: File): string {
  return file.name
    .replace(/[\\/:*?"<>|#%{}^~[\]`]/g, "_")
    .replace(/\s+/g, "_")
    .slice(0, 160);
}

function buildListImageObjectPath(args: {
  listId: string;
  imageId: string;
  file: File;
}): string {
  const listId = args.listId;
  const imageId = args.imageId;
  const name = safeFileName(args.file);

  if (!listId) throw new Error("invalid_list_id");
  if (!imageId) throw new Error("invalid_image_id");

  return `lists/${listId}/images/${imageId}/${name}`;
}

async function uploadFileToFirebaseStorage(args: {
  listId: string;
  file: File;
  imageId: string;
}): Promise<{
  imageId: string;
  objectPath: string;
  downloadURL: string;
}> {
  const listId = args.listId;
  const imageId = args.imageId;
  const file = args.file;

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

  const downloadURL = await getDownloadURL(snapshot.ref);

  if (!downloadURL) {
    throw new Error("firebase_storage_download_url_empty");
  }

  return {
    imageId,
    objectPath,
    downloadURL,
  };
}

/**
 * 複数画像を Firebase Storage へ直接アップロード
 * → backend にメタ情報登録
 * → primary image 設定
 *
 * 旧方式:
 * - POST /lists/{listId}/images/signed-url
 * - signedUrl へ PUT
 * - saveListImageFromGCSHTTP で bucket/objectPath 登録
 *
 * 新方式:
 * - frontend から Firebase Storage へ uploadBytes
 * - getDownloadURL で downloadURL 取得
 * - saveListImageFromFirebaseStorageHTTP で downloadURL/objectPath 登録
 *
 * primary:
 * - backend の List.ImageID は images subcollection docID
 * - objectPath ではなく imageId を渡す
 */
export async function uploadListImagesPolicyB(args: {
  listId: string;
  files: File[];
  mainImageIndex: number;
  createdBy?: string;
}): Promise<{
  registered: Array<{ imageId: string; displayOrder: number }>;
  primaryImageId?: string;
}> {
  const listId = args.listId;
  const files = Array.isArray(args.files) ? args.files : [];
  const mainImageIndex = Number.isFinite(Number(args.mainImageIndex))
    ? Number(args.mainImageIndex)
    : 0;

  console.log(
    "[debug] uploadListImagesPolicyB.files",
    files.map((f) => ({
      name: f.name,
      size: f.size,
      lastModified: f.lastModified,
      type: f.type,
    })),
  );

  if (!listId) throw new Error("invalid_list_id");
  if (files.length === 0) return { registered: [] };

  if (!files[mainImageIndex]) {
    throw new Error("メイン画像が選択されていません。");
  }

  const uid = args.createdBy || auth.currentUser?.uid || "system";
  const now = new Date().toISOString();

  const registered: Array<{ imageId: string; displayOrder: number }> = [];

  for (let i = 0; i < files.length; i++) {
    const file = files[i];
    if (!file) continue;

    const imageId = createImageId();

    const uploaded = await uploadFileToFirebaseStorage({
      listId,
      file,
      imageId,
    });

    await saveListImageFromFirebaseStorageHTTP({
      listId,
      id: uploaded.imageId,
      url: uploaded.downloadURL,
      objectPath: uploaded.objectPath,
      size: Number(file.size || 0),
      displayOrder: i,
      fileName: file.name,
      contentType: file.type || "application/octet-stream",
      createdBy: uid,
      createdAt: now,
    });

    registered.push({
      imageId: uploaded.imageId,
      displayOrder: i,
    });
  }

  const primary =
    registered.find((x) => x.displayOrder === mainImageIndex) || registered[0];

  if (primary?.imageId) {
    await setListPrimaryImageHTTP({
      listId,
      imageId: primary.imageId,
      updatedBy: uid,
      now,
    });
  }

  return {
    registered,
    primaryImageId: primary?.imageId,
  };
}

export function _internal_getListIdFromListDTO(
  dto: ListDTO,
  fallback = "",
): string {
  return getListIdFromListDTO(dto, fallback);
}