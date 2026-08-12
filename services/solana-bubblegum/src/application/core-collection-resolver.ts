// services/solana-bubblegum/src/application/core-collection-resolver.ts

import {
  createCollection,
  fetchCollection,
} from "@metaplex-foundation/mpl-core";

import {
  generateSigner,
  publicKey,
  type KeypairSigner,
  type Umi,
} from "@metaplex-foundation/umi";

import {
  base58,
} from "@metaplex-foundation/umi/serializers";

import type {
  CoreCollectionRegistryPort,
  CoreCollectionRegistryRecord,
} from "./ports/core-collection-registry-port.js";

import {
  FeePayerTopUpUsecase,
} from "./fee-payer-top-up.js";

const COLLECTION_VERIFY_ATTEMPTS = 10;
const COLLECTION_VERIFY_DELAY_MS = 2_000;

function sleep(milliseconds: number): Promise<void> {
  return new Promise((resolve) => {
    setTimeout(resolve, milliseconds);
  });
}

export type CoreCollectionResolverConfig = {
  cluster: string;
};

export type ResolveCoreCollectionInput = {
  tokenBlueprintId: string;
  name: string;
  metadataUri: string;
  umi: Umi;
  feePayer: KeypairSigner;
  reserve: KeypairSigner;
};

export type ResolveCoreCollectionResult = {
  status: "existing" | "created";
  tokenBlueprintId: string;
  collectionAddress: string;
  name: string;
  metadataUri: string;
  cluster: string;
  txSignature: string;
};

export class CoreCollectionResolver {
  private readonly inFlight =
    new Map<string, Promise<ResolveCoreCollectionResult>>();

  constructor(
    private readonly registry: CoreCollectionRegistryPort,
    private readonly feePayerTopUp: FeePayerTopUpUsecase,
    private readonly config: CoreCollectionResolverConfig,
  ) {}

  async resolve(
    input: ResolveCoreCollectionInput,
  ): Promise<ResolveCoreCollectionResult> {
    this.validateInput(input);

    const existingPromise =
      this.inFlight.get(input.tokenBlueprintId);

    if (existingPromise) {
      return existingPromise;
    }

    const promise =
      this.resolveInternal(input)
        .finally(() => {
          this.inFlight.delete(input.tokenBlueprintId);
        });

    this.inFlight.set(
      input.tokenBlueprintId,
      promise,
    );

    return promise;
  }

  private async resolveInternal(
    input: ResolveCoreCollectionInput,
  ): Promise<ResolveCoreCollectionResult> {
    const registered =
      await this.registry.getByTokenBlueprintId(
        input.tokenBlueprintId,
      );

    if (registered) {
      await this.verifyExistingCollection(
        input.umi,
        registered,
      );

      return {
        status: "existing",
        tokenBlueprintId: registered.tokenBlueprintId,
        collectionAddress: registered.collectionAddress,
        name: registered.name,
        metadataUri: registered.metadataUri,
        cluster: registered.cluster,
        txSignature: registered.txSignature,
      };
    }

    const topUpResult =
      await this.feePayerTopUp.execute({
        umi: input.umi,
        feePayer: input.feePayer,
        reserve: input.reserve,
      });

    if (
      topUpResult.status ===
      "reserve_insufficient"
    ) {
      throw new Error(
        [
          "core_collection_resolver: fee payer funding unavailable",
          `feePayer=${topUpResult.feePayerAddress}`,
          `reserve=${topUpResult.reserveAddress}`,
          `feePayerBalanceSOL=${topUpResult.feePayerBalanceBeforeSOL}`,
          `reserveBalanceSOL=${topUpResult.reserveBalanceBeforeSOL}`,
        ].join(" "),
      );
    }

    const collectionSigner =
      generateSigner(input.umi);

    const transactionResult =
      await createCollection(
        input.umi,
        {
          collection: collectionSigner,

          // SOL の支払いは fee payer が担当する。
          payer: input.feePayer,

          // MintV2 の collectionAuthority と一致させる。
          // runtime では umi.identity = mintAuthority。
          updateAuthority:
            input.umi.identity.publicKey,

          name: input.name,
          uri: input.metadataUri,

          plugins: [
            {
              type: "BubblegumV2",
            },
          ],
        },
      ).sendAndConfirm(
        input.umi,
        {
          confirm: {
            commitment: "finalized",
          },
        },
      );

    const txSignature =
      base58.deserialize(
        transactionResult.signature,
      )[0];

    const collectionAddress =
      String(collectionSigner.publicKey);

    await this.verifyCreatedCollection(
      input.umi,
      collectionAddress,
    );

    const now = new Date();

    const record:
      CoreCollectionRegistryRecord = {
        tokenBlueprintId:
          input.tokenBlueprintId,

        collectionAddress,

        name:
          input.name,

        metadataUri:
          input.metadataUri,

        cluster:
          this.config.cluster,

        txSignature,

        createdAt:
          now,

        updatedAt:
          now,
      };

    await this.registry.save(record);

    return {
      status: "created",
      tokenBlueprintId: record.tokenBlueprintId,
      collectionAddress: record.collectionAddress,
      name: record.name,
      metadataUri: record.metadataUri,
      cluster: record.cluster,
      txSignature: record.txSignature,
    };
  }

