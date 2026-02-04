// frontend/console/list/src/infrastructure/http/list/listApi.ts
import { API_BASE } from "./config";
import type {
  CreateListInput,
  ListAggregateDTO,
  ListDTO,
  ListImageDTO,
  SignedListImageUploadDTO,
  UpdateListInput,
} from "./types";
import { requestJSON } from "./httpClient";
import { normalizeListDocId } from "./ids";
import { extractFirstItemFromAny, extractItemsArrayFromAny } from "./extractors";
import {
  buildCreateListPayloadArray,
  buildCreateListPayloadMap,
  buildUpdateListPayloadArray,
  buildUpdateListPayloadMap,
} from "./payloads";
import { s } from "./string";
import { ensureDetailHasImageUrls } from "./detailFallback";
import { normalizeListImageUrls } from "./listImage";

/**
 * ✅ Create list
 * POST /lists
 */
export async function createListHTTP(input: CreateListInput): Promise<ListDTO> {
  const payloadArray = buildCreateListPayloadArray(input);

  console.log("[list/listRepositoryHTTP] createListHTTP payload", payloadArray);

  // ✅ DEBUG: selected files (if any)
  try {
    const anyInput = input as any;
    const selectedFiles =
      (Array.isArray(anyInput?.selectedFiles) && anyInput.selectedFiles) ||
      (Array.isArray(anyInput?.files) && anyInput.files) ||
      (Array.isArray(anyInput?.images) && anyInput.images) ||
      (Array.isArray(anyInput?.imageFiles) && anyInput.imageFiles) ||
      [];

    console.log(
      "[debug] selected files",
      selectedFiles.map((f: any) => ({
        name: f?.name,
        size: f?.size,
        lastModified: f?.lastModified,
        type: f?.type,
      })),
    );
  } catch (e) {
    console.log("[debug] selected files (log_failed)", { err: String(e) });
  }

  try {
    return await requestJSON<ListDTO>({
      method: "POST",
      path: "/lists",
      body: payloadArray,
      debug: {
        tag: "POST /lists",
        url: `${API_BASE}/lists`,
        method: "POST",
        body: payloadArray,
      },
    });
  } catch (e) {
    const msg = String(e instanceof Error ? e.message : e);

    if (msg === "invalid json") {
      const payloadMap = buildCreateListPayloadMap(input);

      console.log("[list/listRepositoryHTTP] createListHTTP retry payload(map)", payloadMap);

      return await requestJSON<ListDTO>({
        method: "POST",
        path: "/lists",
        body: payloadMap,
        debug: {
          tag: "POST /lists (retry map)",
          url: `${API_BASE}/lists`,
          method: "POST",
          body: payloadMap,
        },
      });
    }

    throw e;
  }
}

/**
 * ✅ Update list
 * PUT /lists/{id}
 */
export async function updateListByIdHTTP(input: UpdateListInput): Promise<ListDTO> {
  const listId = normalizeListDocId(input?.listId);
  if (!listId) throw new Error("invalid_list_id");

  const payloadArray = buildUpdateListPayloadArray(input);

  console.log("[list/listRepositoryHTTP] updateListByIdHTTP payload", {
    listId,
    payload: payloadArray,
  });

  try {
    return await requestJSON<ListDTO>({
      method: "PUT",
      path: `/lists/${encodeURIComponent(listId)}`,
      body: payloadArray,
      debug: {
        tag: `PUT /lists/${listId}`,
        url: `${API_BASE}/lists/${encodeURIComponent(listId)}`,
        method: "PUT",
        body: payloadArray,
      },
    });
  } catch (e) {
    const msg = String(e instanceof Error ? e.message : e);

    if (msg === "invalid json") {
      const payloadMap = buildUpdateListPayloadMap(input);

      console.log("[list/listRepositoryHTTP] updateListByIdHTTP retry payload(map)", {
        listId,
        payload: payloadMap,
      });

      return await requestJSON<ListDTO>({
        method: "PUT",
        path: `/lists/${encodeURIComponent(listId)}`,
        body: payloadMap,
        debug: {
          tag: `PUT /lists/${listId} (retry map)`,
          url: `${API_BASE}/lists/${encodeURIComponent(listId)}`,
          method: "PUT",
          body: payloadMap,
        },
      });
    }

    throw e;
  }
}

