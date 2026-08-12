// services/solana-bubblegum/src/scripts/verify-mint-v2-idempotency.ts

import { Buffer } from "node:buffer";
import { randomUUID } from "node:crypto";
import type { KeypairSigner, Umi } from "@metaplex-foundation/umi";
import { CoreCollectionResolver } from "../application/core-collection-resolver.js";
import type { FeePayerTopUpUsecase } from "../application/fee-payer-top-up.js";
import { createMintPayloadHash } from "../application/mint-payload-hash.js";
import { MerkleTreeResolver } from "../application/merkle-tree-resolver.js";
import {
  MintV2Usecase,
  MintV2UsecaseStoredFatalError,
  type MintV2UsecaseInput,
} from "../application/mint-v2-usecase.js";
import { MintOperationPayloadConflictError } from "../application/ports/mint-operation-registry-port.js";
import {
  MintV2TransactionError,
  type BroadcastMintV2TransactionInput,
  type BroadcastMintV2TransactionResult,
  type BuildAndSignMintV2TransactionInput,
  type BuildAndSignMintV2TransactionResult,
  type MintV2TransactionPort,
  type ParseMintV2ResultInput,
  type ParseMintV2ResultResult,
  type WaitForMintV2FinalizedInput,
  type WaitForMintV2FinalizedResult,
} from "../application/ports/mint-v2-transaction-port.js";
import { env } from "../config/env.js";
import { firestore } from "../infrastructure/firestore/firestore-client.js";
import { FirestoreMintOperationRegistryRepository } from "../infrastructure/firestore/mint-operation-registry-repository.js";

const TEST_TREE_ADDRESS = "verify-merkle-tree-address";
const TEST_CORE_COLLECTION_ADDRESS = "verify-core-collection-address";
const TEST_LEAF_OWNER_ADDRESS = "verify-leaf-owner-address";
const TEST_TOKEN_BLUEPRINT_ID = "verify-token-blueprint";
const TEST_BRAND_ID = "verify-brand";

class FakeMintV2TransactionClient implements MintV2TransactionPort {
  buildAndSignCalls = 0;
  broadcastCalls = 0;
  waitForFinalizedCalls = 0;
  parseMintResultCalls = 0;
  failNextBuildRetryable = false;
  failNextBuildFatal = false;
  failNextBroadcast = false;

  async buildAndSign(
    _input: BuildAndSignMintV2TransactionInput,
  ): Promise<BuildAndSignMintV2TransactionResult> {
    this.buildAndSignCalls += 1;

    if (this.failNextBuildRetryable) {
      this.failNextBuildRetryable = false;

      throw new MintV2TransactionError(
        "RETRYABLE",
        "TEST_BUILD_RETRYABLE",
        "verify_mint_v2_idempotency: simulated retryable build failure",
      );
    }

    if (this.failNextBuildFatal) {
      this.failNextBuildFatal = false;

      throw new MintV2TransactionError(
        "FATAL",
        "TEST_BUILD_FATAL",
        "verify_mint_v2_idempotency: simulated fatal build failure",
      );
    }

    const signature = [
      "verify-signature",
      this.buildAndSignCalls,
      randomUUID(),
    ].join("-");

    const signedTransactionBase64 = Buffer
      .from(
        [
          "verify-signed-transaction",
          signature,
        ].join(":"),
        "utf8",
      )
      .toString("base64");

    return {
      signature,
      signedTransactionBase64,
    };
  }

  async broadcast(
    input: BroadcastMintV2TransactionInput,
  ): Promise<BroadcastMintV2TransactionResult> {
    this.broadcastCalls += 1;

    if (this.failNextBroadcast) {
      this.failNextBroadcast = false;

      throw new MintV2TransactionError(
        "RETRYABLE",
        "TEST_BROADCAST_RETRYABLE",
        [
          "verify_mint_v2_idempotency: simulated broadcast failure",
          `signature=${input.signature}`,
        ].join(" "),
      );
    }

    return {
      signature: input.signature,
    };
  }

  async waitForFinalized(
    _input: WaitForMintV2FinalizedInput,
  ): Promise<WaitForMintV2FinalizedResult> {
    this.waitForFinalizedCalls += 1;

    return {
      slot: 100_000 + this.waitForFinalizedCalls,
    };
  }

  async parseMintResult(
    input: ParseMintV2ResultInput,
  ): Promise<ParseMintV2ResultResult> {
    this.parseMintResultCalls += 1;

    return {
      assetId: [
        "verify-asset",
        input.signature,
      ].join("-"),
      leafIndex: 100 + this.parseMintResultCalls,
    };
  }
}