  private async verifyExistingCollection(
    umi: Umi,
    record: CoreCollectionRegistryRecord,
  ): Promise<void> {
    if (
      record.cluster !==
      this.config.cluster
    ) {
      throw new Error(
        [
          "core_collection_resolver: cluster mismatch",
          `tokenBlueprintId=${record.tokenBlueprintId}`,
          `expected=${this.config.cluster}`,
          `actual=${record.cluster}`,
        ].join(" "),
      );
    }

    try {
      await fetchCollection(
        umi,
        publicKey(record.collectionAddress),
        {
          commitment: "finalized",
        },
      );
    } catch (error) {
      const detail =
        error instanceof Error
          ? error.message
          : String(error);

      throw new Error(
        [
          "core_collection_resolver: registered collection not found on-chain",
          `tokenBlueprintId=${record.tokenBlueprintId}`,
          `collectionAddress=${record.collectionAddress}`,
          `lastError=${detail}`,
        ].join(" "),
      );
    }
  }

  private async verifyCreatedCollection(
    umi: Umi,
    collectionAddress: string,
  ): Promise<void> {
    let lastError: unknown = null;

    for (
      let attempt = 1;
      attempt <= COLLECTION_VERIFY_ATTEMPTS;
      attempt += 1
    ) {
      try {
        await fetchCollection(
          umi,
          publicKey(collectionAddress),
          {
            commitment: "finalized",
          },
        );

        return;
      } catch (error) {
        lastError = error;

        if (
          attempt <
          COLLECTION_VERIFY_ATTEMPTS
        ) {
          await sleep(
            COLLECTION_VERIFY_DELAY_MS,
          );
        }
      }
    }

    const detail =
      lastError instanceof Error
        ? lastError.message
        : String(lastError);

    throw new Error(
      [
        "core_collection_resolver: created collection not found on-chain",
        `collectionAddress=${collectionAddress}`,
        `attempts=${COLLECTION_VERIFY_ATTEMPTS}`,
        `lastError=${detail}`,
      ].join(" "),
    );
  }

  private validateInput(
    input: ResolveCoreCollectionInput,
  ): void {
    if (!input.tokenBlueprintId) {
      throw new Error(
        "core_collection_resolver: tokenBlueprintId is required",
      );
    }

    if (!input.name) {
      throw new Error(
        "core_collection_resolver: name is required",
      );
    }

    if (!input.metadataUri) {
      throw new Error(
        "core_collection_resolver: metadataUri is required",
      );
    }

    if (!this.config.cluster) {
      throw new Error(
        "core_collection_resolver: cluster is required",
      );
    }
  }
}