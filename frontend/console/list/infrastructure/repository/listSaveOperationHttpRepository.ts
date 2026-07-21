// frontend/console/list/infrastructure/repository/listSaveOperationHttpRepository.ts
import { API_BASE } from "../../../shell/src/shared/http/apiBase";
import type {
  ListSaveOperationDTO,
  StartListSaveOperationRequestDTO,
} from "../dto/listSaveOperationDto";
import { requestJSON } from "../http/httpClient";
export async function startListSaveOperationHTTP(
  input: StartListSaveOperationRequestDTO,
): Promise<ListSaveOperationDTO> {
  const payload = normalizeStartListSaveOperationRequest(input);
  const json = await requestJSON({
    method: "POST",
    path: "/lists/save-operations",
    body: payload,
    debug: {
      tag: "POST /lists/save-operations",
      url: `${API_BASE}/lists/save-operations`,
      method: "POST",
      body: payload,
    },
  });
  return json as ListSaveOperationDTO;
}
export async function fetchListSaveOperationHTTP(
  operationId: string,
): Promise<ListSaveOperationDTO> {
  const normalizedOperationId = requireListSaveOperationId(operationId);
  const encodedOperationId = encodeURIComponent(normalizedOperationId);
  const json = await requestJSON({
    method: "GET",
    path: `/lists/save-operations/${encodedOperationId}`,
    debug: {
      tag: `GET /lists/save-operations/${normalizedOperationId}`,
      url: `${API_BASE}/lists/save-operations/${encodedOperationId}`,
      method: "GET",
    },
  });
  return json as ListSaveOperationDTO;
}
export async function retryListSaveOperationHTTP(
  operationId: string,
): Promise<ListSaveOperationDTO> {
  const normalizedOperationId = requireListSaveOperationId(operationId);
  const encodedOperationId = encodeURIComponent(normalizedOperationId);
  const json = await requestJSON({
    method: "POST",
    path: `/lists/save-operations/${encodedOperationId}/retry`,
    debug: {
      tag: `POST /lists/save-operations/${normalizedOperationId}/retry`,
      url: `${API_BASE}/lists/save-operations/${encodedOperationId}/retry`,
      method: "POST",
    },
  });
  return json as ListSaveOperationDTO;
}
export async function compensateListSaveOperationHTTP(
  operationId: string,
): Promise<ListSaveOperationDTO> {
  const normalizedOperationId = requireListSaveOperationId(operationId);
  const encodedOperationId = encodeURIComponent(normalizedOperationId);
  const json = await requestJSON({
    method: "POST",
    path: `/lists/save-operations/${encodedOperationId}/compensate`,
    debug: {
      tag: `POST /lists/save-operations/${normalizedOperationId}/compensate`,
      url: `${API_BASE}/lists/save-operations/${encodedOperationId}/compensate`,
      method: "POST",
    },
  });
  return json as ListSaveOperationDTO;
}
function normalizeStartListSaveOperationRequest(
  input: StartListSaveOperationRequestDTO,
): StartListSaveOperationRequestDTO {
  if (!input || typeof input !== "object") {
    throw new Error("invalid_list_save_operation_input");
  }
  const idempotencyKey = requireNonEmptyString(
    input.idempotencyKey,
    "invalid_idempotency_key",
  );
  const listId = requireNonEmptyString(input.listId, "invalid_list_id");
  if (input.type !== "create" && input.type !== "update") {
    throw new Error("invalid_list_save_operation_type");
  }
  if (!input.targetList || typeof input.targetList !== "object") {
    throw new Error("invalid_list_save_operation_target_list");
  }
  const operationId = normalizeOptionalString(input.operationId);
  const primaryImageId =
    input.primaryImageId === undefined
      ? undefined
      : String(input.primaryImageId).trim();
  const maxRetries =
    input.maxRetries === undefined
      ? undefined
      : normalizeMaxRetries(input.maxRetries);
  const newImages = Array.isArray(input.newImages)
    ? input.newImages.map((image, index) => {
        const imageId = requireNonEmptyString(
          image?.imageId,
          `invalid_new_image_id_${index}`,
        );
        const url = requireNonEmptyString(
          image?.url,
          `invalid_new_image_url_${index}`,
        );
        const storagePath = requireNonEmptyString(
          image?.storagePath,
          `invalid_new_image_storage_path_${index}`,
        ).replace(/^\/+/, "");
        const displayOrder = Number(image?.displayOrder);
        if (!Number.isInteger(displayOrder) || displayOrder < 0) {
          throw new Error(`invalid_new_image_display_order_${index}`);
        }
        return {
          imageId,
          url,
          storagePath,
          displayOrder,
        };
      })
    : [];
  const deleteImageIds = Array.isArray(input.deleteImageIds)
    ? input.deleteImageIds.map((imageId, index) =>
        requireNonEmptyString(
          imageId,
          `invalid_delete_image_id_${index}`,
        ),
      )
    : [];
  return {
    operationId,
    idempotencyKey,
    listId,
    type: input.type,
    targetList: input.targetList,
    newImages,
    deleteImageIds,
    primaryImageId,
    maxRetries,
  };
}
function requireListSaveOperationId(operationId: string): string {
  const value = requireNonEmptyString(
    operationId,
    "invalid_list_save_operation_id",
  );
  if (
    value.includes("/") ||
    value.includes("://") ||
    /[\r\n\u0000]/u.test(value)
  ) {
    throw new Error("invalid_list_save_operation_id");
  }
  return value;
}
function requireNonEmptyString(value: unknown, errorCode: string): string {
  if (typeof value !== "string") {
    throw new Error(errorCode);
  }
  const normalized = value.trim();
  if (!normalized) {
    throw new Error(errorCode);
  }
  return normalized;
}
function normalizeOptionalString(value: unknown): string | undefined {
  if (typeof value !== "string") {
    return undefined;
  }
  const normalized = value.trim();
  return normalized || undefined;
}
function normalizeMaxRetries(value: number): number {
  const normalized = Number(value);
  if (!Number.isInteger(normalized) || normalized < 0) {
    throw new Error("invalid_list_save_operation_max_retries");
  }
  return normalized;
}