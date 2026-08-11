// services/solana-bubblegum/src/application/mint-v2-usecase.ts

import type {
  KeypairSigner,
  Umi,
} from "@metaplex-foundation/umi";

import {
  CoreCollectionResolver,
} from "./core-collection-resolver.js";

import {
  createMintPayloadHash,
} from "./mint-payload-hash.js";

import {
  MerkleTreeResolver,
} from "./merkle-tree-resolver.js";

import {
  MintOperationSignedTransactionConflictError,
  type MintOperationRecord,
  type MintOperationRegistryPort,
  type MintOperationResult,
} from "./ports/mint-operation-registry-port.js";

import {
  isMintV2TransactionError,
  type MintV2Metadata,
  type MintV2TransactionPort,
} from "./ports/mint-v2-transaction-port.js";


const ASSET_STANDARD =
  "bubblegum-v2";

const SELLER_FEE_BASIS_POINTS_MIN =
  0;

const SELLER_FEE_BASIS_POINTS_MAX =
  10_000;

const CREATOR_SHARE_MIN =
  0;

const CREATOR_SHARE_MAX =
  100;

const CREATOR_SHARE_TOTAL =
  100;


export type MintV2UsecaseConfig = {
  cluster: string;
};


export type MintV2CoreCollectionInput = {
  name: string;

  metadataUri: string;
};


export type MintV2UsecaseInput = {
  productId: string;

  tokenBlueprintId: string;

  brandId: string;

  leafOwnerAddress: string;

  leafDelegateAddress:
    string | null;

  coreCollection:
    MintV2CoreCollectionInput;

  metadata:
    MintV2Metadata;

  umi:
    Umi;

  feePayer:
    KeypairSigner;

  reserve:
    KeypairSigner;
};


type ResolvedMintResources = {
  treeAddress: string;

  coreCollectionAddress: string;
};


type StoredSignedTransaction = {
  signature: string;

  signedTransactionBase64: string;
};


export class MintV2UsecaseValidationError
  extends Error {
  readonly name =
    "MintV2UsecaseValidationError";

  constructor(
    readonly field:
      string,

    message:
      string,
  ) {
    super(
      [
        "mint_v2_usecase: invalid input",
        `field=${field}`,
        message,
      ].join(
        " ",
      ),
    );
  }
}


export class MintV2UsecaseInvalidStateError
  extends Error {
  readonly name =
    "MintV2UsecaseInvalidStateError";

  constructor(
    readonly productId:
      string,

    readonly status:
      string,

    message:
      string,
  ) {
    super(
      [
        "mint_v2_usecase: invalid operation state",
        `productId=${productId}`,
        `status=${status}`,
        message,
      ].join(
        " ",
      ),
    );
  }
}


export class MintV2UsecaseStoredFatalError
  extends Error {
  readonly name =
    "MintV2UsecaseStoredFatalError";

  constructor(
    readonly productId:
      string,

    readonly errorCode:
      string | null,

    message:
      string,
  ) {
    super(
      [
        "mint_v2_usecase: operation failed fatally",
        `productId=${productId}`,
        `errorCode=${errorCode ?? "UNKNOWN"}`,
        message,
      ].join(
        " ",
      ),
    );
  }
}


export class MintV2UsecasePersistenceError
  extends Error {
  readonly name =
    "MintV2UsecasePersistenceError";

  constructor(
    message:
      string,

    cause:
      unknown,
  ) {
    super(
      message,
      {
        cause,
      },
    );
  }
}


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
    throw new MintV2UsecaseValidationError(
      field,
      "value is required",
    );
  }

  return value;
}


function requiredBoolean(
  field: string,
  value: unknown,
): boolean {
  if (
    typeof value !==
    "boolean"
  ) {
    throw new MintV2UsecaseValidationError(
      field,
      "value must be boolean",
    );
  }

  return value;
}


