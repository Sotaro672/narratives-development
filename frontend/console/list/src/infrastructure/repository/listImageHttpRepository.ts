//frontend\console\list\src\infrastructure\repository\listImageHttpRepository.ts
import { API_BASE } from "../../../../shell/src/shared/http/apiBase";
import type { ListDTO } from "../dto/listDto";
import type { ListImageDTO } from "../dto/listImageDto";
import { requestJSON } from "../http/httpClient";
import { fetchListByIdHTTP } from "./listHttpRepository";

const toStringSafe = (value: unknown): string => {
  if (typeof value === "string") return value.trim();
  if (value == null) return "";
  return String(value).trim();
};

const toListId = (value: unknown): string => toStringSafe(value);

const normalizeImageUrlsFromListDTO = (dto: ListDTO): string[] => {
  const imageUrls = (dto as any)?.imageUrls;
  if (!Array.isArray(imageUrls)) return [];

  return imageUrls
    .map((url) => toStringSafe(url))
    .filter(Boolean);
};

export async function fetchListImagesHTTP(
  listId: string,
): Promise<ListImageDTO[]> {
  const id = toListId(listId);
  if (!id) throw new Error("invalid_list_id");

  return await requestJSON<ListImageDTO[]>({
    method: "GET",
    path: `/lists/${encodeURIComponent(id)}/images`,
  });
}

export async function fetchListImageUrlsHTTP(args: {
  listId: string;
  primaryImageId?: string;
}): Promise<string[]> {
  const listId = toListId(args.listId);
  if (!listId) throw new Error("invalid_list_id");

  const dto = await fetchListByIdHTTP(listId);
  return normalizeImageUrlsFromListDTO(dto);
}

export async function saveListImageFromFirebaseStorageHTTP(args: {
  listId: string;
  id: string;
  url: string;
  objectPath: string;
  size: number;
  displayOrder: number;
  fileName?: string;
  contentType?: string;
  createdBy?: string;
  createdAt?: string;
}): Promise<ListImageDTO> {
  const listId = toListId(args.listId);
  if (!listId) throw new Error("invalid_list_id");

  const id = toStringSafe(args.id);
  const url = toStringSafe(args.url);
  const objectPath = toStringSafe(args.objectPath).replace(/^\/+/, "");
  const fileName = toStringSafe(args.fileName);
  const contentType = toStringSafe(args.contentType);
  const createdBy = toStringSafe(args.createdBy);
  const createdAt = toStringSafe(args.createdAt);

  if (!id || !url || !objectPath) {
    throw new Error("invalid_list_image_payload");
  }

  const payload: Record<string, any> = {
    id,
    url,
    objectPath,
    size: Number(args.size ?? 0),
    displayOrder: Number(args.displayOrder ?? 0),
    fileName: fileName || undefined,
    contentType: contentType || undefined,
    createdBy: createdBy || undefined,
    createdAt: createdAt || undefined,
  };

  for (const key of Object.keys(payload)) {
    if (payload[key] === undefined) delete payload[key];
  }

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

export async function setListPrimaryImageHTTP(args: {
  listId: string;
  imageId: string;
  updatedBy?: string;
  now?: string;
}): Promise<ListDTO> {
  const listId = toListId(args.listId);
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

function extractImageIdForDelete(args: {
  listId: string;
  imageIdOrObjectPathOrUrl: string;
}): string {
  const listId = toStringSafe(args.listId);
  const raw = toStringSafe(args.imageIdOrObjectPathOrUrl);
  if (!listId || !raw) return "";

  if (!raw.includes("/") && !raw.includes("?")) return raw;

  {
    const objectPath = raw.replace(/^\/+/, "");
    const parts = objectPath
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

  try {
    const url = new URL(raw);
    const decodedPathname = decodeURIComponent(toStringSafe(url.pathname));

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

    const pathParts = decodedPathname
      .replace(/^\/+/, "")
      .split("/")
      .map((x) => toStringSafe(x))
      .filter(Boolean);

    const listsIndex = pathParts.indexOf("lists");
    if (
      listsIndex >= 0 &&
      pathParts[listsIndex + 1] === listId &&
      pathParts[listsIndex + 2] === "images"
    ) {
      return toStringSafe(pathParts[listsIndex + 3]);
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
  const listId = toListId(args.listId);
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