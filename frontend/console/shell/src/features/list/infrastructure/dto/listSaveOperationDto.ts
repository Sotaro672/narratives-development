// frontend/console/shell/src/features/list/infrastructure/dto/listSaveOperationDto.ts

export type ListSaveOperationTypeDTO = "create" | "update";

export type ListSaveOperationStatusDTO =
  | "pending"
  | "uploading"
  | "registering_images"
  | "deleting_images"
  | "updating_list"
  | "setting_primary"
  | "completed"
  | "failed_retryable"
  | "failed_fatal"
  | "compensating"
  | "compensated";

export type ListSaveOperationListStatusDTO =
  | "listing"
  | "suspended";

export type ListSaveOperationListPriceRowDTO = {
  modelId: string;
  price: number;
};

export type ListSaveOperationTargetListDTO = {
  id?: string;
  readableId?: string;
  status?: ListSaveOperationListStatusDTO;
  assigneeId: string;
  title: string;
  inventoryId: string;
  imageId?: string;
  description: string;
  prices: ListSaveOperationListPriceRowDTO[];
  createdBy: string;
  createdAt?: string;
  updatedBy?: string;
  updatedAt?: string;
};

export type ListSaveOperationImageDTO = {
  imageId: string;
  url: string;
  storagePath: string;
  displayOrder: number;
};

export type ListSaveOperationPreviousImageDTO = {
  id: string;
  listId: string;
  url: string;
  displayOrder: number;
  createdAt: string;
  createdBy?: string;
  updatedAt?: string;
  updatedBy?: string;
};

export type StartListSaveOperationRequestDTO = {
  operationId?: string;
  idempotencyKey: string;
  listId: string;
  type: ListSaveOperationTypeDTO;
  targetList: ListSaveOperationTargetListDTO;
  newImages: ListSaveOperationImageDTO[];
  deleteImageIds: string[];

  /**
   * undefined: 現在のprimary imageを維持する。
   * 空文字: primary imageを解除する。
   * imageId: 指定した画像をprimary imageにする。
   */
  primaryImageId?: string;

  maxRetries?: number;
};

export type ListSaveOperationPayloadDTO = {
  targetList: ListSaveOperationTargetListDTO;
  previousList?: ListSaveOperationTargetListDTO | null;
  newImages: ListSaveOperationImageDTO[];
  deleteImageIds: string[];
  previousImages: ListSaveOperationPreviousImageDTO[];
  primaryImageId: string;
  previousPrimaryImageId: string;
};

export type ListSaveOperationProgressDTO = {
  uploadedImageIds: string[];
  registeredImageIds: string[];
  deletedImageIds: string[];
  compensatedStoragePaths: string[];
  listUpdated: boolean;
  primaryImageUpdated: boolean;
};

export type ListSaveOperationDTO = {
  id: string;
  idempotencyKey: string;
  listId: string;
  type: ListSaveOperationTypeDTO;
  status: ListSaveOperationStatusDTO;
  resumeStatus?: ListSaveOperationStatusDTO | "";
  payload: ListSaveOperationPayloadDTO;
  progress: ListSaveOperationProgressDTO;
  retryCount: number;
  maxRetries: number;
  lastError: string;
  version: number;
  createdAt: string;
  updatedAt: string;
  failedAt?: string | null;
  completedAt?: string | null;
  compensatedAt?: string | null;
};