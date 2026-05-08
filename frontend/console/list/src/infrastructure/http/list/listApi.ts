// frontend/console/list/src/infrastructure/http/list/listApi.ts
import { API_BASE } from "../../../../../shell/src/shared/http/apiBase";
import type {
  CreateListInput,
  ListAggregateDTO,
  ListDTO,
  ListImageDTO,
  UpdateListInput,
} from "./types";
import { requestJSON } from "./httpClient";
import { normalizeListDocId } from "./ids";
import { extractItemsArrayFromAny } from "./extractors";
import {
  buildCreateListPayloadArray,
  buildUpdateListPayloadArray,
} from "./payloads";
import { normalizeListImageUrls } from "./listImage";

const toStringSafe = (value: unknown): string => {
  if (typeof value === "string") return value.trim();
  if (value == null) return "";
  return String(value).trim();
};

/**
 * ✅ Create list
 * POST /lists
 */
export async function createListHTTP(input: CreateListInput): Promise<ListDTO> {
  const payloadArray = buildCreateListPayloadArray(input);

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
}

/**
 * ✅ Update list
 * PUT /lists/{id}
 */
export async function updateListByIdHTTP(input: UpdateListInput): Promise<ListDTO> {
  const listId = normalizeListDocId(input?.listId);
  if (!listId) throw new Error("invalid_list_id");

  const payloadArray = buildUpdateListPayloadArray(input);

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

  const dto = await requestJSON<ListDTO>({
    method: "GET",
    path: `/lists/${encodeURIComponent(id)}`,
    debug: {
      tag: `GET /lists/${id}`,
      url: `${API_BASE}/lists/${encodeURIComponent(id)}`,
      method: "GET",
    },
  });

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

  const dto = await fetchListByIdHTTP(listId);

  return dto;
}

/**
 * GET /lists/{id}/aggregate
 */
export async function fetchListAggregateHTTP(
  listId: string,
): Promise<ListAggregateDTO> {
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
export async function fetchListImagesHTTP(
  listId: string,
): Promise<ListImageDTO[]> {
  const id = normalizeListDocId(listId);
  if (!id) throw new Error("invalid_list_id");

  return await requestJSON<ListImageDTO[]>({
    method: "GET",
    path: `/lists/${encodeURIComponent(id)}/images`,
  });
}

/**
 * ✅ listImage の「表示用URL配列」を取得
 *
 * Firebase Storage 直接アップロード後は、
 * ListImageDTO 側の downloadURL / url / imageUrl などを
 * normalizeListImageUrls 側で吸収して表示用URLへ正規化する。
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
// ✅ Firebase Storage direct upload registration
// ==========================================================

/**
 * Firebase Storage へ frontend から直接アップロード済みの listImage を backend に登録する。
 *
 * 旧方式:
 * - POST /lists/{listId}/images/signed-url
 * - signedUrl へ PUT
 * - bucket/objectPath を POST /lists/{listId}/images
 *
 * 新方式:
 * - frontend で Firebase Storage へ直接 uploadBytes / uploadBytesResumable
 * - getDownloadURL で downloadURL を取得
 * - downloadURL / objectPath を POST /lists/{listId}/images に登録
 */
export async function saveListImageFromFirebaseStorageHTTP(args: {
  listId: string;
  id: string;
  downloadURL: string;
  objectPath: string;
  size: number;
  displayOrder: number;
  fileName?: string;
  contentType?: string;
  createdBy?: string;
  createdAt?: string;
}): Promise<ListImageDTO> {
  const listId = normalizeListDocId(args.listId);
  if (!listId) throw new Error("invalid_list_id");

  const id = toStringSafe(args.id);
  const downloadURL = toStringSafe(args.downloadURL);
  const objectPath = String(args.objectPath ?? "").replace(/^\/+/, "");
  const fileName = toStringSafe(args.fileName);
  const contentType = toStringSafe(args.contentType);

  if (!id || !downloadURL || !objectPath) {
    throw new Error("invalid_list_image_payload");
  }

  const payload: any = {
    id,

    // Firebase Storage getDownloadURL() の戻り値
    downloadURL,
    downloadUrl: downloadURL,
    download_url: downloadURL,

    // 移行期間・表示側互換用
    url: downloadURL,
    imageUrl: downloadURL,
    image_url: downloadURL,

    // Firebase Storage object path
    objectPath,
    object_path: objectPath,

    size: Number(args.size ?? 0),
    displayOrder: Number(args.displayOrder ?? 0),
    display_order: Number(args.displayOrder ?? 0),

    fileName: fileName || undefined,
    file_name: fileName || undefined,

    contentType: contentType || undefined,
    content_type: contentType || undefined,

    createdBy: args.createdBy ? toStringSafe(args.createdBy) : undefined,
    created_by: args.createdBy ? toStringSafe(args.createdBy) : undefined,

    createdAt: args.createdAt ? toStringSafe(args.createdAt) : undefined,
    created_at: args.createdAt ? toStringSafe(args.createdAt) : undefined,
  };

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
    imageId: toStringSafe(args.imageId),
    updatedBy: args.updatedBy ? toStringSafe(args.updatedBy) : undefined,
    now: args.now ? toStringSafe(args.now) : undefined,
  };

  if (!payload.imageId) {
    throw new Error("invalid_image_id");
  }

  return await requestJSON<ListDTO>({
    method: "PUT",
    path: `/lists/${encodeURIComponent(listId)}/primary-image`,
    body: payload,
    debug: {
      tag: `PUT /lists/${listId}/primary-image`,
      url: `${API_BASE}/lists/${encodeURIComponent(listId)}/primary-image`,
      method: "PUT",
      body: payload,
    },
  });
}

