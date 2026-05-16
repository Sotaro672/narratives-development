// frontend/console/list/src/application/listDetail/listDetailSave.usecase.ts

import {
  saveListImageFromFirebaseStorageHTTP,
  setListPrimaryImageHTTP,
} from "../../infrastructure/repository";

import type { ListDTO } from "../../infrastructure/dto";

import {
  loadListDetailDTO,
  updateListDetailDTO,
  type ListDetailDTO,
} from "../listDetailService";

import { uploadListImageToFirebaseStorage } from "../../infrastructure/firebase/listImageStorage";

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
  currentImageId?: string;
  selectedUrl: string;
  currentImageUrls: string[];
  uploadedItems: SavedDraftImageItem[];
}): string {
  const selectedUrl = String(args.selectedUrl ?? "").trim();

  if (!selectedUrl) return "";

  const uploadedPrimary = args.uploadedItems.find(
    (item) => String(item.url ?? "").trim() === selectedUrl,
  );

  if (uploadedPrimary?.imageId) {
    return String(uploadedPrimary.imageId ?? "").trim();
  }

  const currentImageId = String(args.currentImageId ?? "").trim();
  const currentImageUrls = Array.isArray(args.currentImageUrls)
    ? args.currentImageUrls
    : [];

  if (
    currentImageId &&
    currentImageUrls.some((url) => String(url ?? "").trim() === selectedUrl)
  ) {
    return currentImageId;
  }

  return "";
}

export async function saveListDetailChanges(
  input: SaveListDetailChangesInput,
): Promise<SaveListDetailChangesResult> {
  const listId = String(input.listId ?? "").trim();
  if (!listId) throw new Error("invalid_list_id");

  const updatedBy = String(input.updatedBy ?? "").trim() || "system";
  const draftImages = normalizeDraftImages(input.draftImages);

  const currentDTO: any = input.currentDTO;
  const currentImageId = String(currentDTO?.imageId ?? "").trim();
  const currentImageUrls = Array.isArray(currentDTO?.imageUrls)
    ? currentDTO.imageUrls
        .map((url: unknown) => String(url ?? "").trim())
        .filter(Boolean)
    : [];

  const uploadedItems: SavedDraftImageItem[] = [];

  for (let index = 0; index < draftImages.length; index++) {
    const image = draftImages[index];

    if (!isNewDraftImageWithFile(image)) continue;

    const file = image.file;
    const displayOrder = currentImageUrls.length + uploadedItems.length;

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

  const selectedUrl = String(afterUrls[input.mainImageIndex] ?? "").trim();

  if (selectedUrl) {
    const primaryImageId = resolvePrimaryImageId({
      currentImageId,
      currentImageUrls,
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