/**
 * ✅ List lists
 * GET /lists
 */
export async function fetchListsHTTP(): Promise<ListDTO[]> {
  const json = await requestJSON<any>({
    method: "GET",
    path: "/lists",
  });

  const items = extractItemsArrayFromAny(json);
  return items as ListDTO[];
}

/**
 * GET /lists/{id}
 */
export async function fetchListByIdHTTP(listId: string): Promise<ListDTO> {
  const id = normalizeListDocId(listId);
  if (!id) {
    throw new Error("invalid_list_id");
  }

  const dto0 = await requestJSON<ListDTO>({
    method: "GET",
    path: `/lists/${encodeURIComponent(id)}`,
    debug: {
      tag: `GET /lists/${id}`,
      url: `${API_BASE}/lists/${encodeURIComponent(id)}`,
      method: "GET",
    },
  });

  const dto = await ensureDetailHasImageUrls(dto0, id, fetchListImagesHTTP);

  try {
    const anyDto = dto as any;
    console.log("[list/listRepositoryHTTP] fetchListByIdHTTP ok", {
      listId: id,
      hasCreatedByName: Boolean(s(anyDto?.createdByName)),
      createdBy: s(anyDto?.createdBy),
      createdByName: s(anyDto?.createdByName),
      updatedBy: s(anyDto?.updatedBy),
      updatedByName: s(anyDto?.updatedByName),
      createdAt: s(anyDto?.createdAt),
      updatedAt: s(anyDto?.updatedAt),
      imageId: s(anyDto?.imageId),
      imageUrlsCount: Array.isArray(anyDto?.imageUrls) ? anyDto.imageUrls.length : 0,
      keys: anyDto && typeof anyDto === "object" ? Object.keys(anyDto) : [],
      dto,
    });
  } catch (e) {
    console.log("[list/listRepositoryHTTP] fetchListByIdHTTP ok (log_failed)", {
      listId: id,
      err: String(e),
    });
  }

  return dto;
}

/**
 * ✅ ListDetail 用
 */
export async function fetchListDetailHTTP(args: {
  listId: string;
  inventoryIdHint?: string;
}): Promise<ListDTO> {
  const listId = normalizeListDocId(args.listId);
  if (!listId) {
    throw new Error("invalid_list_id");
  }

  console.log("[list/listRepositoryHTTP] fetchListDetailHTTP start", {
    listId,
    inventoryIdHint: s(args.inventoryIdHint),
    url: `${API_BASE}/lists/${encodeURIComponent(listId)}`,
  });

  try {
    const dto = await fetchListByIdHTTP(listId);

    console.log("[list/listRepositoryHTTP] fetchListDetailHTTP resolved", {
      source: "GET /lists/{id}",
      listId,
      createdByName: s((dto as any)?.createdByName),
      updatedByName: s((dto as any)?.updatedByName),
      imageUrlsCount: Array.isArray((dto as any)?.imageUrls) ? (dto as any).imageUrls.length : 0,
      dto,
    });

    return dto;
  } catch (e1) {
    const inv = s(args.inventoryIdHint) || listId;

    console.log("[list/listRepositoryHTTP] fetchListDetailHTTP fallback start", {
      listId,
      inventoryId: inv,
      url: `${API_BASE}/lists?inventoryId=${encodeURIComponent(inv)}`,
      err: String(e1 instanceof Error ? e1.message : e1),
    });

    try {
      const json = await requestJSON<any>({
        method: "GET",
        path: `/lists?inventoryId=${encodeURIComponent(inv)}`,
        debug: {
          tag: `GET /lists?inventoryId=${inv}`,
          url: `${API_BASE}/lists?inventoryId=${encodeURIComponent(inv)}`,
          method: "GET",
        },
      });

      const first0 = extractFirstItemFromAny(json);
      if (!first0) throw new Error("not_found");

      const first = await ensureDetailHasImageUrls(first0 as ListDTO, listId, fetchListImagesHTTP);

      console.log("[list/listRepositoryHTTP] fetchListDetailHTTP resolved", {
        source: "GET /lists?inventoryId=xxx",
        listId,
        inventoryId: inv,
        createdByName: s((first as any)?.createdByName),
        updatedByName: s((first as any)?.updatedByName),
        imageUrlsCount: Array.isArray((first as any)?.imageUrls)
          ? (first as any).imageUrls.length
          : 0,
        dto: first,
        raw: json,
      });

      return first as ListDTO;
    } catch (e2) {
      console.log("[list/listRepositoryHTTP] fetchListDetailHTTP fallback failed", {
        listId,
        inventoryId: inv,
        err: String(e2 instanceof Error ? e2.message : e2),
      });
      throw e1;
    }
  }
}