function requiredIntegerInRange(
  field: string,
  value: unknown,
  minimum: number,
  maximum: number,
): number {
  if (
    typeof value !==
      "number" ||
    !Number.isInteger(
      value,
    ) ||
    value <
      minimum ||
    value >
      maximum
  ) {
    throw new MintV2UsecaseValidationError(
      field,
      [
        "integer is out of range",
        `minimum=${minimum}`,
        `maximum=${maximum}`,
      ].join(
        " ",
      ),
    );
  }

  return value;
}


function errorMessage(
  error: unknown,
): string {
  if (
    error instanceof
      Error &&
    error.message.length >
      0
  ) {
    return error.message;
  }

  return String(
    error,
  );
}


export class MintV2Usecase {
  constructor(
    private readonly operationRegistry:
      MintOperationRegistryPort,

    private readonly transaction:
      MintV2TransactionPort,

    private readonly merkleTreeResolver:
      MerkleTreeResolver,

    private readonly coreCollectionResolver:
      CoreCollectionResolver,

    private readonly config:
      MintV2UsecaseConfig,

    private readonly now:
      () => Date =
      () => new Date(),
  ) {}


  async execute(
    input: MintV2UsecaseInput,
  ): Promise<MintOperationResult> {
    this.validateConfig();
    this.validateInput(
      input,
    );

    const payloadHash =
      this.createPayloadHash(
        input,
      );

    const reservation =
      await this.operationRegistry
        .reserve({
          productId:
            input.productId,

          payloadHash,

          now:
            this.now(),
        });

    return this.continueFromRecord(
      input,
      payloadHash,
      reservation.record,
    );
  }


  private async continueFromRecord(
    input: MintV2UsecaseInput,
    payloadHash: string,
    record: MintOperationRecord,
  ): Promise<MintOperationResult> {
    if (
      record.payloadHash !==
      payloadHash
    ) {
      throw new MintV2UsecaseInvalidStateError(
        input.productId,
        record.status,
        "payloadHash mismatch",
      );
    }

    if (
      record.status ===
      "CONFIRMED"
    ) {
      return this.requireConfirmedResult(
        record,
      );
    }

    if (
      record.status ===
      "FAILED_FATAL"
    ) {
      throw new MintV2UsecaseStoredFatalError(
        record.productId,
        record.errorCode,
        record.errorMessage ??
          "fatal mint operation failure",
      );
    }

    const signedTransaction =
      this.getStoredSignedTransaction(
        record,
      );

    if (
      signedTransaction !==
      null
    ) {
      return this.executeSubmittedTransaction(
        input,
        payloadHash,
        signedTransaction,
      );
    }

    if (
      record.status ===
      "SUBMITTED"
    ) {
      throw new MintV2UsecaseInvalidStateError(
        record.productId,
        record.status,
        "SUBMITTED operation has no signed transaction",
      );
    }

    return this.executeNewTransaction(
      input,
      payloadHash,
    );
  }


  private async executeNewTransaction(
    input: MintV2UsecaseInput,
    payloadHash: string,
  ): Promise<MintOperationResult> {
    let resources:
      ResolvedMintResources;

    let signedTransaction:
      StoredSignedTransaction;

    try {
      resources =
        await this.resolveResources(
          input,
        );

      signedTransaction =
        await this.transaction
          .buildAndSign({
            treeAddress:
              resources.treeAddress,

            leafOwnerAddress:
              input.leafOwnerAddress,

            leafDelegateAddress:
              input.leafDelegateAddress,

            coreCollectionAddress:
              resources.coreCollectionAddress,

            metadata:
              input.metadata,
          });
    } catch (error) {
      const concurrentResult =
        await this.recordExecutionFailure(
          input.productId,
          payloadHash,
          error,
        );

      if (
        concurrentResult !==
        null
      ) {
        return concurrentResult;
      }

      throw error;
    }

    let submittedRecord:
      MintOperationRecord;

    try {
      submittedRecord =
        await this.operationRegistry
          .markSubmitted({
            productId:
              input.productId,

            payloadHash,

            signature:
              signedTransaction.signature,

            signedTransactionBase64:
              signedTransaction
                .signedTransactionBase64,

            updatedAt:
              this.now(),
          });
    } catch (error) {
      if (
        error instanceof
        MintOperationSignedTransactionConflictError
      ) {
        const latest =
          await this.operationRegistry
            .getByProductId(
              input.productId,
            );

        if (
          latest !==
            null &&
          latest.payloadHash ===
            payloadHash
        ) {
          if (
            latest.status ===
            "CONFIRMED"
          ) {
            return this.requireConfirmedResult(
              latest,
            );
          }

          const winnerTransaction =
            this.getStoredSignedTransaction(
              latest,
            );

          if (
            winnerTransaction !==
            null
          ) {
            return this.executeSubmittedTransaction(
              input,
              payloadHash,
              winnerTransaction,
            );
          }
        }
      }

      throw error;
    }

    if (
      submittedRecord.status ===
      "CONFIRMED"
    ) {
      return this.requireConfirmedResult(
        submittedRecord,
      );
    }

    const persistedTransaction =
      this.getStoredSignedTransaction(
        submittedRecord,
      );

    if (
      persistedTransaction ===
      null
    ) {
      throw new MintV2UsecaseInvalidStateError(
        submittedRecord.productId,
        submittedRecord.status,
        "markSubmitted returned no signed transaction",
      );
    }

    return this.executeSubmittedTransaction(
      input,
      payloadHash,
      persistedTransaction,
      resources,
    );
  }


