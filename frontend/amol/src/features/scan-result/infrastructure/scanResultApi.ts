// frontend/amol/src/features/scan-result/infrastructure/scanResultApi.ts 
 
// frontend/amol/src/features/scan-result/infrastructure/scanResultApi.ts 
 
import { requestJson } from "../../../lib/http"; 
import { getOptionalAuthHeaders } from "../../../lib/authHeaders"; 
import { HttpError } from "../../../lib/http/httpError"; 
 
import type { 
  ProductBlueprintReview, 
  ProductBlueprintReviewPage, 
} from "../../shared/types/review"; 
 
import type { 
  MallPreviewResponse, 
  MallScanTransferResponse, 
  PreviewState, 
} from "../../shared/types/scanResult"; 
 
export const RETURN_IN_PROGRESS_OPENED_ERROR_CODE = 
  "return_in_progress_opened" as const; 
 
const RETURN_IN_PROGRESS_OPENED_MESSAGE = 
  "返品申請中の商品で開封が確認されました。返品区分を「開封後の返品」に変更しました。Token Transfer は実行していません。"; 
 
export type WalletResolvedTokenResponse = { 
  productId: string; 
  brandId: string; 
  brandName: string; 
  productBlueprintId: string; 
  productName: string; 
  metadataUri: string; 
  assetId: string; 
}; 
 
type ReturnInProgressOpenedErrorParams = { 
  message: string; 
  avatarId: string; 
  productId: string; 
  matchedOrderId: string; 
  matchedItemIndex: number; 
}; 
 
export class ReturnInProgressOpenedError extends Error { 
  readonly code = 
    RETURN_IN_PROGRESS_OPENED_ERROR_CODE; 
 
  readonly status = 409; 
 
  readonly avatarId: string; 
 
  readonly productId: string; 
 
  readonly matchedOrderId: string; 
 
  readonly matchedItemIndex: number; 
 
  constructor( 
    params: ReturnInProgressOpenedErrorParams, 
  ) { 
    super(params.message); 
 
    this.name = 
      "ReturnInProgressOpenedError"; 
 
    this.avatarId = 
      params.avatarId; 
 
    this.productId = 
      params.productId; 
 
    this.matchedOrderId = 
      params.matchedOrderId; 
 
    this.matchedItemIndex = 
      params.matchedItemIndex; 
  } 
} 
 
export function isReturnInProgressOpenedError( 
  caught: unknown, 
): caught is ReturnInProgressOpenedError { 
  return caught instanceof 
    ReturnInProgressOpenedError; 
} 
 
function isRecord( 
  value: unknown, 
): value is Record<string, unknown> { 
  return ( 
    typeof value === "object" && 
    value !== null && 
    !Array.isArray(value) 
  ); 
} 
 
function unwrapErrorBody( 
  value: unknown, 
): Record<string, unknown> | null { 
  if (!isRecord(value)) { 
    return null; 
  } 
 
  if (isRecord(value.data)) { 
    return value.data; 
  } 
 
  return value; 
} 
 
function readStringValue( 
  value: unknown, 
): string { 
  return typeof value === "string" 
    ? value.trim() 
    : ""; 
} 
 
function readItemIndex( 
  value: unknown, 
): number | null { 
  if ( 
    typeof value !== "number" || 
    !Number.isInteger(value) || 
    value < 0 
  ) { 
    return null; 
  } 
 
  return value; 
} 
 
function toReturnInProgressOpenedError( 
  caught: unknown, 
  requestedProductId: string, 
): ReturnInProgressOpenedError | null { 
  if ( 
    !(caught instanceof HttpError) || 
    caught.status !== 409 
  ) { 
    return null; 
  } 
 
  const body = 
    unwrapErrorBody( 
      caught.body, 
    ); 
 
  if (!body) { 
    return null; 
  } 
 
  const errorCode = 
    readStringValue( 
      body.error, 
    ); 
 
  if ( 
    errorCode !== 
    RETURN_IN_PROGRESS_OPENED_ERROR_CODE 
  ) { 
    return null; 
  } 
 
  const matchedOrderId = 
    readStringValue( 
      body.orderId, 
    ); 
 
  const matchedItemIndex = 
    readItemIndex( 
      body.itemIndex, 
    ); 
 
  if ( 
    !matchedOrderId || 
    matchedItemIndex === null 
  ) { 
    return null; 
  } 
 
  const message = 
    readStringValue( 
      body.message, 
    ) || 
    RETURN_IN_PROGRESS_OPENED_MESSAGE; 
 
  return new ReturnInProgressOpenedError({ 
    message, 
    avatarId: 
      readStringValue( 
        body.avatarId, 
      ), 
    productId: 
      readStringValue( 
        body.productId, 
      ) || requestedProductId, 
    matchedOrderId, 
    matchedItemIndex, 
  }); 
} 
 
export async function loadPreviewState(productId: string): Promise<PreviewState> { 
  const id = productId.trim(); 
 
  if (!id) { 
    throw new Error("preview: productId is empty"); 
  } 
 
  const authHeaders = await getOptionalAuthHeaders(); 
  const path = authHeaders ? "/mall/me/preview" : "/mall/preview"; 
 
  const raw = await requestJson<MallPreviewResponse>(path, { 
    method: "GET", 
    headers: authHeaders, 
    query: { productId: id }, 
    unwrapData: true, 
    messages: { 
      requestErrorMessage: "fetchPreviewByProductId failed", 
      nonJsonErrorMessage: "fetchPreviewByProductId failed: response is not json", 
      invalidJsonErrorMessage: "fetchPreviewByProductId failed: invalid json", 
    }, 
  }); 
 
  return { 
    raw, 
    tokenIconUrlEncoded: raw.tokenBlueprintPatch?.tokenIcon ?? null, 
  }; 
} 
 
