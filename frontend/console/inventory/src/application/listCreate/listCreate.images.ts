// frontend/console/inventory/src/application/listCreate/listCreate.images.ts

import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

import {
  saveListImageFromGCSHTTP,
  setListPrimaryImageHTTP,
  issueListImageSignedUrlHTTP,
  type ListDTO,
  type SignedListImageUploadDTO,
} from "../../../../list/src/infrastructure/http/list";

import { normalizeListId, s } from "./listCreate.utils";

export function dedupeFiles(prev: File[], add: File[]): File[] {
  const exists = new Set(prev.map((f) => `${f.name}__${f.size}__${f.lastModified}`));
  const filtered = add.filter((f) => !exists.has(`${f.name}__${f.size}__${f.lastModified}`));
  return [...prev, ...filtered];
}

function getListIdFromListDTO(dto: ListDTO, fallback = ""): string {
  const raw =
    s((dto as any)?.id) ||
    s((dto as any)?.ID) ||
    s((dto as any)?.listId) ||
    s((dto as any)?.ListID) ||
    s(fallback);

  return normalizeListId(raw);
}

async function putFileToSignedUrl(args: { signedUrl: string; file: File }): Promise<void> {
  const url = s(args.signedUrl);
  const file = args.file;
  if (!url) throw new Error("missing_signed_url");

  const res = await fetch(url, {
    method: "PUT",
    headers: {
      "Content-Type": file.type || "application/octet-stream",
    },
    body: file,
  });

  if (!res.ok) {
    const t = await res.text().catch(() => "");
    throw new Error(`listImage_upload_failed_${res.status}_${t || "no_body"}`);
  }
}

/**
 * ✅ 複数画像を Policy A（signed-url）でアップロード→メタ登録→primary 設定
 */
export async function uploadListImagesPolicyA(args: {
  listId: string;
  files: File[];
  mainImageIndex: number;
  createdBy?: string;
}): Promise<{ registered: Array<{ imageId: string; displayOrder: number }>; primaryImageId?: string }> {
  const listId = normalizeListId(args.listId);
  const files = Array.isArray(args.files) ? args.files : [];
  const mainImageIndex = Number.isFinite(Number(args.mainImageIndex)) ? Number(args.mainImageIndex) : 0;

  if (!listId) throw new Error("invalid_list_id");
  if (files.length === 0) return { registered: [] };

  if (!files[mainImageIndex]) {
    throw new Error("メイン画像が選択されていません。");
  }

  const uid = s(args.createdBy) || s(auth.currentUser?.uid) || "system";
  const now = new Date().toISOString();

  const registered: Array<{ imageId: string; displayOrder: number }> = [];

  for (let i = 0; i < files.length; i++) {
    const file = files[i];
    if (!file) continue;

    const signed: SignedListImageUploadDTO = await issueListImageSignedUrlHTTP({
      listId,
      fileName: file.name,
      contentType: file.type || "application/octet-stream",
      size: file.size || 0,
      displayOrder: i,
    });

    const objectPath = s(signed.objectPath);
    const signedUrl = s(signed.signedUrl);
    const bucket = s(signed.bucket);

    if (!objectPath || !signedUrl) {
      throw new Error("signed_url_response_invalid");
    }

    await putFileToSignedUrl({ signedUrl, file });

    await saveListImageFromGCSHTTP({
      listId,
      id: objectPath,
      fileName: s(file.name),
      bucket,
      objectPath,
      size: Number(file.size || 0),
      displayOrder: i,
      createdBy: uid,
      createdAt: now,
    });

    registered.push({ imageId: objectPath, displayOrder: i });
  }

  const primary = registered.find((x) => x.displayOrder === mainImageIndex) || registered[0];

  if (primary?.imageId) {
    await setListPrimaryImageHTTP({
      listId,
      imageId: primary.imageId,
      updatedBy: uid,
      now,
    } as any);
  }

  return { registered, primaryImageId: primary?.imageId };
}

// （createListWithImages 側で使用するための内部ヘルパーをここに残す場合）
// export { getListIdFromListDTO }; // ←公開したい場合のみ
export function _internal_getListIdFromListDTO(dto: ListDTO, fallback = ""): string {
  return getListIdFromListDTO(dto, fallback);
}
