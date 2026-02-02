//frontend\console\list\src\infrastructure\http\list\listApi.ts
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
import { normalizeSignedListImageUploadDTO } from "./signedUrl";

/**
 * ✅ Create list
 * POST /lists
 */
export async function createListHTTP(input: CreateListInput): Promise<ListDTO> {
  const payloadArray = buildCreateListPayloadArray(input);

  console.log("[list/listRepositoryHTTP] createListHTTP payload", payloadArray);

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
      imageUrlsCount: Array.isArray((dto as any)?.imageUrls)
        ? (dto as any).imageUrls.length
        : 0,
      dto,
    });

    return dto;
  } catch (e1) {
    // ✅ inventoryIdHint は pb__tb をそのまま使う（splitしない）
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

/**
 * ✅ NEW: signed-url 発行（Policy A）
 * POST /lists/{id}/images/signed-url
 */
export async function issueListImageSignedUrlHTTP(args: {
  listId: string;
  fileName: string;
  contentType: string;
  size: number;
  displayOrder: number;
}): Promise<SignedListImageUploadDTO> {
  // ✅ ここは list の docId なので normalize してOK（事故混入対策）
  const listId = normalizeListDocId(args.listId);
  if (!listId) throw new Error("invalid_list_id");

  const payload = {
    fileName: s(args.fileName),
    contentType: s(args.contentType) || "application/octet-stream",
    size: Number(args.size || 0),
    displayOrder: Number(args.displayOrder || 0),
  };

  // backend の返却キー揺れ（uploadUrl / signedUrl / publicUrl など）をここで吸収する
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

  return normalizeSignedListImageUploadDTO(raw);
}

/**
 * POST /lists/{id}/images
 * - GCS objectPath を登録する（アップロード自体は別途）
 */
export async function saveListImageFromGCSHTTP(args: {
  listId: string;
  id: string; // ListImage.ID
  fileName?: string;
  bucket?: string; // optional
  objectPath: string;
  size: number; // bytes
  displayOrder: number;
  createdBy?: string;
  createdAt?: string; // RFC3339 optional
}): Promise<ListImageDTO> {
  const listId = normalizeListDocId(args.listId);
  if (!listId) throw new Error("invalid_list_id");

  const payload = {
    id: String(args.id ?? "").trim(),
    fileName: String(args.fileName ?? "").trim(),
    bucket: String(args.bucket ?? "").trim(),
    objectPath: String(args.objectPath ?? "").trim(),
    size: Number(args.size ?? 0),
    displayOrder: Number(args.displayOrder ?? 0),
    createdBy: String(args.createdBy ?? "").trim(),
    createdAt: args.createdAt ? String(args.createdAt).trim() : undefined,
  };

  return await requestJSON<ListImageDTO>({
    method: "POST",
    path: `/lists/${encodeURIComponent(listId)}/images`,
    body: payload,
  });
}

/**
 * PUT /lists/{id}/primary-image
 */
export async function setListPrimaryImageHTTP(args: {
  listId: string;
  imageId: string;
  updatedBy?: string;
  now?: string; // RFC3339 optional
}): Promise<ListDTO> {
  const listId = normalizeListDocId(args.listId);
  if (!listId) throw new Error("invalid_list_id");

  const payload = {
    imageId: String(args.imageId ?? "").trim(),
    updatedBy: args.updatedBy ? String(args.updatedBy).trim() : undefined,
    now: args.now ? String(args.now).trim() : undefined,
  };

  return await requestJSON<ListDTO>({
    method: "PUT",
    path: `/lists/${encodeURIComponent(listId)}/primary-image`,
    body: payload,
  });
}
