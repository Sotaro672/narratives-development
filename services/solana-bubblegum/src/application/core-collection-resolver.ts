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


export type CoreCollectionResolverConfig = {
  cluster: string;
};


export type ResolveCoreCollectionInput = {
  tokenBlueprintId: string;

  name: string;

  metadataUri: string;

  umi: Umi;

  feePayer:
    KeypairSigner;

  reserve:
    KeypairSigner;
};


export type ResolveCoreCollectionResult = {
  status:
    | "existing"
    | "created";

  tokenBlueprintId: string;

  collectionAddress: string;

  name: string;

  metadataUri: string;

  cluster: string;

  txSignature: string;
};


export class CoreCollectionResolver {
  private readonly inFlight =
    new Map<
      string,
      Promise<ResolveCoreCollectionResult>
    >();


  constructor(
    private readonly registry:
      CoreCollectionRegistryPort,

    private readonly feePayerTopUp:
      FeePayerTopUpUsecase,

    private readonly config:
      CoreCollectionResolverConfig,
  ) {}


  async resolve(
    input: ResolveCoreCollectionInput,
  ): Promise<ResolveCoreCollectionResult> {
    this.validateInput(
      input,
    );


    const existingPromise =
      this.inFlight.get(
        input.tokenBlueprintId,
      );


    if (existingPromise) {
      return existingPromise;
    }


    const promise =
      this.resolveInternal(
        input,
      )
        .finally(
          () => {
            this.inFlight.delete(
              input.tokenBlueprintId,
            );
          },
        );


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
      await this.registry
        .getByTokenBlueprintId(
          input.tokenBlueprintId,
        );


    if (registered) {
      await this.verifyExistingCollection(
        input.umi,
        registered,
      );


      return {
        status:
          "existing",

        tokenBlueprintId:
          registered.tokenBlueprintId,

        collectionAddress:
          registered.collectionAddress,

        name:
          registered.name,

        metadataUri:
          registered.metadataUri,

        cluster:
          registered.cluster,

        txSignature:
          registered.txSignature,
      };
    }


    const topUpResult =
      await this.feePayerTopUp
        .execute({
          umi:
            input.umi,

          feePayer:
            input.feePayer,

          reserve:
            input.reserve,
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
        ].join(
          " ",
        ),
      );
    }


    const collectionSigner =
      generateSigner(
        input.umi,
      );


    const transactionResult =
      await createCollection(
        input.umi,
        {
          collection:
            collectionSigner,

          name:
            input.name,

          uri:
            input.metadataUri,

          plugins: [
            {
              type:
                "BubblegumV2",
            },
          ],
        },
      )
        .sendAndConfirm(
          input.umi,
        );


    const txSignature =
      base58.deserialize(
        transactionResult.signature,
      )[0];


    await fetchCollection(
      input.umi,
      collectionSigner.publicKey,
    );


    const now =
      new Date();


    const record:
      CoreCollectionRegistryRecord = {
        tokenBlueprintId:
          input.tokenBlueprintId,

        collectionAddress:
          String(
            collectionSigner.publicKey,
          ),

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


    await this.registry.save(
      record,
    );


    return {
      status:
        "created",

      tokenBlueprintId:
        record.tokenBlueprintId,

      collectionAddress:
        record.collectionAddress,

      name:
        record.name,

      metadataUri:
        record.metadataUri,

      cluster:
        record.cluster,

      txSignature:
        record.txSignature,
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
        ].join(
          " ",
        ),
      );
    }


    try {
      await fetchCollection(
        umi,
        publicKey(
          record.collectionAddress,
        ),
      );
    } catch {
      throw new Error(
        [
          "core_collection_resolver: registered collection not found on-chain",
          `tokenBlueprintId=${record.tokenBlueprintId}`,
          `collectionAddress=${record.collectionAddress}`,
        ].join(
          " ",
        ),
      );
    }
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