function assertCondition(
  condition: unknown,
  message: string,
): asserts condition {
  if (!condition) {
    throw new Error([
      "verify_mint_v2_idempotency:",
      message,
    ].join(" "));
  }
}

function assertNumberEquals(
  actual: number,
  expected: number,
  message: string,
): void {
  if (actual !== expected) {
    throw new Error([
      "verify_mint_v2_idempotency:",
      message,
      `expected=${expected}`,
      `actual=${actual}`,
    ].join(" "));
  }
}

function createProductId(
  scenario: string,
): string {
  return [
    "verify-mint-v2",
    scenario,
    randomUUID(),
  ].join("-");
}

function createInput(
  productId: string,
): MintV2UsecaseInput {
  return {
    productId,
    tokenBlueprintId: TEST_TOKEN_BLUEPRINT_ID,
    brandId: TEST_BRAND_ID,
    leafOwnerAddress: TEST_LEAF_OWNER_ADDRESS,
    leafDelegateAddress: null,
    coreCollection: {
      name: "Verify Collection",
      metadataUri: "https://example.com/verify-collection.json",
    },
    metadata: {
      name: "Verify Asset",
      symbol: "VERIFY",
      uri: "https://example.com/verify-asset.json",
      sellerFeeBasisPoints: 0,
      primarySaleHappened: false,
      isMutable: false,
      creators: [],
    },
    umi: {} as Umi,
    feePayer: {} as KeypairSigner,
    reserve: {} as KeypairSigner,
  };
}

function createPayloadHash(
  input: MintV2UsecaseInput,
): string {
  return createMintPayloadHash({
    productId: input.productId,
    tokenBlueprintId: input.tokenBlueprintId,
    brandId: input.brandId,
    leafOwnerAddress: input.leafOwnerAddress,
    leafDelegateAddress: input.leafDelegateAddress,
    coreCollection: {
      name: input.coreCollection.name,
      metadataUri: input.coreCollection.metadataUri,
    },
    metadata: {
      name: input.metadata.name,
      symbol: input.metadata.symbol,
      uri: input.metadata.uri,
      sellerFeeBasisPoints:
        input.metadata.sellerFeeBasisPoints,
      primarySaleHappened:
        input.metadata.primarySaleHappened,
      isMutable: input.metadata.isMutable,
      creators: input.metadata.creators.map(
        (creator) => ({
          address: creator.address,
          verified: creator.verified,
          share: creator.share,
        }),
      ),
    },
  });
}

function createFakeMerkleTreeResolver(): MerkleTreeResolver {
  return {
    resolve: async () => ({
      status: "existing" as const,
      treeAddress: TEST_TREE_ADDRESS,
      cluster: env.solanaCluster,
      maxDepth: 14,
      maxBufferSize: 64,
      canopyDepth: 8,
      public: false,
      txSignature: "verify-tree-signature",
    }),
  } as unknown as MerkleTreeResolver;
}

function createFakeCoreCollectionResolver(): CoreCollectionResolver {
  return {
    resolve: async (
      input: {
        tokenBlueprintId: string;
        name: string;
        metadataUri: string;
      },
    ) => ({
      status: "existing" as const,
      tokenBlueprintId: input.tokenBlueprintId,
      collectionAddress: TEST_CORE_COLLECTION_ADDRESS,
      name: input.name,
      metadataUri: input.metadataUri,
      cluster: env.solanaCluster,
      txSignature: "verify-collection-signature",
    }),
  } as unknown as CoreCollectionResolver;
}

function createFakeFeePayerTopUpUsecase(): FeePayerTopUpUsecase {
  return {
    execute: async () => ({
      status: "balance_sufficient" as const,
      feePayerAddress: "verify-fee-payer-address",
      reserveAddress: "verify-reserve-address",
      feePayerBalanceBeforeSOL: 1,
      feePayerBalanceAfterSOL: 1,
      reserveBalanceBeforeSOL: 1,
      reserveBalanceAfterSOL: 1,
      transferredSOL: 0,
    }),
  } as unknown as FeePayerTopUpUsecase;
}

function createUsecase(
  registry: FirestoreMintOperationRegistryRepository,
  transaction: FakeMintV2TransactionClient,
): MintV2Usecase {
  return new MintV2Usecase(
    registry,
    transaction,
    createFakeMerkleTreeResolver(),
    createFakeCoreCollectionResolver(),
    createFakeFeePayerTopUpUsecase(),
    {
      cluster: env.solanaCluster,
    },
  );
}

