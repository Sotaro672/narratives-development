// services/solana-bubblegum/src/infrastructure/firestore/mint-operation-registry-repository.ts

import {
  Timestamp,
} from "@google-cloud/firestore";

import {
  MintOperationNotFoundError,
  MintOperationPayloadConflictError,
  MintOperationSignedTransactionConflictError,
  MintOperationStateConflictError,
  type MarkMintOperationConfirmedInput,
  type MarkMintOperationFailedInput,
  type MarkMintOperationSubmittedInput,
  type MintOperationRegistryPort,
  type MintOperationRecord,
  type MintOperationResult,
  type MintOperationStatus,
  type ReserveMintOperationInput,
  type ReserveMintOperationResult,
} from "../../application/ports/mint-operation-registry-port.js";

import {
  firestore,
} from "./firestore-client.js";


type FirestoreMintOperationResult = {
  signature?: unknown;

  assetStandard?: unknown;

  cluster?: unknown;

  assetId?: unknown;

  treeAddress?: unknown;

  leafIndex?: unknown;

  coreCollectionAddress?: unknown;

  slot?: unknown;
};


type FirestoreMintOperationRecord = {
  productId?: unknown;

  payloadHash?: unknown;

  status?: unknown;

  signature?: unknown;

  signedTransactionBase64?: unknown;

  result?: unknown;

  errorCode?: unknown;

  errorMessage?: unknown;

  createdAt?: unknown;

  updatedAt?: unknown;

  submittedAt?: unknown;

  confirmedAt?: unknown;

  failedAt?: unknown;
};


const COLLECTION_NAME =
  "bubblegumMintOperations";


function requiredString(
  field: string,
  value: unknown,
): string {
  if (
    typeof value !==
      "string" ||
    value.length ===
      0
  ) {
    throw new Error(
      `mint_operation_registry: invalid ${field}`,
    );
  }

  return value;
}


function nullableString(
  field: string,
  value: unknown,
): string | null {
  if (
    value ===
      null ||
    value ===
      undefined
  ) {
    return null;
  }

  return requiredString(
    field,
    value,
  );
}


function requiredInteger(
  field: string,
  value: unknown,
): number {
  if (
    typeof value !==
      "number" ||
    !Number.isSafeInteger(
      value,
    ) ||
    value <
      0
  ) {
    throw new Error(
      `mint_operation_registry: invalid ${field}`,
    );
  }

  return value;
}


function requiredDate(
  field: string,
  value: unknown,
): Date {
  if (
    value instanceof
    Timestamp
  ) {
    return value.toDate();
  }

  if (
    value instanceof
    Date
  ) {
    return value;
  }

  throw new Error(
    `mint_operation_registry: invalid ${field}`,
  );
}


function nullableDate(
  field: string,
  value: unknown,
): Date | null {
  if (
    value ===
      null ||
    value ===
      undefined
  ) {
    return null;
  }

  return requiredDate(
    field,
    value,
  );
}


function requiredStatus(
  value: unknown,
): MintOperationStatus {
  switch (value) {
    case "RESERVED":
    case "SUBMITTED":
    case "CONFIRMED":
    case "FAILED_RETRYABLE":
    case "FAILED_FATAL":
      return value;

    default:
      throw new Error(
        "mint_operation_registry: invalid status",
      );
  }
}


function requiredResult(
  value: unknown,
): MintOperationResult {
  if (
    typeof value !==
      "object" ||
    value ===
      null
  ) {
    throw new Error(
      "mint_operation_registry: invalid result",
    );
  }

  const data =
    value as FirestoreMintOperationResult;

  return {
    signature:
      requiredString(
        "result.signature",
        data.signature,
      ),

    assetStandard:
      requiredString(
        "result.assetStandard",
        data.assetStandard,
      ),

    cluster:
      requiredString(
        "result.cluster",
        data.cluster,
      ),

    assetId:
      requiredString(
        "result.assetId",
        data.assetId,
      ),

    treeAddress:
      requiredString(
        "result.treeAddress",
        data.treeAddress,
      ),

    leafIndex:
      requiredInteger(
        "result.leafIndex",
        data.leafIndex,
      ),

    coreCollectionAddress:
      requiredString(
        "result.coreCollectionAddress",
        data.coreCollectionAddress,
      ),

    slot:
      requiredInteger(
        "result.slot",
        data.slot,
      ),
  };
}


