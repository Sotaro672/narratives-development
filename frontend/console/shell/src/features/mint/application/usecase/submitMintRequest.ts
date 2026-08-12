// frontend/console/shell/src/features/mintRequest/application/usecase/submitMintRequest.ts


import type {
  InspectionBatchDTO,
} from "../../../../shared/types/inspections";


import type {
  MintQueuedResponse,
} from "../port/MintRequestRepository";


import {
  validateMintRequestSubmit,
} from "../validator/validateMintRequestSubmit";


/**
 * Mint申請処理に必要なRepository契約。
 */
export interface SubmitMintRequestRepository {
  postMintRequest(
    productionId: string,
    tokenBlueprintId: string,
  ): Promise<
    MintQueuedResponse | null
  >;
}


export type SubmitMintRequestInput = {
  inspectionBatch:
    | InspectionBatchDTO
    | null
    | undefined;


  selectedTokenBlueprintId:
    | string
    | null
    | undefined;


  /**
   * URLパラメータ由来のproductionId。
   *
   * inspectionBatch.productionIdが存在する場合は、
   * validator側でそちらを優先する。
   */
  productionId?:
    | string
    | null;
};


export type SubmitMintRequestResult =
  | {
      ok: true;
      queuedResponse:
        MintQueuedResponse;
    }
  | {
      ok: false;
      reason:
        | "validation"
        | "empty-response";
      message: string;
    };


/**
 * Mint申請条件を検証し、
 * Backendへ非同期Mint処理を申請する。
 *
 * loading state、alert、成功後の再取得は
 * Presentation層の責務とする。
 *
 * Repositoryから発生した通信エラーは握りつぶさず、
 * 呼び出し元へthrowする。
 */
export async function submitMintRequest(
  repository: SubmitMintRequestRepository,
  input: SubmitMintRequestInput,
): Promise<SubmitMintRequestResult> {
  const inspectionBatch =
    input.inspectionBatch ??
    null;


  const validation =
    validateMintRequestSubmit({
      inspectionBatch,


      isInspectionCompleted:
        inspectionBatch?.status ===
        "completed",


      selectedTokenBlueprintId:
        input.selectedTokenBlueprintId,


      productionId:
        input.productionId,
    });


  if (!validation.ok) {
    return {
      ok: false,
      reason:
        "validation",
      message:
        validation.message,
    };
  }


  const queuedResponse =
    await repository.postMintRequest(
      validation.productionId,
      validation.tokenBlueprintId,
    );


  if (
    !queuedResponse ||
    queuedResponse.status !==
      "QUEUED"
  ) {
    return {
      ok: false,
      reason:
        "empty-response",
      message:
        "ミント申請の受付結果を取得できませんでした。",
    };
  }


  return {
    ok: true,
    queuedResponse,
  };
}