async function deleteTestRecords(
  productIds: Set<string>,
): Promise<void> {
  await Promise.all(
    Array
      .from(productIds)
      .map(async (productId) => {
        await firestore
          .collection("bubblegumMintOperations")
          .doc(productId)
          .delete();
      }),
  );
}

async function verifyConfirmedIdempotency(
  registry: FirestoreMintOperationRegistryRepository,
  productIds: Set<string>,
): Promise<void> {
  const productId = createProductId("confirmed");
  productIds.add(productId);

  const transaction =
    new FakeMintV2TransactionClient();

  const usecase =
    createUsecase(registry, transaction);

  const input =
    createInput(productId);

  const first =
    await usecase.execute(input);

  assertNumberEquals(
    transaction.buildAndSignCalls,
    1,
    "first execution must build exactly one transaction",
  );

  assertNumberEquals(
    transaction.broadcastCalls,
    1,
    "first execution must broadcast exactly once",
  );

  assertNumberEquals(
    transaction.waitForFinalizedCalls,
    1,
    "first execution must wait for finalization once",
  );

  assertNumberEquals(
    transaction.parseMintResultCalls,
    1,
    "first execution must parse mint result once",
  );

  const afterFirst =
    await registry.getByProductId(productId);

  assertCondition(
    afterFirst !== null,
    "CONFIRMED record must exist",
  );

  assertCondition(
    afterFirst.status === "CONFIRMED",
    `expected CONFIRMED but received ${afterFirst.status}`,
  );

  assertCondition(
    afterFirst.result !== null,
    "CONFIRMED record must contain result",
  );

  const second =
    await usecase.execute(input);

  assertCondition(
    second.signature === first.signature,
    "second execution must return same signature",
  );

  assertCondition(
    second.assetId === first.assetId,
    "second execution must return same assetId",
  );

  assertCondition(
    second.leafIndex === first.leafIndex,
    "second execution must return same leafIndex",
  );

  assertNumberEquals(
    transaction.buildAndSignCalls,
    1,
    "confirmed replay must not build another transaction",
  );

  assertNumberEquals(
    transaction.broadcastCalls,
    1,
    "confirmed replay must not broadcast again",
  );

  assertNumberEquals(
    transaction.waitForFinalizedCalls,
    1,
    "confirmed replay must not wait again",
  );

  assertNumberEquals(
    transaction.parseMintResultCalls,
    1,
    "confirmed replay must not parse again",
  );

  console.log("CONFIRMED replay: OK");
}

async function verifyPayloadConflict(
  registry: FirestoreMintOperationRegistryRepository,
  productIds: Set<string>,
): Promise<void> {
  const productId =
    createProductId("payload-conflict");

  productIds.add(productId);

  const transaction =
    new FakeMintV2TransactionClient();

  const usecase =
    createUsecase(registry, transaction);

  const firstInput =
    createInput(productId);

  await usecase.execute(firstInput);

  const conflictingInput: MintV2UsecaseInput = {
    ...firstInput,
    metadata: {
      ...firstInput.metadata,
      uri: "https://example.com/different-metadata.json",
    },
  };

  let conflict: unknown;

  try {
    await usecase.execute(conflictingInput);
  } catch (error) {
    conflict = error;
  }

  assertCondition(
    conflict instanceof MintOperationPayloadConflictError,
    "different payload for same productId must raise MintOperationPayloadConflictError",
  );

  assertNumberEquals(
    transaction.buildAndSignCalls,
    1,
    "payload conflict must not build another transaction",
  );

  assertNumberEquals(
    transaction.broadcastCalls,
    1,
    "payload conflict must not broadcast another transaction",
  );

  console.log("Payload conflict: OK");
}

async function verifySubmittedRecovery(
  registry: FirestoreMintOperationRegistryRepository,
  productIds: Set<string>,
): Promise<void> {
  const productId =
    createProductId("submitted-recovery");

  productIds.add(productId);

  const input =
    createInput(productId);

  const payloadHash =
    createPayloadHash(input);

  await registry.reserve({
    productId,
    payloadHash,
    now: new Date(),
  });

  const signature = [
    "verify-submitted-signature",
    randomUUID(),
  ].join("-");

  const signedTransactionBase64 = Buffer
    .from(
      `verify-submitted-transaction:${signature}`,
      "utf8",
    )
    .toString("base64");

  await registry.markSubmitted({
    productId,
    payloadHash,
    signature,
    signedTransactionBase64,
    updatedAt: new Date(),
  });

  const transaction =
    new FakeMintV2TransactionClient();

  const usecase =
    createUsecase(registry, transaction);

  const result =
    await usecase.execute(input);

  assertNumberEquals(
    transaction.buildAndSignCalls,
    0,
    "SUBMITTED recovery must not build a new transaction",
  );

  assertNumberEquals(
    transaction.broadcastCalls,
    1,
    "SUBMITTED recovery must rebroadcast stored transaction",
  );

  assertCondition(
    result.signature === signature,
    "SUBMITTED recovery must preserve stored signature",
  );

  const record =
    await registry.getByProductId(productId);

  assertCondition(
    record !== null,
    "SUBMITTED recovery record must exist",
  );

  assertCondition(
    record.status === "CONFIRMED",
    `SUBMITTED recovery must finish CONFIRMED but received ${record.status}`,
  );

  console.log("SUBMITTED recovery: OK");
}