/**
 * GET /lists/{id}/aggregate
 */
export async function fetchListAggregateHTTP(listId: string): Promise<ListAggregateDTO> {
  const id = normalizeListDocId(listId);
  if (!id) throw new Error("invalid_list_id");

  return await requestJSON<ListAggregateDTO>({
    method: "GET",
    path: `/lists/${encodeURIComponent(id)}/aggregate`,
  });
}

/**
 * GET /lists/{id}/images
 */
export async function fetchListImagesHTTP(listId: string): Promise<ListImageDTO[]> {
  const id = normalizeListDocId(listId);
  if (!id) throw new Error("invalid_list_id");

  return await requestJSON<ListImageDTO[]>({
    method: "GET",
    path: `/lists/${encodeURIComponent(id)}/images`,
  });
}

/**
 * ✅ NEW: listImage bucket の「表示用URL配列」を取得
 */
export async function fetchListImageUrlsHTTP(args: {
  listId: string;
  primaryImageId?: string;
}): Promise<string[]> {
  const listId = normalizeListDocId(args.listId);
  if (!listId) throw new Error("invalid_list_id");

  const imgs = await fetchListImagesHTTP(listId);
  return normalizeListImageUrls(imgs, args.primaryImageId);
}

// ==========================================================
// ✅ Policy A: Signed URL (backend 実仕様に固定)
// ==========================================================

