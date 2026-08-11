// services/solana-bubblegum/src/application/ports/mint-operation-registry-port.ts

export type MintOperationStatus =
  | "RESERVED"
  | "SUBMITTED"
  | "CONFIRMED"
  | "FAILED_RETRYABLE"
  | "FAILED_FATAL";


export type MintOperationFailureStatus =
  | "FAILED_RETRYABLE"
  | "FAILED_FATAL";


export type MintOperationResult = {
  signature: string;

  assetStandard: string;

  cluster: string;

  assetId: string;

  treeAddress: string;

  leafIndex: number;

  coreCollectionAddress: string;

  slot: number;
};


export type MintOperationRecord = {
  productId: string;

  payloadHash: string;

  status:
    MintOperationStatus;

  signature:
    string | null;

  signedTransactionBase64:
    string | null;

  result:
    MintOperationResult | null;

  errorCode:
    string | null;

  errorMessage:
    string | null;

  createdAt:
    Date;

  updatedAt:
    Date;

  submittedAt:
    Date | null;

  confirmedAt:
    Date | null;

  failedAt:
    Date | null;
};


export type ReserveMintOperationInput = {
  productId: string;

  payloadHash: string;

  now: Date;
};


export type ReserveMintOperationResult = {
  kind:
    | "reserved"
    | "existing";

  record:
    MintOperationRecord;
};


export type MarkMintOperationSubmittedInput = {
  productId: string;

  payloadHash: string;

  signature: string;

  signedTransactionBase64: string;

  updatedAt: Date;
};


export type MarkMintOperationConfirmedInput = {
  productId: string;

  payloadHash: string;

  result:
    MintOperationResult;

  updatedAt: Date;
};


export type MarkMintOperationFailedInput = {
  productId: string;

  payloadHash: string;

  status:
    MintOperationFailureStatus;

  errorCode:
    string | null;

  errorMessage: string;

  updatedAt: Date;
};


export interface MintOperationRegistryPort {
  getByProductId(
    productId: string,
  ): Promise<MintOperationRecord | null>;

  reserve(
    input: ReserveMintOperationInput,
  ): Promise<ReserveMintOperationResult>;

  markSubmitted(
    input: MarkMintOperationSubmittedInput,
  ): Promise<MintOperationRecord>;

  markConfirmed(
    input: MarkMintOperationConfirmedInput,
  ): Promise<MintOperationRecord>;

  markFailed(
    input: MarkMintOperationFailedInput,
  ): Promise<MintOperationRecord>;
}


export class MintOperationPayloadConflictError
  extends Error {
  readonly name =
    "MintOperationPayloadConflictError";

  constructor(
    readonly productId: string,
    readonly existingPayloadHash: string,
    readonly requestedPayloadHash: string,
  ) {
    super(
      [
        "mint_operation_registry: payload conflict",
        `productId=${productId}`,
        `existingPayloadHash=${existingPayloadHash}`,
        `requestedPayloadHash=${requestedPayloadHash}`,
      ].join(
        " ",
      ),
    );
  }
}


export class MintOperationNotFoundError
  extends Error {
  readonly name =
    "MintOperationNotFoundError";

  constructor(
    readonly productId: string,
  ) {
    super(
      [
        "mint_operation_registry: operation not found",
        `productId=${productId}`,
      ].join(
        " ",
      ),
    );
  }
}


export class MintOperationStateConflictError
  extends Error {
  readonly name =
    "MintOperationStateConflictError";

  constructor(
    readonly productId: string,
    readonly currentStatus: MintOperationStatus,
    readonly requestedStatus: MintOperationStatus,
  ) {
    super(
      [
        "mint_operation_registry: state conflict",
        `productId=${productId}`,
        `currentStatus=${currentStatus}`,
        `requestedStatus=${requestedStatus}`,
      ].join(
        " ",
      ),
    );
  }
}


export class MintOperationSignedTransactionConflictError
  extends Error {
  readonly name =
    "MintOperationSignedTransactionConflictError";

  constructor(
    readonly productId: string,
  ) {
    super(
      [
        "mint_operation_registry: signed transaction conflict",
        `productId=${productId}`,
      ].join(
        " ",
      ),
    );
  }
}