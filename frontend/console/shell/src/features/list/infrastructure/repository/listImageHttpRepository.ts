// frontend/console/shell/src/features/list/infrastructure/repository/listImageHttpRepository.ts

import type { List } from "../../../../shared/types/list";
import type { SaveListImageFromFirebaseStorageInput } from "../dto/listImageDto";
import { requestJSON } from "../http/httpClient";

export type SavedListImageDTO = {
  id: string;
  listId: string;
  url: string;
  displayOrder: number;
  createdAt: string;
  createdBy: string;
  updatedAt?: string;
  updatedBy?: string;
};

export type DeleteListImageResponse = {
  ok: boolean;
  listId: string;
  imageId: string;
};

export async function saveListImageFromFirebaseStorageHTTP(
  args: SaveListImageFromFirebaseStorageInput,
): Promise<SavedListImageDTO> {
  const listId = args.listId;
  const id = args.id;
  const url = args.url;

  if (!listId) {
    throw new Error("invalid_list_id");
  }

  if (!id || !url) {
    throw new Error("invalid_list_image_payload");
  }

  return requestJSON<SavedListImageDTO>({
    method: "POST",
    path: `/lists/${encodeURIComponent(listId)}/images`,
    body: {
      id,
      url,
      displayOrder: args.displayOrder,
    },
  });
}

export async function setListPrimaryImageHTTP(args: {
  listId: string;
  imageId: string;
}): Promise<List> {
  const listId = args.listId;
  const imageId = args.imageId;

  if (!listId) {
    throw new Error("invalid_list_id");
  }

  if (!imageId) {
    throw new Error("invalid_image_id");
  }

  return requestJSON<List>({
    method: "PUT",
    path: `/lists/${encodeURIComponent(listId)}/primary-image`,
    body: {
      imageId,
    },
  });
}

export async function deleteListImageHTTP(args: {
  listId: string;
  imageId: string;
}): Promise<DeleteListImageResponse> {
  const listId = args.listId;
  const imageId = args.imageId;

  if (!listId) {
    throw new Error("invalid_list_id");
  }

  if (!imageId) {
    throw new Error("invalid_image_id");
  }

  return requestJSON<DeleteListImageResponse>({
    method: "DELETE",
    path: `/lists/${encodeURIComponent(
      listId,
    )}/images/${encodeURIComponent(imageId)}`,
  });
}