  private async executeSubmittedTransaction(
    input: MintV2UsecaseInput,
    payloadHash: string,
    signedTransaction: StoredSignedTransaction,
    existingResources?:
      ResolvedMintResources,
  ): Promise<MintOperationResult> {
    let resources:
      ResolvedMintResources;

    try {
      resources =
        existingResources ??
        await this.resolveResources(
          input,
        );

      const broadcastResult =
        await this.transaction
          .broadcast({
            signature:
              signedTransaction.signature,

            signedTransactionBase64:
              signedTransaction
                .signedTransactionBase64,
          });

      if (
        broadcastResult.signature !==
        signedTransaction.signature
      ) {
        throw new MintV2UsecaseInvalidStateError(
          input.productId,
          "SUBMITTED",
          [
            "broadcast signature mismatch",
            `expected=${signedTransaction.signature}`,
            `actual=${broadcastResult.signature}`,
          ].join(
            " ",
          ),
        );
      }

      const finalized =
        await this.transaction
          .waitForFinalized({
            signature:
              signedTransaction.signature,
          });

      const parsed =
        await this.transaction
          .parseMintResult({
            signature:
              signedTransaction.signature,
          });

      const result:
        MintOperationResult = {
          signature:
            signedTransaction.signature,

          assetStandard:
            ASSET_STANDARD,

          cluster:
            this.config.cluster,

          assetId:
            parsed.assetId,

          treeAddress:
            resources.treeAddress,

          leafIndex:
            parsed.leafIndex,

          coreCollectionAddress:
            resources
              .coreCollectionAddress,

          slot:
            finalized.slot,
        };

      const confirmed =
        await this.operationRegistry
          .markConfirmed({
            productId:
              input.productId,

            payloadHash,

            result,

            updatedAt:
              this.now(),
          });

      return this.requireConfirmedResult(
        confirmed,
      );
    } catch (error) {
      const concurrentResult =
        await this.recordExecutionFailure(
          input.productId,
          payloadHash,
          error,
        );

      if (
        concurrentResult !==
        null
      ) {
        return concurrentResult;
      }

      throw error;
    }
  }