async function verifyRetryableBroadcastRecovery(
  registry: FirestoreMintOperationRegistryRepository,
  productIds: Set<string>,
): Promise<void> {
  const productId =
    createProductId("broadcast-retry");

  productIds.add(productId);

  const transaction =
    new FakeMintV2TransactionClient();

  transaction.failNextBroadcast = true;

  const usecase =
    createUsecase(registry, transaction);

  const input =
    createInput(productId);

  let firstError: unknown;

  try {
    await usecase.execute(input);
  } catch (error) {
    firstError = error;
  }

  assertCondition(
    firstError instanceof MintV2TransactionError,
    "simulated broadcast failure must propagate MintV2TransactionError",
  );

  assertCondition(
    firstError.kind === "RETRYABLE",
    "broadcast failure must be retryable",
  );

  const failedRecord =
    await registry.getByProductId(productId);

  assertCondition(
    failedRecord !== null,
    "retryable failed record must exist",
  );

  assertCondition(
    failedRecord.status === "FAILED_RETRYABLE",
    `expected FAILED_RETRYABLE but received ${failedRecord.status}`,
  );

  assertCondition(
    failedRecord.signature !== null,
    "FAILED_RETRYABLE after broadcast attempt must keep signature",
  );

  assertCondition(
    failedRecord.signedTransactionBase64 !== null,
    "FAILED_RETRYABLE after broadcast attempt must keep signed transaction bytes",
  );

  const originalSignature =
    failedRecord.signature;

  const originalSignedTransaction =
    failedRecord.signedTransactionBase64;

  assertNumberEquals(
    transaction.buildAndSignCalls,
    1,
    "first retryable attempt must build one transaction",
  );

  assertNumberEquals(
    transaction.broadcastCalls,
    1,
    "first retryable attempt must broadcast once",
  );

  const result =
    await usecase.execute(input);

  assertNumberEquals(
    transaction.buildAndSignCalls,
    1,
    "retry must not build a second transaction when signed bytes exist",
  );

  assertNumberEquals(
    transaction.broadcastCalls,
    2,
    "retry must rebroadcast stored transaction",
  );

  assertCondition(
    result.signature === originalSignature,
    "retry must preserve original signature",
  );

  const confirmed =
    await registry.getByProductId(productId);

  assertCondition(
    confirmed !== null,
    "retry confirmation record must exist",
  );

  assertCondition(
    confirmed.status === "CONFIRMED",
    `retry must finish CONFIRMED but received ${confirmed.status}`,
  );

  assertCondition(
    confirmed.signature === originalSignature,
    "confirmed retry must keep original signature",
  );

  assertCondition(
    confirmed.signedTransactionBase64 ===
      originalSignedTransaction,
    "confirmed retry must keep original transaction bytes",
  );

  console.log(
    "FAILED_RETRYABLE signed transaction recovery: OK",
  );
}

