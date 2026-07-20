// frontend/console/list/src/application/listDetail/listDetailSave.usecase.ts
import {
  deleteListImageHTTP,
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
  /**
   * Existing image id.
   *
   * Existing backend DTOs may expose either id or imageId, so this type accepts both.
   * New local images usually do not have either until they are uploaded.
   */
  id?: string;
  imageId?: string;
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
  displayOrder: number;
};
type CurrentImageItem = {
  imageId: string;
  url: string;
  displayOrder: number | null;
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
function normalizeImageID(value: unknown): string {
  return String(value ?? "").trim();
}
function normalizeURL(value: unknown): string {
  return String(value ?? "").trim();
}
function resolveDraftImageID(
  image: SaveListDetailDraftImage | undefined,
): string {
  return normalizeImageID(image?.imageId || image?.id);
}
function normalizeCurrentImages(currentDTO: any): CurrentImageItem[] {
  const images = Array.isArray(currentDTO?.images) ? currentDTO.images : [];
  if (images.length > 0) {
    return images
      .map((img: any, index: number) => {
        const imageId = normalizeImageID(img?.imageId || img?.id);
        const url = normalizeURL(img?.url);
        if (!imageId || !url) return null;
        const displayOrderRaw = img?.displayOrder;
        const displayOrder =
          displayOrderRaw === null || displayOrderRaw === undefined
            ? index
            : Number(displayOrderRaw);
        return {
          imageId,
          url,
          displayOrder: Number.isFinite(displayOrder) ? displayOrder : index,
        };
      })
      .filter(Boolean) as CurrentImageItem[];
  }
  const primaryImageId = normalizeImageID(currentDTO?.imageId);
  const imageUrls = Array.isArray(currentDTO?.imageUrls)
    ? currentDTO.imageUrls.map(normalizeURL).filter(Boolean)
    : [];
  return imageUrls
    .map((url: string, index: number) => {
      // imageUrls だけだと 2枚目以降の imageId は復元できない。
      // primary image だけは currentDTO.imageId から復元できる。
      return {
        imageId: index === 0 ? primaryImageId : "",
        url,
        displayOrder: index,
      };
    })
    .filter((img: CurrentImageItem) => Boolean(img.url));
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
      url = normalizeURL(uploaded.url);
    } else if (!image?.isNew) {
      url = normalizeURL(image?.url);
    }
    if (!url || seen.has(url)) return;
    seen.add(url);
    out.push(url);
  });
  return out;
}
function collectRemovedImages(args: {
  currentImages: CurrentImageItem[];
  draftImages: SaveListDetailDraftImage[];
}): CurrentImageItem[] {
  const keptImageIDs = new Set<string>();
  const keptUrls = new Set<string>();
  for (const image of args.draftImages) {
    if (image?.isNew) continue;
    const imageId = resolveDraftImageID(image);
    const url = normalizeURL(image?.url);
    if (imageId) keptImageIDs.add(imageId);
    if (url) keptUrls.add(url);
  }
  return args.currentImages.filter((current) => {
    if (!current.imageId) return false;
    if (keptImageIDs.has(current.imageId)) {
      return false;
    }
    // 既存 draft 側に imageId がまだ入っていない場合の fallback。
    // ただし URL 一致 fallback は duplicate URL がない前提。
    if (current.url && keptUrls.has(current.url)) {
      return false;
    }
    return true;
  });
}
function resolvePrimaryImageId(args: {
  selectedUrl: string;
  currentImages: CurrentImageItem[];
  uploadedItems: SavedDraftImageItem[];
}): string {
  const selectedUrl = normalizeURL(args.selectedUrl);
  if (!selectedUrl) return "";
  const uploadedPrimary = args.uploadedItems.find(
    (item) => normalizeURL(item.url) === selectedUrl,
  );
  if (uploadedPrimary?.imageId) {
    return normalizeImageID(uploadedPrimary.imageId);
  }
  const currentPrimary = args.currentImages.find(
    (item) => normalizeURL(item.url) === selectedUrl,
  );
  return normalizeImageID(currentPrimary?.imageId);
}
export async function saveListDetailChanges(
  input: SaveListDetailChangesInput,
): Promise<SaveListDetailChangesResult> {
  const listId = String(input.listId ?? "").trim();
  if (!listId) throw new Error("invalid_list_id");
  const updatedBy = String(input.updatedBy ?? "").trim() || "system";
  const draftImages = normalizeDraftImages(input.draftImages);
  const currentDTO: any = input.currentDTO;
  const currentImages = normalizeCurrentImages(currentDTO);
  const uploadedItems: SavedDraftImageItem[] = [];
  for (let index = 0; index < draftImages.length; index++) {
    const image = draftImages[index];
    if (!isNewDraftImageWithFile(image)) continue;
    const file = image.file;
    const displayOrder = index;
    const uploaded = await uploadListImageToFirebaseStorage({
      listId,
      file,
    });
    const saved = await saveListImageFromFirebaseStorageHTTP({
      listId,
      id: uploaded.imageId,
      url: uploaded.url,
      displayOrder,
      createdBy: updatedBy,
      createdAt: new Date().toISOString(),
    });
    const savedUrl = normalizeURL((saved as any)?.url) || uploaded.url;
    uploadedItems.push({
      draftIndex: index,
      imageId: uploaded.imageId,
      url: savedUrl,
      displayOrder,
    });
  }
  const removedImages = collectRemovedImages({
    currentImages,
    draftImages,
  });
  for (const image of removedImages) {
    await deleteListImageHTTP({
      listId,
      imageId: image.imageId,
    });
  }
  const afterUrls = buildAfterUrls({
    draftImages,
    uploadedItems,
  });
  const selectedUrl = normalizeURL(
    afterUrls[input.mainImageIndex] ?? afterUrls[0],
  );
  if (selectedUrl) {
    const primaryImageId = resolvePrimaryImageId({
      selectedUrl,
      currentImages,
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
    priceRows: Array.isArray(input.draftPriceRows)
      ? input.draftPriceRows
      : [],
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