function nullableResult(
  value: unknown,
): MintOperationResult | null {
  if (
    value ===
      null ||
    value ===
      undefined
  ) {
    return null;
  }

  return requiredResult(
    value,
  );
}


function assertRecordConsistency(
  record: MintOperationRecord,
): void {
  if (
    record.status ===
      "SUBMITTED" ||
    record.status ===
      "CONFIRMED"
  ) {
    if (
      record.signature ===
        null ||
      record.signedTransactionBase64 ===
        null
    ) {
      throw new Error(
        [
          "mint_operation_registry: submitted operation missing signed transaction",
          `productId=${record.productId}`,
          `status=${record.status}`,
        ].join(
          " ",
        ),
      );
    }
  }

  if (
    record.status ===
    "CONFIRMED"
  ) {
    if (
      record.result ===
      null
    ) {
      throw new Error(
        [
          "mint_operation_registry: confirmed operation missing result",
          `productId=${record.productId}`,
        ].join(
          " ",
        ),
      );
    }

    if (
      record.result.signature !==
      record.signature
    ) {
      throw new Error(
        [
          "mint_operation_registry: confirmed signature mismatch",
          `productId=${record.productId}`,
        ].join(
          " ",
        ),
      );
    }
  }
}


function fromFirestore(
  data: FirestoreMintOperationRecord,
): MintOperationRecord {
  const record:
    MintOperationRecord = {
      productId:
        requiredString(
          "productId",
          data.productId,
        ),

      payloadHash:
        requiredString(
          "payloadHash",
          data.payloadHash,
        ),

      status:
        requiredStatus(
          data.status,
        ),

      signature:
        nullableString(
          "signature",
          data.signature,
        ),

      signedTransactionBase64:
        nullableString(
          "signedTransactionBase64",
          data.signedTransactionBase64,
        ),

      result:
        nullableResult(
          data.result,
        ),

      errorCode:
        nullableString(
          "errorCode",
          data.errorCode,
        ),

      errorMessage:
        nullableString(
          "errorMessage",
          data.errorMessage,
        ),

      createdAt:
        requiredDate(
          "createdAt",
          data.createdAt,
        ),

      updatedAt:
        requiredDate(
          "updatedAt",
          data.updatedAt,
        ),

      submittedAt:
        nullableDate(
          "submittedAt",
          data.submittedAt,
        ),

      confirmedAt:
        nullableDate(
          "confirmedAt",
          data.confirmedAt,
        ),

      failedAt:
        nullableDate(
          "failedAt",
          data.failedAt,
        ),
    };

  assertRecordConsistency(
    record,
  );

  return record;
}


function resultToFirestore(
  result: MintOperationResult,
): Record<string, unknown> {
  return {
    signature:
      result.signature,

    assetStandard:
      result.assetStandard,

    cluster:
      result.cluster,

    assetId:
      result.assetId,

    treeAddress:
      result.treeAddress,

    leafIndex:
      result.leafIndex,

    coreCollectionAddress:
      result.coreCollectionAddress,

    slot:
      result.slot,
  };
}