async function verifyRetryableBeforeSigning(
  registry: FirestoreMintOperationRegistryRepository,
  productIds: Set<string>,
): Promise<void> {
  const productId =
    createProductId("build-retry");

  productIds.add(productId);

  const transaction =
    new FakeMintV2TransactionClient();

  transaction.failNextBuildRetryable = true;

  const usecase =
    createUsecase(registry, transaction);

  const input =
    createInput(productId);

  let firstError: unknown;

  try {
    await usecase.execute(input);
  } catch (error) {
    firstError = error;
  }

  assertCondition(
    firstError instanceof MintV2TransactionError,
    "simulated build failure must propagate MintV2TransactionError",
  );

  assertCondition(
    firstError.kind === "RETRYABLE",
    "build failure must be retryable",
  );

  const failedRecord =
    await registry.getByProductId(productId);

  assertCondition(
    failedRecord !== null,
    "pre-sign retryable record must exist",
  );

  assertCondition(
    failedRecord.status === "FAILED_RETRYABLE",
    `expected FAILED_RETRYABLE but received ${failedRecord.status}`,
  );

  assertCondition(
    failedRecord.signature === null,
    "pre-sign retryable failure must not contain signature",
  );

  assertCondition(
    failedRecord.signedTransactionBase64 === null,
    "pre-sign retryable failure must not contain signed transaction",
  );

  assertNumberEquals(
    transaction.buildAndSignCalls,
    1,
    "first pre-sign retryable attempt must build once",
  );

  assertNumberEquals(
    transaction.broadcastCalls,
    0,
    "pre-sign failure must not broadcast",
  );

  const result =
    await usecase.execute(input);

  assertNumberEquals(
    transaction.buildAndSignCalls,
    2,
    "retry without signed bytes must build a new transaction",
  );

  assertNumberEquals(
    transaction.broadcastCalls,
    1,
    "retry after successful rebuild must broadcast once",
  );

  assertCondition(
    result.signature.length > 0,
    "retry result signature must exist",
  );

  console.log(
    "FAILED_RETRYABLE before signing recovery: OK",
  );
}

async function verifyFatalFailure(
  registry: FirestoreMintOperationRegistryRepository,
  productIds: Set<string>,
): Promise<void> {
  const productId =
    createProductId("fatal");

  productIds.add(productId);

  const transaction =
    new FakeMintV2TransactionClient();

  transaction.failNextBuildFatal = true;

  const usecase =
    createUsecase(registry, transaction);

  const input =
    createInput(productId);

  let firstError: unknown;

  try {
    await usecase.execute(input);
  } catch (error) {
    firstError = error;
  }

  assertCondition(
    firstError instanceof MintV2TransactionError,
    "simulated fatal build failure must propagate MintV2TransactionError",
  );

  assertCondition(
    firstError.kind === "FATAL",
    "simulated fatal failure must be fatal",
  );

  const failedRecord =
    await registry.getByProductId(productId);

  assertCondition(
    failedRecord !== null,
    "fatal record must exist",
  );

  assertCondition(
    failedRecord.status === "FAILED_FATAL",
    `expected FAILED_FATAL but received ${failedRecord.status}`,
  );

  let secondError: unknown;

  try {
    await usecase.execute(input);
  } catch (error) {
    secondError = error;
  }

  assertCondition(
    secondError instanceof MintV2UsecaseStoredFatalError,
    "FAILED_FATAL replay must return stored fatal error",
  );

  assertNumberEquals(
    transaction.buildAndSignCalls,
    1,
    "FAILED_FATAL replay must not build a new transaction",
  );

  assertNumberEquals(
    transaction.broadcastCalls,
    0,
    "FAILED_FATAL replay must never broadcast",
  );

  console.log("FAILED_FATAL replay: OK");
}

async function main(): Promise<void> {
  const registry =
    new FirestoreMintOperationRegistryRepository();

  const productIds =
    new Set<string>();

  console.log(
    "MintV2 idempotency verification:",
  );

  console.log(
    JSON.stringify(
      {
        cluster: env.solanaCluster,
        onChainMintEnabled: false,
        firestoreRegistry: true,
        cleanupEnabled: true,
      },
      null,
      2,
    ),
  );

  try {
    await verifyConfirmedIdempotency(
      registry,
      productIds,
    );

    await verifyPayloadConflict(
      registry,
      productIds,
    );

    await verifySubmittedRecovery(
      registry,
      productIds,
    );

    await verifyRetryableBroadcastRecovery(
      registry,
      productIds,
    );

    await verifyRetryableBeforeSigning(
      registry,
      productIds,
    );

    await verifyFatalFailure(
      registry,
      productIds,
    );

    console.log(
      JSON.stringify(
        {
          status: "OK",
          confirmedReplay: true,
          payloadConflict: true,
          submittedRecovery: true,
          retryableSignedTransactionRecovery: true,
          retryableBeforeSigningRecovery: true,
          fatalReplayBlocked: true,
          onChainMintCreated: false,
        },
        null,
        2,
      ),
    );

    console.log(
      "MintV2 idempotency verification: OK",
    );
  } finally {
    await deleteTestRecords(
      productIds,
    );

    console.log(
      [
        "MintV2 idempotency verification:",
        "test Firestore records cleaned up",
        `count=${productIds.size}`,
      ].join(" "),
    );
  }
}

main().catch(
  (error: unknown) => {
    console.error(
      "MintV2 idempotency verification: FAILED",
    );

    if (error instanceof Error) {
      console.error(error.message);
    } else {
      console.error(String(error));
    }

    process.exitCode = 1;
  },
);