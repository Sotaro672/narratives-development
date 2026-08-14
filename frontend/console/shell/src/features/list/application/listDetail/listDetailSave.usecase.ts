// frontend/console/shell/src/features/list/application/listDetail/listDetailSave.usecase.ts

import type {
  ListDetailDTO,
  ListSaveOperationDTO,
  ListSaveOperationListPriceRowDTO,
  ListSaveOperationTargetListDTO,
} from "../../infrastructure/dto";
import { startListSaveOperationHTTP } from "../../infrastructure/repository";
import type { ListStatus } from "../../../../shared/types/list";
import { loadListDetailDTO } from "../listDetailService";
import {
  deleteListImageFromFirebaseStorage,
  uploadListImageToFirebaseStorage,
} from "../../infrastructure/firebase/listImageStorage";

export type SaveListDetailDraftImage = {
  id?: string;
  url: string;
  isNew: boolean;
  file?: File;
};

export type SaveListDetailChangesInput = {
  listId: string;
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
  dto: ListDetailDTO;
};

type UploadedDraftImageItem = {
  draftIndex: number;
  imageId: string;
  url: string;
  storagePath: string;
  displayOrder: number;
};

type CurrentImageItem = {
  id: string;
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
  return Boolean(image?.isNew && image.file);
}

function getCurrentImages(
  currentDTO: ListDetailDTO,
): CurrentImageItem[] {
  return currentDTO.images.map((image) => ({
    id: image.id,
  }));
}

function collectRemovedImages(args: {
  currentImages: CurrentImageItem[];
  draftImages: SaveListDetailDraftImage[];
}): CurrentImageItem[] {
  const keptImageIds = new Set<string>();

  for (const image of args.draftImages) {
    if (image.isNew) {
      continue;
    }

    if (!image.id) {
      throw new Error("existing_image_id_unavailable");
    }

    keptImageIds.add(image.id);
  }

  return args.currentImages.filter(
    (current) => !keptImageIds.has(current.id),
  );
}

function resolveListStatus(
  inputStatus: ListStatus | undefined,
  currentStatus: ListStatus,
): ListStatus {
  return inputStatus ?? currentStatus;
}

function normalizePriceRows(
  rows: any[],
): ListSaveOperationListPriceRowDTO[] {
  return rows.map((row, index) => {
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
  title: string;
  description: string;
  status?: ListStatus;
  assigneeId?: string;
  updatedBy: string;
  priceRows: any[];
}): ListSaveOperationTargetListDTO {
  const title = args.title;
  const description = args.description;
  const assigneeId = args.assigneeId ?? args.currentDTO.assigneeId;
  const inventoryId = args.currentDTO.inventoryId;
  const createdBy = args.currentDTO.createdBy;
  const createdAt = args.currentDTO.createdAt;

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
    status: resolveListStatus(args.status, args.currentDTO.status),
    assigneeId,
    title,
    inventoryId,
    imageId: args.currentDTO.primaryImageId,
    description,
    prices: normalizePriceRows(args.priceRows),
    createdBy,
    createdAt,
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
        url: uploaded.url,
        storagePath: uploaded.objectPath,
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

  if (!selected) {
    throw new Error("primary_image_unavailable");
  }

  if (selected.isNew) {
    const uploaded = args.uploadedItems.find(
      (item) => item.draftIndex === selectedIndex,
    );

    if (!uploaded?.imageId) {
      throw new Error("primary_image_id_unavailable");
    }

    return uploaded.imageId;
  }

  if (!selected.id) {
    throw new Error("primary_image_id_unavailable");
  }

  return selected.id;
}

async function createStableListImageID(args: {
  listId: string;
  file: File;
  draftIndex: number;
}): Promise<string> {
  const source = JSON.stringify({
    listId: args.listId,
    draftIndex: args.draftIndex,
    name: args.file.name,
    size: args.file.size,
    type: args.file.type,
    lastModified: args.file.lastModified,
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
    currentUpdatedAt: args.currentDTO.updatedAt ?? "",
    targetList: args.targetList,
    newImages: args.plans.map((plan) => ({
      imageId: plan.imageId,
      displayOrder: plan.displayOrder,
      fileName: plan.file.name,
      fileSize: plan.file.size,
      fileType: plan.file.type,
      lastModified: plan.file.lastModified,
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
  const listId = input.listId.trim();

  if (!listId) {
    throw new Error("invalid_list_id");
  }

  if (!input.currentDTO) {
    throw new Error("list_detail_not_loaded");
  }

  const updatedBy = input.updatedBy.trim();

  if (!updatedBy) {
    throw new Error("invalid_list_updated_by");
  }

  const draftImages = input.draftImages;
  const currentImages = getCurrentImages(input.currentDTO);

  const targetList = buildTargetList({
    listId,
    currentDTO: input.currentDTO,
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

  const deleteImageIds = removedImages.map(
    (image) => image.id,
  );

  const provisionalUploadedItems: UploadedDraftImageItem[] =
    uploadPlans.map((plan) => ({
      draftIndex: plan.draftIndex,
      imageId: plan.imageId,
      url: "",
      storagePath: "",
      displayOrder: plan.displayOrder,
    }));

  const primaryImageId = resolvePrimaryImageID({
    draftImages,
    mainImageIndex: input.mainImageIndex,
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
  });

  return {
    dto,
  };
}