export async function transferScanPurchased(args: { 
  productId: string; 
  operationId: string; 
  headers?: HeadersInit; 
}): Promise<MallScanTransferResponse> { 
  const productId = args.productId.trim(); 
  const operationId = args.operationId.trim(); 
 
  if (!productId) { 
    throw new Error("productId is empty"); 
  } 
 
  if (!operationId) { 
    throw new Error("operationId is empty"); 
  } 
 
  const headers = new Headers(args.headers); 
  headers.set("Idempotency-Key", operationId); 
 
  try { 
    return await requestJson<MallScanTransferResponse>( 
      "/mall/me/orders/scan/transfer", 
      { 
        method: "POST", 
        auth: "required", 
        headers, 
        json: { productId }, 
        unwrapData: true, 
        messages: { 
          requestErrorMessage: "transferScanPurchased failed", 
          nonJsonErrorMessage: "transferScanPurchased failed: response is not json", 
          invalidJsonErrorMessage: "transferScanPurchased failed: invalid json", 
        }, 
      }, 
    ); 
  } catch (caught) { 
    const returnInProgressError = 
      toReturnInProgressOpenedError( 
        caught, 
        productId, 
      ); 
 
    if (returnInProgressError) { 
      throw returnInProgressError; 
    } 
 
    throw caught; 
  } 
} 
 
export async function fetchReviewsByProductBlueprintId(args: { 
  productBlueprintId: string; 
  page: number; 
  perPage: number; 
}): Promise<ProductBlueprintReviewPage> { 
  const productBlueprintId = args.productBlueprintId.trim(); 
 
  if (!productBlueprintId) { 
    throw new Error("preview review: productBlueprintId is empty"); 
  } 
 
  return requestJson<ProductBlueprintReviewPage>( 
    `/mall/catalog/product-blueprints/${encodeURIComponent(productBlueprintId)}/reviews`, 
    { 
      method: "GET", 
      auth: "none", 
      query: { 
        page: args.page, 
        perPage: args.perPage, 
      }, 
      messages: { 
        requestErrorMessage: "fetchReviewsByProductBlueprintId failed", 
        nonJsonErrorMessage: "fetchReviewsByProductBlueprintId failed: response is not json", 
        invalidJsonErrorMessage: "fetchReviewsByProductBlueprintId failed: invalid json", 
      }, 
    }, 
  ); 
} 
 
export async function createProductBlueprintReview(args: { 
  productBlueprintId: string; 
  body: string; 
  rating: number; 
  title?: string; 
  headers?: HeadersInit; 
}): Promise<ProductBlueprintReview> { 
  const productBlueprintId = args.productBlueprintId.trim(); 
  const body = args.body.trim(); 
 
  if (!productBlueprintId) { 
    throw new Error("preview review create: productBlueprintId is empty"); 
  } 
 
  if (!body) { 
    throw new Error("preview review create: body is empty"); 
  } 
 
  const rating = Math.max(1, Math.min(5, Math.trunc(args.rating))); 
  const title = args.title?.trim() || "Review"; 
 
  return requestJson<ProductBlueprintReview>( 
    `/mall/me/catalog/product-blueprints/${encodeURIComponent(productBlueprintId)}/reviews`, 
    { 
      method: "POST", 
      auth: "required", 
      headers: args.headers, 
      json: { 
        body, 
        rating, 
        title, 
      }, 
      messages: { 
        requestErrorMessage: "createProductBlueprintReview failed", 
        nonJsonErrorMessage: "createProductBlueprintReview failed: response is not json", 
        invalidJsonErrorMessage: "createProductBlueprintReview failed: invalid json", 
      }, 
    }, 
  ); 
} 
 
export async function resolveOwnedWalletTokenByAssetId( 
  assetId: string, 
  headers?: HeadersInit, 
): Promise<WalletResolvedTokenResponse> { 
  const id = assetId.trim(); 
 
  if (!id) { 
    throw new Error("assetId is empty"); 
  } 
 
  return requestJson<WalletResolvedTokenResponse>( 
    "/mall/me/wallets/tokens/resolve", 
    { 
      method: "GET", 
      auth: "required", 
      headers, 
      query: { assetId: id }, 
      messages: { 
        requestErrorMessage: "resolveOwnedWalletTokenByAssetId failed", 
        nonJsonErrorMessage: "resolveOwnedWalletTokenByAssetId failed: response is not json", 
        invalidJsonErrorMessage: "resolveOwnedWalletTokenByAssetId failed: invalid json", 
      }, 
    }, 
  ); 
} 
 
export async function isOwnedByWalletAssetId( 
  assetId: string, 
  headers?: HeadersInit, 
): Promise<boolean> { 
  const id = assetId.trim(); 
 
  if (!id) { 
    throw new Error("assetId is empty"); 
  } 
 
  await requestJson<WalletResolvedTokenResponse>( 
    "/mall/me/wallets/tokens/resolve", 
    { 
      method: "GET", 
      auth: "required", 
      headers, 
      query: { assetId: id }, 
      messages: { 
        requestErrorMessage: "isOwnedByWalletAssetId failed", 
        nonJsonErrorMessage: "isOwnedByWalletAssetId failed: response is not json", 
        invalidJsonErrorMessage: "isOwnedByWalletAssetId failed: invalid json", 
      }, 
    }, 
  ); 
 
  return true; 
} 