function toFirestore(
  record: MintOperationRecord,
): Record<string, unknown> {
  assertRecordConsistency(
    record,
  );

  return {
    productId:
      record.productId,

    payloadHash:
      record.payloadHash,

    status:
      record.status,

    signature:
      record.signature,

    signedTransactionBase64:
      record.signedTransactionBase64,

    result:
      record.result ===
        null
        ? null
        : resultToFirestore(
            record.result,
          ),

    errorCode:
      record.errorCode,

    errorMessage:
      record.errorMessage,

    createdAt:
      Timestamp.fromDate(
        record.createdAt,
      ),

    updatedAt:
      Timestamp.fromDate(
        record.updatedAt,
      ),

    submittedAt:
      record.submittedAt ===
        null
        ? null
        : Timestamp.fromDate(
            record.submittedAt,
          ),

    confirmedAt:
      record.confirmedAt ===
        null
        ? null
        : Timestamp.fromDate(
            record.confirmedAt,
          ),

    failedAt:
      record.failedAt ===
        null
        ? null
        : Timestamp.fromDate(
            record.failedAt,
          ),
  };
}


function assertPayloadHash(
  existing: MintOperationRecord,
  requestedPayloadHash: string,
): void {
  if (
    existing.payloadHash ===
    requestedPayloadHash
  ) {
    return;
  }

  throw new MintOperationPayloadConflictError(
    existing.productId,
    existing.payloadHash,
    requestedPayloadHash,
  );
}


function assertSignedTransactionMatches(
  existing: MintOperationRecord,
  signature: string,
  signedTransactionBase64: string,
): void {
  if (
    existing.signature !==
      null &&
    existing.signature !==
      signature
  ) {
    throw new MintOperationSignedTransactionConflictError(
      existing.productId,
    );
  }

  if (
    existing.signedTransactionBase64 !==
      null &&
    existing.signedTransactionBase64 !==
      signedTransactionBase64
  ) {
    throw new MintOperationSignedTransactionConflictError(
      existing.productId,
    );
  }
}


function resultsEqual(
  left: MintOperationResult,
  right: MintOperationResult,
): boolean {
  return (
    left.signature ===
      right.signature &&
    left.assetStandard ===
      right.assetStandard &&
    left.cluster ===
      right.cluster &&
    left.assetId ===
      right.assetId &&
    left.treeAddress ===
      right.treeAddress &&
    left.leafIndex ===
      right.leafIndex &&
    left.coreCollectionAddress ===
      right.coreCollectionAddress &&
    left.slot ===
      right.slot
  );
}


