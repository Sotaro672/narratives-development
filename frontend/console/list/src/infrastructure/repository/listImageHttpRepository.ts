// frontend\console\list\src\infrastructure\repository\listImageHttpRepository.ts

import { API_BASE } from "../../../../shell/src/shared/http/apiBase";
import type { ListDTO } from "../dto/listDto";
import type { SaveListImageFromFirebaseStorageInput } from "../dto/listImageDto";
import { requestJSON } from "../http/httpClient";

export type SavedListImageDTO = {
  id: string;
  url: string;
};

export async function saveListImageFromFirebaseStorageHTTP(
  args: SaveListImageFromFirebaseStorageInput,
): Promise<SavedListImageDTO> {
  const listId = args.listId;
  if (!listId) throw new Error("invalid_list_id");

  const id = args.id;
  const url = args.url;
  const objectPath = args.objectPath.replace(/^\/+/, "");
  const fileName = args.fileName;
  const contentType = args.contentType;
  const createdBy = args.createdBy;
  const createdAt = args.createdAt;

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

  return await requestJSON<SavedListImageDTO>({
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
  const listId = args.listId;
  if (!listId) throw new Error("invalid_list_id");

  const payload = {
    imageId: args.imageId,
    updatedBy: args.updatedBy || undefined,
    now: args.now || undefined,
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

export async function deleteListImageHTTP(args: {
  listId: string;
  imageId: string;
}): Promise<any> {
  const listId = args.listId;
  if (!listId) throw new Error("invalid_list_id");

  const imageId = args.imageId;
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