  private async resolveResources(
    input: MintV2UsecaseInput,
  ): Promise<ResolvedMintResources> {
    const tree =
      await this.merkleTreeResolver
        .resolve({
          umi:
            input.umi,

          feePayer:
            input.feePayer,

          reserve:
            input.reserve,
        });

    if (
      tree.cluster !==
      this.config.cluster
    ) {
      throw new Error(
        [
          "mint_v2_usecase: merkle tree cluster mismatch",
          `expected=${this.config.cluster}`,
          `actual=${tree.cluster}`,
        ].join(
          " ",
        ),
      );
    }

    const collection =
      await this.coreCollectionResolver
        .resolve({
          tokenBlueprintId:
            input.tokenBlueprintId,

          name:
            input.coreCollection.name,

          metadataUri:
            input.coreCollection.metadataUri,

          umi:
            input.umi,

          feePayer:
            input.feePayer,

          reserve:
            input.reserve,
        });

    if (
      collection.cluster !==
      this.config.cluster
    ) {
      throw new Error(
        [
          "mint_v2_usecase: core collection cluster mismatch",
          `expected=${this.config.cluster}`,
          `actual=${collection.cluster}`,
        ].join(
          " ",
        ),
      );
    }

    return {
      treeAddress:
        tree.treeAddress,

      coreCollectionAddress:
        collection.collectionAddress,
    };
  }


  private async recordExecutionFailure(
    productId: string,
    payloadHash: string,
    error: unknown,
  ): Promise<MintOperationResult | null> {
    const transactionError =
      isMintV2TransactionError(
        error,
      )
        ? error
        : null;

    const failureStatus =
      transactionError?.kind ===
      "FATAL"
        ? "FAILED_FATAL"
        : "FAILED_RETRYABLE";

    const errorCode =
      transactionError?.code ??
      "MINT_V2_EXECUTION_FAILED";

    const message =
      errorMessage(
        error,
      );

    try {
      await this.operationRegistry
        .markFailed({
          productId,

          payloadHash,

          status:
            failureStatus,

          errorCode,

          errorMessage:
            message,

          updatedAt:
            this.now(),
        });

      return null;
    } catch (persistenceError) {
      let latest:
        MintOperationRecord | null =
        null;

      try {
        latest =
          await this.operationRegistry
            .getByProductId(
              productId,
            );
      } catch {
        latest =
          null;
      }

      if (
        latest !==
          null &&
        latest.payloadHash ===
          payloadHash &&
        latest.status ===
          "CONFIRMED"
      ) {
        return this.requireConfirmedResult(
          latest,
        );
      }

      throw new MintV2UsecasePersistenceError(
        [
          "mint_v2_usecase: failed to persist operation failure",
          `productId=${productId}`,
          `originalError=${message}`,
        ].join(
          " ",
        ),
        persistenceError,
      );
    }
  }


  private requireConfirmedResult(
    record: MintOperationRecord,
  ): MintOperationResult {
    if (
      record.status !==
      "CONFIRMED"
    ) {
      throw new MintV2UsecaseInvalidStateError(
        record.productId,
        record.status,
        "operation is not CONFIRMED",
      );
    }

    if (
      record.result ===
      null
    ) {
      throw new MintV2UsecaseInvalidStateError(
        record.productId,
        record.status,
        "CONFIRMED operation has no result",
      );
    }

    return record.result;
  }


  private getStoredSignedTransaction(
    record: MintOperationRecord,
  ): StoredSignedTransaction | null {
    if (
      record.signature ===
        null &&
      record.signedTransactionBase64 ===
        null
    ) {
      return null;
    }

    if (
      record.signature ===
        null ||
      record.signedTransactionBase64 ===
        null
    ) {
      throw new MintV2UsecaseInvalidStateError(
        record.productId,
        record.status,
        "signature and signedTransactionBase64 must both exist",
      );
    }

    return {
      signature:
        record.signature,

      signedTransactionBase64:
        record.signedTransactionBase64,
    };
  }


  private createPayloadHash(
    input: MintV2UsecaseInput,
  ): string {
    return createMintPayloadHash({
      productId:
        input.productId,

      tokenBlueprintId:
        input.tokenBlueprintId,

      brandId:
        input.brandId,

      leafOwnerAddress:
        input.leafOwnerAddress,

      leafDelegateAddress:
        input.leafDelegateAddress,

      coreCollection: {
        name:
          input.coreCollection.name,

        metadataUri:
          input.coreCollection.metadataUri,
      },

      metadata: {
        name:
          input.metadata.name,

        symbol:
          input.metadata.symbol,

        uri:
          input.metadata.uri,

        sellerFeeBasisPoints:
          input.metadata
            .sellerFeeBasisPoints,

        primarySaleHappened:
          input.metadata
            .primarySaleHappened,

        isMutable:
          input.metadata.isMutable,

        creators:
          input.metadata.creators.map(
            (creator) => ({
              address:
                creator.address,

              verified:
                creator.verified,

              share:
                creator.share,
            }),
          ),
      },
    });
  }