export class FirestoreMintOperationRegistryRepository
  implements MintOperationRegistryPort {
  async getByProductId(
    productId: string,
  ): Promise<MintOperationRecord | null> {
    requiredString(
      "productId",
      productId,
    );

    const snapshot =
      await firestore
        .collection(
          COLLECTION_NAME,
        )
        .doc(
          productId,
        )
        .get();

    if (!snapshot.exists) {
      return null;
    }

    const record =
      fromFirestore(
        snapshot.data() as FirestoreMintOperationRecord,
      );

    if (
      record.productId !==
      productId
    ) {
      throw new Error(
        [
          "mint_operation_registry: productId mismatch",
          `documentId=${productId}`,
          `recordProductId=${record.productId}`,
        ].join(
          " ",
        ),
      );
    }

    return record;
  }


  async reserve(
    input: ReserveMintOperationInput,
  ): Promise<ReserveMintOperationResult> {
    requiredString(
      "productId",
      input.productId,
    );

    requiredString(
      "payloadHash",
      input.payloadHash,
    );

    requiredDate(
      "now",
      input.now,
    );

    const ref =
      firestore
        .collection(
          COLLECTION_NAME,
        )
        .doc(
          input.productId,
        );

    return firestore.runTransaction(
      async (
        transaction,
      ): Promise<ReserveMintOperationResult> => {
        const snapshot =
          await transaction.get(
            ref,
          );

        if (snapshot.exists) {
          const existing =
            fromFirestore(
              snapshot.data() as FirestoreMintOperationRecord,
            );

          assertPayloadHash(
            existing,
            input.payloadHash,
          );

          return {
            kind:
              "existing",

            record:
              existing,
          };
        }

        const record:
          MintOperationRecord = {
            productId:
              input.productId,

            payloadHash:
              input.payloadHash,

            status:
              "RESERVED",

            signature:
              null,

            signedTransactionBase64:
              null,

            result:
              null,

            errorCode:
              null,

            errorMessage:
              null,

            createdAt:
              input.now,

            updatedAt:
              input.now,

            submittedAt:
              null,

            confirmedAt:
              null,

            failedAt:
              null,
          };

        transaction.create(
          ref,
          toFirestore(
            record,
          ),
        );

        return {
          kind:
            "reserved",

          record,
        };
      },
    );
  }


  async markSubmitted(
    input: MarkMintOperationSubmittedInput,
  ): Promise<MintOperationRecord> {
    requiredString(
      "productId",
      input.productId,
    );

    requiredString(
      "payloadHash",
      input.payloadHash,
    );

    requiredString(
      "signature",
      input.signature,
    );

    requiredString(
      "signedTransactionBase64",
      input.signedTransactionBase64,
    );

    requiredDate(
      "updatedAt",
      input.updatedAt,
    );

    const ref =
      firestore
        .collection(
          COLLECTION_NAME,
        )
        .doc(
          input.productId,
        );

    return firestore.runTransaction(
      async (
        transaction,
      ): Promise<MintOperationRecord> => {
        const snapshot =
          await transaction.get(
            ref,
          );

        if (!snapshot.exists) {
          throw new MintOperationNotFoundError(
            input.productId,
          );
        }

        const existing =
          fromFirestore(
            snapshot.data() as FirestoreMintOperationRecord,
          );

        assertPayloadHash(
          existing,
          input.payloadHash,
        );

        assertSignedTransactionMatches(
          existing,
          input.signature,
          input.signedTransactionBase64,
        );

        if (
          existing.status ===
          "CONFIRMED"
        ) {
          return existing;
        }

        if (
          existing.status ===
          "FAILED_FATAL"
        ) {
          throw new MintOperationStateConflictError(
            input.productId,
            existing.status,
            "SUBMITTED",
          );
        }

        if (
          existing.status ===
            "SUBMITTED" &&
          existing.signature ===
            input.signature &&
          existing.signedTransactionBase64 ===
            input.signedTransactionBase64
        ) {
          return existing;
        }

        const next:
          MintOperationRecord = {
            ...existing,

            status:
              "SUBMITTED",

            signature:
              input.signature,

            signedTransactionBase64:
              input.signedTransactionBase64,

            errorCode:
              null,

            errorMessage:
              null,

            updatedAt:
              input.updatedAt,

            submittedAt:
              existing.submittedAt ??
              input.updatedAt,
          };

        transaction.set(
          ref,
          toFirestore(
            next,
          ),
        );

        return next;
      },
    );
  }


  async markConfirmed(
    input: MarkMintOperationConfirmedInput,
  ): Promise<MintOperationRecord> {
    requiredString(
      "productId",
      input.productId,
    );

    requiredString(
      "payloadHash",
      input.payloadHash,
    );

    requiredDate(
      "updatedAt",
      input.updatedAt,
    );

    const ref =
      firestore
        .collection(
          COLLECTION_NAME,
        )
        .doc(
          input.productId,
        );

    return firestore.runTransaction(
      async (
        transaction,
      ): Promise<MintOperationRecord> => {
        const snapshot =
          await transaction.get(
            ref,
          );

        if (!snapshot.exists) {
          throw new MintOperationNotFoundError(
            input.productId,
          );
        }

        const existing =
          fromFirestore(
            snapshot.data() as FirestoreMintOperationRecord,
          );

        assertPayloadHash(
          existing,
          input.payloadHash,
        );

        if (
          existing.status ===
            "CONFIRMED"
        ) {
          if (
            existing.result !==
              null &&
            resultsEqual(
              existing.result,
              input.result,
            )
          ) {
            return existing;
          }

          throw new MintOperationStateConflictError(
            input.productId,
            existing.status,
            "CONFIRMED",
          );
        }

        if (
          existing.status ===
          "FAILED_FATAL"
        ) {
          throw new MintOperationStateConflictError(
            input.productId,
            existing.status,
            "CONFIRMED",
          );
        }

        if (
          existing.status ===
          "RESERVED"
        ) {
          throw new MintOperationStateConflictError(
            input.productId,
            existing.status,
            "CONFIRMED",
          );
        }

        if (
          existing.signature ===
            null ||
          existing.signedTransactionBase64 ===
            null
        ) {
          throw new MintOperationStateConflictError(
            input.productId,
            existing.status,
            "CONFIRMED",
          );
        }

        if (
          existing.signature !==
          input.result.signature
        ) {
          throw new MintOperationSignedTransactionConflictError(
            input.productId,
          );
        }

        const next:
          MintOperationRecord = {
            ...existing,

            status:
              "CONFIRMED",

            result:
              input.result,

            errorCode:
              null,

            errorMessage:
              null,

            updatedAt:
              input.updatedAt,

            confirmedAt:
              existing.confirmedAt ??
              input.updatedAt,
          };

        transaction.set(
          ref,
          toFirestore(
            next,
          ),
        );

        return next;
      },
    );
  }


  async markFailed(
    input: MarkMintOperationFailedInput,
  ): Promise<MintOperationRecord> {
    requiredString(
      "productId",
      input.productId,
    );

    requiredString(
      "payloadHash",
      input.payloadHash,
    );

    requiredString(
      "errorMessage",
      input.errorMessage,
    );

    requiredDate(
      "updatedAt",
      input.updatedAt,
    );

    if (
      input.status !==
        "FAILED_RETRYABLE" &&
      input.status !==
        "FAILED_FATAL"
    ) {
      throw new Error(
        "mint_operation_registry: invalid failure status",
      );
    }

    if (
      input.errorCode !==
        null
    ) {
      requiredString(
        "errorCode",
        input.errorCode,
      );
    }

    const ref =
      firestore
        .collection(
          COLLECTION_NAME,
        )
        .doc(
          input.productId,
        );

    return firestore.runTransaction(
      async (
        transaction,
      ): Promise<MintOperationRecord> => {
        const snapshot =
          await transaction.get(
            ref,
          );

        if (!snapshot.exists) {
          throw new MintOperationNotFoundError(
            input.productId,
          );
        }

        const existing =
          fromFirestore(
            snapshot.data() as FirestoreMintOperationRecord,
          );

        assertPayloadHash(
          existing,
          input.payloadHash,
        );

        if (
          existing.status ===
          "CONFIRMED"
        ) {
          throw new MintOperationStateConflictError(
            input.productId,
            existing.status,
            input.status,
          );
        }

        if (
          existing.status ===
            "FAILED_FATAL"
        ) {
          if (
            input.status ===
              "FAILED_FATAL" &&
            existing.errorCode ===
              input.errorCode &&
            existing.errorMessage ===
              input.errorMessage
          ) {
            return existing;
          }

          throw new MintOperationStateConflictError(
            input.productId,
            existing.status,
            input.status,
          );
        }

        if (
          existing.status ===
            input.status &&
          existing.errorCode ===
            input.errorCode &&
          existing.errorMessage ===
            input.errorMessage
        ) {
          return existing;
        }

        const next:
          MintOperationRecord = {
            ...existing,

            status:
              input.status,

            errorCode:
              input.errorCode,

            errorMessage:
              input.errorMessage,

            updatedAt:
              input.updatedAt,

            failedAt:
              input.updatedAt,
          };

        transaction.set(
          ref,
          toFirestore(
            next,
          ),
        );

        return next;
      },
    );
  }
}