export async function issueListImageSignedUrlHTTP(args: {
  listId: string;
  fileName: string;
  contentType: string;
  size: number;
  displayOrder: number;
}): Promise<SignedListImageUploadDTO> {
  const listId = normalizeListDocId(args.listId);
  if (!listId) throw new Error("invalid_list_id");

  const payload = {
    fileName: s(args.fileName),
    contentType: s(args.contentType) || "application/octet-stream",
    size: Number(args.size || 0),
    displayOrder: Number(args.displayOrder || 0),
  };

  const raw = await requestJSON<any>({
    method: "POST",
    path: `/lists/${encodeURIComponent(listId)}/images/signed-url`,
    body: payload,
    debug: {
      tag: `POST /lists/${listId}/images/signed-url`,
      url: `${API_BASE}/lists/${encodeURIComponent(listId)}/images/signed-url`,
      method: "POST",
      body: payload,
    },
  });

  // ✅ snake_case/camelCase 両対応
  const rawId = s(raw?.id) || s(raw?.ID);
  const rawBucket = s(raw?.bucket) || s(raw?.Bucket);
  const rawObjectPath =
    s(raw?.objectPath) ||
    s(raw?.object_path) ||
    s(raw?.ObjectPath) ||
    s(raw?.object_path_str);

  const rawUploadUrl =
    s(raw?.uploadUrl) ||
    s(raw?.upload_url) ||
    s(raw?.signedUrl) ||
    s(raw?.signed_url) ||
    s(raw?.UploadURL);

  const rawPublicUrl = s(raw?.publicUrl) || s(raw?.public_url) || s(raw?.PublicURL);

  const out: SignedListImageUploadDTO = {
    id: rawId,
    bucket: rawBucket,
    objectPath: rawObjectPath,
    signedUrl: rawUploadUrl,
    publicUrl: rawPublicUrl || undefined,
    expiresAt: s(raw?.expiresAt) || s(raw?.expires_at) || undefined,
    contentType: s(raw?.contentType) || s(raw?.content_type) || undefined,
    size: Number.isFinite(Number(raw?.size)) ? Number(raw.size) : undefined,
    displayOrder: Number.isFinite(Number(raw?.displayOrder)) ? Number(raw.displayOrder) : undefined,
    fileName: s(raw?.fileName) || s(raw?.file_name) || undefined,
  };

  console.log("[list/listRepositoryHTTP] signed-url response normalized", {
    listId,
    raw,
    out,
  });

  if (!out.id || !out.bucket || !out.objectPath || !out.signedUrl) {
    throw new Error("signed_url_response_invalid");
  }

  // canonical: "lists/{listId}/images/{imageId}"
  {
    const op = out.objectPath.replace(/^\/+/, "");
    const parts = op.split("/").map((x) => s(x)).filter(Boolean);
    if (
      parts.length !== 4 ||
      parts[0] !== "lists" ||
      parts[1] !== listId ||
      parts[2] !== "images" ||
      parts[3] !== out.id
    ) {
      throw new Error("signed_url_object_path_not_canonical");
    }
  }

  return out;
}

export async function saveListImageFromGCSHTTP(args: {
  listId: string;
  id: string;
  bucket: string;
  objectPath: string;
  size: number;
  displayOrder: number;
  fileName?: string;
  createdBy?: string;
  createdAt?: string;
}): Promise<ListImageDTO> {
  const listId = normalizeListDocId(args.listId);
  if (!listId) throw new Error("invalid_list_id");

  const id = s(args.id);
  const bucket = s(args.bucket);

  // ✅ objectPath が string で来ない事故も吸収
  const objectPath = String(args.objectPath ?? "").replace(/^\/+/, "");

  const fileName = s(args.fileName);

  // ✅ objectPath が空なら POST しない（原因特定のためログを厚くする）
  if (!id || !bucket || !objectPath) {
    console.log("[list/listRepositoryHTTP] saveImageFromGCS invalid payload", {
      listId,
      id,
      bucket,
      objectPath,
      keys: args && typeof args === "object" ? Object.keys(args as any) : [],
      args,
    });
    throw new Error("invalid_list_image_payload");
  }

  // canonical check: lists/{listId}/images/{imageId}
  const parts = objectPath.split("/").map((x) => s(x)).filter(Boolean);
  if (
    parts.length !== 4 ||
    parts[0] !== "lists" ||
    parts[1] !== listId ||
    parts[2] !== "images" ||
    parts[3] !== id
  ) {
    console.log("[list/listRepositoryHTTP] saveImageFromGCS objectPath mismatch", {
      listId,
      id,
      objectPath,
      parts,
    });
    throw new Error("objectPath_id_mismatch");
  }

  const payload: any = {
    id,
    bucket,

    // ✅ backend の実装差異に備えて両方送る（どっちを読んでても通す）
    objectPath, // camelCase
    object_path: objectPath, // snake_case

    size: Number(args.size ?? 0),
    displayOrder: Number(args.displayOrder ?? 0),

    // 互換用（backend 側で required になっても耐える）
    fileName: fileName || undefined,

    createdBy: args.createdBy ? s(args.createdBy) : undefined,
    createdAt: args.createdAt ? s(args.createdAt) : undefined,
  };

  console.log("[list/listRepositoryHTTP] saveImageFromGCS payload", {
    listId,
    payload,
    bodyJSON: JSON.stringify(payload),
    hasObjectPath: Object.prototype.hasOwnProperty.call(payload, "objectPath"),
    hasObjectPathSnake: Object.prototype.hasOwnProperty.call(payload, "object_path"),
    objectPath: payload.objectPath,
    object_path: payload.object_path,
  });

  return await requestJSON<ListImageDTO>({
    method: "POST",
    path: `/lists/${encodeURIComponent(listId)}/images`,
    body: payload,
    debug: {
      tag: `POST /lists/${listId}/images`,
      url: `${API_BASE}/lists/${encodeURIComponent(listId)}/images`,
      method: "POST",
      body: payload,
    },
  });
}