  private validateConfig(): void {
    if (
      !this.config.cluster
    ) {
      throw new Error(
        "mint_v2_usecase: cluster is required",
      );
    }
  }


  private validateInput(
    input: MintV2UsecaseInput,
  ): void {
    requiredString(
      "productId",
      input.productId,
    );

    requiredString(
      "tokenBlueprintId",
      input.tokenBlueprintId,
    );

    requiredString(
      "brandId",
      input.brandId,
    );

    requiredString(
      "leafOwnerAddress",
      input.leafOwnerAddress,
    );

    if (
      input.leafDelegateAddress !==
      null
    ) {
      requiredString(
        "leafDelegateAddress",
        input.leafDelegateAddress,
      );
    }

    if (
      input.coreCollection ===
        null ||
      typeof input.coreCollection !==
        "object"
    ) {
      throw new MintV2UsecaseValidationError(
        "coreCollection",
        "value is required",
      );
    }

    requiredString(
      "coreCollection.name",
      input.coreCollection.name,
    );

    requiredString(
      "coreCollection.metadataUri",
      input.coreCollection.metadataUri,
    );

    if (
      input.metadata ===
        null ||
      typeof input.metadata !==
        "object"
    ) {
      throw new MintV2UsecaseValidationError(
        "metadata",
        "value is required",
      );
    }

    requiredString(
      "metadata.name",
      input.metadata.name,
    );

    if (
      typeof input.metadata.symbol !==
      "string"
    ) {
      throw new MintV2UsecaseValidationError(
        "metadata.symbol",
        "value must be string",
      );
    }

    requiredString(
      "metadata.uri",
      input.metadata.uri,
    );

    requiredIntegerInRange(
      "metadata.sellerFeeBasisPoints",
      input.metadata.sellerFeeBasisPoints,
      SELLER_FEE_BASIS_POINTS_MIN,
      SELLER_FEE_BASIS_POINTS_MAX,
    );

    requiredBoolean(
      "metadata.primarySaleHappened",
      input.metadata.primarySaleHappened,
    );

    requiredBoolean(
      "metadata.isMutable",
      input.metadata.isMutable,
    );

    if (
      !Array.isArray(
        input.metadata.creators,
      )
    ) {
      throw new MintV2UsecaseValidationError(
        "metadata.creators",
        "value must be array",
      );
    }

    const creatorAddresses =
      new Set<string>();

    let totalShare =
      0;

    input.metadata.creators.forEach(
      (
        creator,
        index,
      ) => {
        if (
          creator ===
            null ||
          typeof creator !==
            "object"
        ) {
          throw new MintV2UsecaseValidationError(
            `metadata.creators[${index}]`,
            "creator is invalid",
          );
        }

        const address =
          requiredString(
            `metadata.creators[${index}].address`,
            creator.address,
          );

        if (
          creatorAddresses.has(
            address,
          )
        ) {
          throw new MintV2UsecaseValidationError(
            `metadata.creators[${index}].address`,
            `duplicate creator address=${address}`,
          );
        }

        creatorAddresses.add(
          address,
        );

        requiredBoolean(
          `metadata.creators[${index}].verified`,
          creator.verified,
        );

        const share =
          requiredIntegerInRange(
            `metadata.creators[${index}].share`,
            creator.share,
            CREATOR_SHARE_MIN,
            CREATOR_SHARE_MAX,
          );

        totalShare +=
          share;
      },
    );

    if (
      input.metadata.creators.length >
        0 &&
      totalShare !==
        CREATOR_SHARE_TOTAL
    ) {
      throw new MintV2UsecaseValidationError(
        "metadata.creators",
        [
          "creator share total must equal 100",
          `actual=${totalShare}`,
        ].join(
          " ",
        ),
      );
    }
  }
}