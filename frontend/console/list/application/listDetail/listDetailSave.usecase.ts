// frontend/console/list/src/application/listDetail/listDetailSave.usecase.ts
import type {
  ListDTO,
  ListSaveOperationDTO,
  ListSaveOperationListPriceRowDTO,
  ListSaveOperationTargetListDTO,
} from "../../infrastructure/dto";
import { startListSaveOperationHTTP } from "../../infrastructure/repository";
import type { ListStatus } from "../../domain/list";
import {
  loadListDetailDTO,
  type ListDetailDTO,
} from "../listDetailService";
import {
  deleteListImageFromFirebaseStorage,
  uploadListImageToFirebaseStorage,
} from "../../infrastructure/firebase/listImageStorage";
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
  status?: ListStatus;
  assigneeId?: string;
  updatedBy: string;
  draftPriceRows: any[];
  draftImages: SaveListDetailDraftImage[];
  mainImageIndex: number;
};
export type SaveListDetailChangesResult = {
  dto: ListDTO;
};
type UploadedDraftImageItem = {
  draftIndex: number;
  imageId: string;
  url: string;
  storagePath: string;
  displayOrder: number;
};
type CurrentImageItem = {
  imageId: string;
  url: string;
  displayOrder: number;
};
type NewImageUploadPlan = {
  draftIndex: number;
  imageId: string;
  file: File;
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
function normalizeCurrentImages(currentDTO: ListDetailDTO | null): CurrentImageItem[] {
  const source = currentDTO as any;
  const images = Array.isArray(source?.images) ? source.images : [];
  if (images.length > 0) {
    return images
      .map((image: any, index: number): CurrentImageItem | null => {
        const imageId = normalizeImageID(image?.imageId || image?.id);
        const url = normalizeURL(image?.url);
        if (!url) {
          return null;
        }
        const rawDisplayOrder = Number(image?.displayOrder);
        return {
          imageId,
          url,
          displayOrder: Number.isInteger(rawDisplayOrder)
            ? rawDisplayOrder
            : index,
        };
      })
      .filter((image: CurrentImageItem | null): image is CurrentImageItem =>
        Boolean(image),
      );
  }
  const primaryImageId = normalizeImageID(source?.imageId);
  const imageUrls = Array.isArray(source?.imageUrls)
    ? source.imageUrls.map(normalizeURL).filter(Boolean)
    : [];
  return imageUrls.map((url: string, index: number) => ({
    imageId: index === 0 ? primaryImageId : "",
    url,
    displayOrder: index,
  }));
}
function collectRemovedImages(args: {
  currentImages: CurrentImageItem[];
  draftImages: SaveListDetailDraftImage[];
}): CurrentImageItem[] {
  const keptImageIDs = new Set<string>();
  const keptURLs = new Set<string>();
  for (const image of args.draftImages) {
    if (image?.isNew) {
      continue;
    }
    const imageId = resolveDraftImageID(image);
    const url = normalizeURL(image?.url);
    if (imageId) {
      keptImageIDs.add(imageId);
    }
    if (url) {
      keptURLs.add(url);
    }
  }
  return args.currentImages.filter((current) => {
    if (!current.imageId) {
      return false;
    }
    if (keptImageIDs.has(current.imageId)) {
      return false;
    }
    if (current.url && keptURLs.has(current.url)) {
      return false;
    }
    return true;
  });
}
function normalizeListStatus(
  inputStatus: ListStatus | undefined,
  currentStatus: unknown,
): "listing" | "suspended" {
  const status = String(inputStatus ?? currentStatus ?? "").trim();
  if (status === "listing" || status === "suspended") {
    return status;
  }
  throw new Error("invalid_list_status");
}
function normalizePriceRows(
  rows: any[] | null | undefined,
): ListSaveOperationListPriceRowDTO[] {
  const source = Array.isArray(rows) ? rows : [];
  return source.map((row, index) => {
    const modelId = String(row?.modelId ?? "").trim();
    const price = Number(row?.price);
    if (!modelId) {
      throw new Error(`invalid_price_model_id_${index}`);
    }
    if (!Number.isInteger(price) || price < 0 || price > 10_000_000) {
      throw new Error(`invalid_price_${index}`);
    }
    return {
      modelId,
      price,
    };
  });
}
function buildTargetList(args: {
  listId: string;
  currentDTO: ListDetailDTO;
  inventoryIdHint?: string;
  title: string;
  description: string;
  status?: ListStatus;
  assigneeId?: string;
  updatedBy: string;
  priceRows: any[];
}): ListSaveOperationTargetListDTO {
  const title = String(args.title ?? "");
  const description = String(args.description ?? "");
  const assigneeId = String(
    args.assigneeId ?? args.currentDTO.assigneeId ?? "",
  ).trim();
  const inventoryId = String(
    args.currentDTO.inventoryId ?? args.inventoryIdHint ?? "",
  ).trim();
  const createdBy = String(
    args.currentDTO.createdBy ?? args.updatedBy,
  ).trim();
  const createdAt = String(args.currentDTO.createdAt ?? "").trim();
  if (!title.trim()) {
    throw new Error("invalid_list_title");
  }
  if (!description.trim()) {
    throw new Error("invalid_list_description");
  }
  if (!assigneeId) {
    throw new Error("invalid_list_assignee_id");
  }
  if (!inventoryId) {
    throw new Error("invalid_inventory_id");
  }
  if (!createdBy) {
    throw new Error("invalid_list_created_by");
  }
  return {
    id: args.listId,
    status: normalizeListStatus(args.status, args.currentDTO.status),
    assigneeId,
    title,
    inventoryId,
    imageId: normalizeImageID(args.currentDTO.imageId),
    description,
    prices: normalizePriceRows(args.priceRows),
    createdBy,
    createdAt: createdAt || undefined,
    updatedBy: args.updatedBy,
    updatedAt: new Date().toISOString(),
  };
}
async function buildNewImageUploadPlans(args: {
  listId: string;
  draftImages: SaveListDetailDraftImage[];
}): Promise<NewImageUploadPlan[]> {
  const plans: NewImageUploadPlan[] = [];
  for (let index = 0; index < args.draftImages.length; index++) {
    const image = args.draftImages[index];
    if (!isNewDraftImageWithFile(image)) {
      continue;
    }
    const imageId = await createStableListImageID({
      listId: args.listId,
      file: image.file,
      draftIndex: index,
    });
    plans.push({
      draftIndex: index,
      imageId,
      file: image.file,
      displayOrder: index,
    });
  }
  return plans;
}
async function uploadNewImages(args: {
  listId: string;
  plans: NewImageUploadPlan[];
}): Promise<UploadedDraftImageItem[]> {
  const uploadedItems: UploadedDraftImageItem[] = [];
  try {
    for (const plan of args.plans) {
      const uploaded = await uploadListImageToFirebaseStorage({
        listId: args.listId,
        imageId: plan.imageId,
        file: plan.file,
      });
      uploadedItems.push({
        draftIndex: plan.draftIndex,
        imageId: uploaded.imageId,
        url: normalizeURL(uploaded.url),
        storagePath: String(uploaded.objectPath ?? "").trim(),
        displayOrder: plan.displayOrder,
      });
    }
    return uploadedItems;
  } catch (uploadError) {
    const cleanupErrors = await cleanupUploadedImages(uploadedItems);
    if (cleanupErrors.length > 0) {
      throw new AggregateError(
        [uploadError, ...cleanupErrors],
        "list_image_upload_and_compensation_failed",
      );
    }
    throw uploadError;
  }
}
async function cleanupUploadedImages(
  uploadedItems: UploadedDraftImageItem[],
): Promise<unknown[]> {
  const results = await Promise.allSettled(
    uploadedItems.map((item) =>
      deleteListImageFromFirebaseStorage({
        storagePath: item.storagePath,
      }),
    ),
  );
  return results
    .filter(
      (result): result is PromiseRejectedResult =>
        result.status === "rejected",
    )
    .map((result) => result.reason);
}
function resolvePrimaryImageID(args: {
  draftImages: SaveListDetailDraftImage[];
  mainImageIndex: number;
  currentImages: CurrentImageItem[];
  uploadedItems: UploadedDraftImageItem[];
}): string {
  if (args.draftImages.length === 0) {
    return "";
  }
  const selectedIndex =
    Number.isInteger(args.mainImageIndex) &&
    args.mainImageIndex >= 0 &&
    args.mainImageIndex < args.draftImages.length
      ? args.mainImageIndex
      : 0;
  const selected = args.draftImages[selectedIndex];
  const uploaded = args.uploadedItems.find(
    (item) => item.draftIndex === selectedIndex,
  );
  if (uploaded?.imageId) {
    return uploaded.imageId;
  }
  const directImageId = resolveDraftImageID(selected);
  if (directImageId) {
    return directImageId;
  }
  const selectedURL = normalizeURL(selected?.url);
  const current = args.currentImages.find(
    (image) => normalizeURL(image.url) === selectedURL,
  );
  if (current?.imageId) {
    return current.imageId;
  }
  throw new Error("primary_image_id_unavailable");
}
async function createStableListImageID(args: {
  listId: string;
  file: File;
  draftIndex: number;
}): Promise<string> {
  const source = JSON.stringify({
    listId: args.listId,
    draftIndex: args.draftIndex,
    name: String(args.file.name ?? ""),
    size: Number(args.file.size ?? 0),
    type: String(args.file.type ?? ""),
    lastModified: Number(args.file.lastModified ?? 0),
  });
  const digest = await hashText(source);
  return `img_${digest.slice(0, 48)}`;
}
async function createIdempotencyKey(args: {
  listId: string;
  currentDTO: ListDetailDTO;
  targetList: ListSaveOperationTargetListDTO;
  plans: NewImageUploadPlan[];
  deleteImageIds: string[];
  primaryImageId: string;
}): Promise<string> {
  const fingerprint = JSON.stringify({
    listId: args.listId,
    currentUpdatedAt: String(args.currentDTO.updatedAt ?? ""),
    targetList: args.targetList,
    newImages: args.plans.map((plan) => ({
      imageId: plan.imageId,
      displayOrder: plan.displayOrder,
      fileName: String(plan.file.name ?? ""),
      fileSize: Number(plan.file.size ?? 0),
      fileType: String(plan.file.type ?? ""),
      lastModified: Number(plan.file.lastModified ?? 0),
    })),
    deleteImageIds: [...args.deleteImageIds].sort(),
    primaryImageId: args.primaryImageId,
  });
  return `list-save-${await hashText(fingerprint)}`;
}
async function hashText(value: string): Promise<string> {
  if (
    typeof crypto !== "undefined" &&
    crypto.subtle &&
    typeof TextEncoder !== "undefined"
  ) {
    const bytes = new TextEncoder().encode(value);
    const digest = await crypto.subtle.digest("SHA-256", bytes);
    return Array.from(new Uint8Array(digest))
      .map((item) => item.toString(16).padStart(2, "0"))
      .join("");
  }
  let first = 2166136261;
  let second = 2246822519;
  for (let index = 0; index < value.length; index++) {
    const code = value.charCodeAt(index);
    first ^= code;
    first = Math.imul(first, 16777619);
    second ^= code + index;
    second = Math.imul(second, 3266489917);
  }
  return (
    (first >>> 0).toString(16).padStart(8, "0") +
    (second >>> 0).toString(16).padStart(8, "0")
  ).repeat(4);
}
function assertCompletedOperation(
  operation: ListSaveOperationDTO,
): void {
  if (operation.status === "completed") {
    return;
  }
  const detail = String(operation.lastError ?? "").trim();
  switch (operation.status) {
    case "failed_retryable":
      throw new Error(
        detail ||
          `list_save_operation_retry_scheduled:${operation.id}`,
      );
    case "failed_fatal":
      throw new Error(
        detail ||
          `list_save_operation_failed_fatal:${operation.id}`,
      );
    case "compensated":
      throw new Error(
        detail ||
          `list_save_operation_compensated:${operation.id}`,
      );
    case "compensating":
      throw new Error(
        detail ||
          `list_save_operation_compensating:${operation.id}`,
      );
    default:
      throw new Error(
        detail ||
          `list_save_operation_incomplete:${operation.id}:${operation.status}`,
      );
  }
}
export async function saveListDetailChanges(
  input: SaveListDetailChangesInput,
): Promise<SaveListDetailChangesResult> {
  const listId = String(input.listId ?? "").trim();
  if (!listId) {
    throw new Error("invalid_list_id");
  }
  if (!input.currentDTO) {
    throw new Error("list_detail_not_loaded");
  }
  const updatedBy = String(input.updatedBy ?? "").trim() || "system";
  const draftImages = normalizeDraftImages(input.draftImages);
  const currentImages = normalizeCurrentImages(input.currentDTO);
  const targetList = buildTargetList({
    listId,
    currentDTO: input.currentDTO,
    inventoryIdHint: input.inventoryIdHint,
    title: input.title,
    description: input.description,
    status: input.status,
    assigneeId: input.assigneeId,
    updatedBy,
    priceRows: input.draftPriceRows,
  });
  const uploadPlans = await buildNewImageUploadPlans({
    listId,
    draftImages,
  });
  const removedImages = collectRemovedImages({
    currentImages,
    draftImages,
  });
  const deleteImageIds = removedImages.map((image) => image.imageId);
  const provisionalUploadedItems = uploadPlans.map((plan) => ({
    draftIndex: plan.draftIndex,
    imageId: plan.imageId,
    url: "",
    storagePath: "",
    displayOrder: plan.displayOrder,
  }));
  const primaryImageId = resolvePrimaryImageID({
    draftImages,
    mainImageIndex: input.mainImageIndex,
    currentImages,
    uploadedItems: provisionalUploadedItems,
  });
  const idempotencyKey = await createIdempotencyKey({
    listId,
    currentDTO: input.currentDTO,
    targetList,
    plans: uploadPlans,
    deleteImageIds,
    primaryImageId,
  });
  const uploadedItems = await uploadNewImages({
    listId,
    plans: uploadPlans,
  });
  const operation = await startListSaveOperationHTTP({
    idempotencyKey,
    listId,
    type: "update",
    targetList,
    newImages: uploadedItems.map((image) => ({
      imageId: image.imageId,
      url: image.url,
      storagePath: image.storagePath,
      displayOrder: image.displayOrder,
    })),
    deleteImageIds,
    primaryImageId,
    maxRetries: 3,
  });
  assertCompletedOperation(operation);
  const dto = await loadListDetailDTO({
    listId,
    inventoryIdHint: input.inventoryIdHint,
  });
  return {
    dto,
  };
}