/**
 * PUT /lists/{id}/primary-image
 */
export async function setListPrimaryImageHTTP(args: {
  listId: string;
  imageId: string;
  updatedBy?: string;
  now?: string;
}): Promise<ListDTO> {
  const listId = normalizeListDocId(args.listId);
  if (!listId) throw new Error("invalid_list_id");

  const payload = {
    imageId: s(args.imageId),
    updatedBy: args.updatedBy ? s(args.updatedBy) : undefined,
    now: args.now ? s(args.now) : undefined,
  };

  if (!payload.imageId) {
    throw new Error("invalid_image_id");
  }

  return await requestJSON<ListDTO>({
    method: "PUT",
    path: `/lists/${encodeURIComponent(listId)}/primary-image`,
    body: payload,
  });
}

// ==========================================================
// ✅ delete image
// DELETE /lists/{id}/images/{imageId}
// ==========================================================

function extractImageIdForDelete(args: { listId: string; imageIdOrObjectPathOrUrl: string }): string {
  const listId = s(args.listId);
  const raw = s(args.imageIdOrObjectPathOrUrl);
  if (!listId || !raw) return "";

  // 1) already imageId (no slash)
  if (!raw.includes("/")) return raw;

  // 2) objectPath: "lists/{listId}/images/{imageId}"
  {
    const p = raw.replace(/^\/+/, "");
    const parts = p.split("/").map((x) => s(x)).filter(Boolean);
    if (parts.length >= 4 && parts[0] === "lists" && parts[1] === listId && parts[2] === "images") {
      return s(parts[3]);
    }
  }

  // 3) URL: https://storage.googleapis.com/{bucket}/lists/{listId}/images/{imageId}
  try {
    const u = new URL(raw);
    const p = s(u.pathname).replace(/^\/+/, "");
    const parts = p.split("/").map((x) => s(x)).filter(Boolean);

    // parts[0]=bucket, parts[1]=lists, parts[2]=listId, parts[3]=images, parts[4]=imageId
    if (parts.length >= 5 && parts[1] === "lists" && parts[2] === listId && parts[3] === "images") {
      return s(parts[4]);
    }
  } catch {
    // ignore
  }

  return "";
}

export async function deleteListImageHTTP(args: {
  listId: string;
  imageId: string;
}): Promise<any> {
  const listId = normalizeListDocId(args.listId);
  if (!listId) throw new Error("invalid_list_id");

  const imageId = extractImageIdForDelete({
    listId,
    imageIdOrObjectPathOrUrl: s(args.imageId),
  });
  if (!imageId) throw new Error("invalid_image_id");

  return await requestJSON<any>({
    method: "DELETE",
    path: `/lists/${encodeURIComponent(listId)}/images/${encodeURIComponent(imageId)}`,
    debug: {
      tag: `DELETE /lists/${listId}/images/${imageId}`,
      url: `${API_BASE}/lists/${encodeURIComponent(listId)}/images/${encodeURIComponent(imageId)}`,
      method: "DELETE",
    },
  });
}
