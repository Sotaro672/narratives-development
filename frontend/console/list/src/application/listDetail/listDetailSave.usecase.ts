// frontend/console/list/src/application/listDetail/listDetailSave.usecase.ts

import {
  deleteListImageHTTP,
  saveListImageFromFirebaseStorageHTTP,
  setListPrimaryImageHTTP,
} from "../../infrastructure/repository";

import type { ListDTO } from "../../infrastructure/dto";

import {
  loadListDetailDTO,
  normalizeImageUrls,
  updateListDetailDTO,
  type ListDetailDTO,
} from "../listDetailService";

import { uploadListImageToFirebaseStorage } from "../../infrastructure/firebase/listImageStorage";
import { extractListImageIdFromUrlOrObjectPath } from "./listImageId";

export type SaveListDetailDraftImage = {
  url: string;
  isNew: boolean;
  file?: File;
};

export type SaveListDetailChangesInput = {
  listId: string;
  inventoryIdHint?: string;

  currentDTO: ListDetailDTO | null;

  title: string;
  description: string;
  decision?: "list" | "hold";

  assigneeId?: string;
  updatedBy: string;

  draftPriceRows: any[];
  draftImages: SaveListDetailDraftImage[];

  mainImageIndex: number;
};

export type SaveListDetailChangesResult = {
  dto: ListDTO;
};

type SavedDraftImageItem = {
  draftIndex: number;
  imageId: string;
  url: string;
  objectPath: string;
  displayOrder: number;
};

function isNewDraftImageWithFile(
  image: SaveListDetailDraftImage | undefined,
): image is SaveListDetailDraftImage & { file: File } {
  return Boolean(image?.isNew && image?.file);
}

function normalizeDraftImages(
  draftImages: SaveListDetailDraftImage[] | null | undefined,
): SaveListDetailDraftImage[] {
  return Array.isArray(draftImages) ? draftImages : [];
}

function buildAfterUrls(args: {
  draftImages: SaveListDetailDraftImage[];
  uploadedItems: SavedDraftImageItem[];
}): string[] {
  const { draftImages, uploadedItems } = args;

  const uploadedByDraftIndex = new Map<number, SavedDraftImageItem>();
  for (const item of uploadedItems) {
    uploadedByDraftIndex.set(item.draftIndex, item);
  }

  const out: string[] = [];
  const seen = new Set<string>();

  draftImages.forEach((image, index) => {
    let url = "";

    const uploaded = uploadedByDraftIndex.get(index);
    if (uploaded) {
      url = String(uploaded.url ?? "").trim();
    } else if (!image?.isNew) {
      url = String(image?.url ?? "").trim();
    }

    if (!url || seen.has(url)) return;

    seen.add(url);
    out.push(url);
  });

  return out;
}

function resolvePrimaryImageId(args: {
  listId: string;
  selectedUrl: string;
  uploadedItems: SavedDraftImageItem[];
}): string {
  const listId = String(args.listId ?? "").trim();
  const selectedUrl = String(args.selectedUrl ?? "").trim();

  if (!listId || !selectedUrl) return "";

  const uploadedPrimary = args.uploadedItems.find(
    (item) => String(item.url ?? "").trim() === selectedUrl,
  );

  if (uploadedPrimary?.imageId) {
    return String(uploadedPrimary.imageId ?? "").trim();
  }

  return extractListImageIdFromUrlOrObjectPath({
    listId,
    raw: selectedUrl,
  });
}

export async function saveListDetailChanges(
  input: SaveListDetailChangesInput,
): Promise<SaveListDetailChangesResult> {
  const listId = String(input.listId ?? "").trim();
  if (!listId) throw new Error("invalid_list_id");

  const updatedBy = String(input.updatedBy ?? "").trim() || "system";
  const draftImages = normalizeDraftImages(input.draftImages);

  const beforeUrls = normalizeImageUrls(input.currentDTO);
  const uploadedItems: SavedDraftImageItem[] = [];

  for (let index = 0; index < draftImages.length; index++) {
    const image = draftImages[index];

    if (!isNewDraftImageWithFile(image)) continue;

    const file = image.file;
    const displayOrder = beforeUrls.length + uploadedItems.length;

    const uploaded = await uploadListImageToFirebaseStorage({
      listId,
      file,
    });

    const saved = await saveListImageFromFirebaseStorageHTTP({
      listId,
      id: uploaded.imageId,
      url: uploaded.url,
      objectPath: uploaded.objectPath,
      size: file.size,
      displayOrder,
      fileName: file.name,
      contentType: file.type || "application/octet-stream",
      createdBy: updatedBy,
      createdAt: new Date().toISOString(),
    });

    const savedUrl = String((saved as any)?.url ?? "").trim() || uploaded.url;

    uploadedItems.push({
      draftIndex: index,
      imageId: uploaded.imageId,
      url: savedUrl,
      objectPath: uploaded.objectPath,
      displayOrder,
    });
  }

  const afterUrls = buildAfterUrls({
    draftImages,
    uploadedItems,
  });

  const removedUrls = beforeUrls.filter((url) => !afterUrls.includes(url));

  for (const removedUrl of removedUrls) {
    const imageId =
      extractListImageIdFromUrlOrObjectPath({
        listId,
        raw: removedUrl,
      }) || String(removedUrl ?? "").trim();

    if (!imageId) continue;

    await deleteListImageHTTP({
      listId,
      imageId,
    });
  }

  const selectedUrl = String(afterUrls[input.mainImageIndex] ?? "").trim();

  if (selectedUrl) {
    const primaryImageId = resolvePrimaryImageId({
      listId,
      selectedUrl,
      uploadedItems,
    });

    if (primaryImageId) {
      await setListPrimaryImageHTTP({
        listId,
        imageId: primaryImageId,
        updatedBy,
        now: new Date().toISOString(),
      });
    }
  }

  await updateListDetailDTO({
    listId,
    title: input.title,
    description: input.description,
    priceRows: Array.isArray(input.draftPriceRows) ? input.draftPriceRows : [],
    decision: input.decision,
    assigneeId: input.assigneeId,
    updatedBy,
  });

  const dto = await loadListDetailDTO({
    listId,
    inventoryIdHint: input.inventoryIdHint,
  });

  return {
    dto,
  };
}