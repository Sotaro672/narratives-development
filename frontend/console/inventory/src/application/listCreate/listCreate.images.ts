// frontend/console/inventory/src/application/listCreate/listCreate.images.ts

import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

import {
  saveListImageFromFirebaseStorageHTTP,
  setListPrimaryImageHTTP,
} from "../../../../list/src/infrastructure/repository";

import type { ListDTO } from "../../../../list/src/infrastructure/dto";

import { uploadListImageToFirebaseStorage } from "../../../../list/src/infrastructure/firebase/listImageStorage";

/**
 * 複数画像を Firebase Storage へ直接アップロード
 * → backend にメタ情報登録
 * → primary image 設定
 *
 * Policy B:
 * - List 作成後の listId を使って Firebase Storage へ upload
 * - Firebase Storage download URL を取得
 * - saveListImageFromFirebaseStorageHTTP で image record を登録
 *
 * primary:
 * - backend の List.imageId は images subcollection docID
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
  const listId = String(args.listId ?? "").trim();
  const files = Array.isArray(args.files) ? args.files : [];

  const requestedMainImageIndex = Number.isFinite(Number(args.mainImageIndex))
    ? Number(args.mainImageIndex)
    : 0;

  const mainImageIndex =
    requestedMainImageIndex >= 0 && requestedMainImageIndex < files.length
      ? requestedMainImageIndex
      : 0;

  if (!listId) throw new Error("invalid_list_id");
  if (files.length === 0) return { registered: [] };

  const uid = args.createdBy || auth.currentUser?.uid || "system";
  const now = new Date().toISOString();

  const registered: Array<{ imageId: string; displayOrder: number }> = [];

  for (let i = 0; i < files.length; i++) {
    const file = files[i];
    if (!file) continue;

    const uploaded = await uploadListImageToFirebaseStorage({
      listId,
      file,
    });

    await saveListImageFromFirebaseStorageHTTP({
      listId,
      id: uploaded.imageId,
      url: uploaded.url,
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

export function _internal_getListIdFromListDTO(dto: ListDTO): string {
  return dto.id;
}