// ==========================================================
// ✅ delete image
// DELETE /lists/{id}/images/{imageId}
// ==========================================================

function extractImageIdForDelete(args: {
  listId: string;
  imageIdOrObjectPathOrUrl: string;
}): string {
  const listId = toStringSafe(args.listId);
  const raw = toStringSafe(args.imageIdOrObjectPathOrUrl);
  if (!listId || !raw) return "";

  if (!raw.includes("/")) return raw;

  {
    const p = raw.replace(/^\/+/, "");
    const parts = p.split("/").map((x) => toStringSafe(x)).filter(Boolean);
    if (
      parts.length >= 4 &&
      parts[0] === "lists" &&
      parts[1] === listId &&
      parts[2] === "images"
    ) {
      return toStringSafe(parts[3]);
    }
  }

  try {
    const u = new URL(raw);

    const fromNameParam = toStringSafe(u.searchParams.get("name"));
    if (fromNameParam) {
      const parts = fromNameParam
        .replace(/^\/+/, "")
        .split("/")
        .map((x) => toStringSafe(x))
        .filter(Boolean);

      if (
        parts.length >= 4 &&
        parts[0] === "lists" &&
        parts[1] === listId &&
        parts[2] === "images"
      ) {
        return toStringSafe(parts[3]);
      }
    }

    const decodedPathname = decodeURIComponent(toStringSafe(u.pathname));
    const marker = "/o/";
    const markerIndex = decodedPathname.indexOf(marker);

    if (markerIndex >= 0) {
      const objectPath = decodedPathname.slice(markerIndex + marker.length);
      const parts = objectPath
        .replace(/^\/+/, "")
        .split("/")
        .map((x) => toStringSafe(x))
        .filter(Boolean);

      if (
        parts.length >= 4 &&
        parts[0] === "lists" &&
        parts[1] === listId &&
        parts[2] === "images"
      ) {
        return toStringSafe(parts[3]);
      }
    }

    const p = decodedPathname.replace(/^\/+/, "");
    const parts = p.split("/").map((x) => toStringSafe(x)).filter(Boolean);

    if (
      parts.length >= 5 &&
      parts[1] === "lists" &&
      parts[2] === listId &&
      parts[3] === "images"
    ) {
      return toStringSafe(parts[4]);
    }
  } catch {
    // noop
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
    imageIdOrObjectPathOrUrl: toStringSafe(args.imageId),
  });
  if (!imageId) throw new Error("invalid_image_id");

  return await requestJSON<any>({
    method: "DELETE",
    path: `/lists/${encodeURIComponent(listId)}/images/${encodeURIComponent(
      imageId,
    )}`,
    debug: {
      tag: `DELETE /lists/${listId}/images/${imageId}`,
      url: `${API_BASE}/lists/${encodeURIComponent(
        listId,
      )}/images/${encodeURIComponent(imageId)}`,
      method: "DELETE",
    },
  });
}