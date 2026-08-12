// services/solana-bubblegum/src/application/merkle-tree-resolver.ts

import {
  createTreeV2,
  fetchTreeConfigFromSeeds,
} from "@metaplex-foundation/mpl-bubblegum";

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
  MerkleTreeRegistryPort,
  MerkleTreeRegistryRecord,
} from "./ports/merkle-tree-registry-port.js";

import {
  FeePayerTopUpUsecase,
} from "./fee-payer-top-up.js";

const TREE_VERIFY_ATTEMPTS = 10;
const TREE_VERIFY_DELAY_MS = 2_000;

function sleep(milliseconds: number): Promise<void> {
  return new Promise((resolve) => {
    setTimeout(resolve, milliseconds);
  });
}

export type MerkleTreeResolverConfig = {
  registryKey: string;
  cluster: string;
  maxDepth: number;
  maxBufferSize: number;
  canopyDepth: number;
  public: boolean;
};

export type ResolveMerkleTreeInput = {
  umi: Umi;
  feePayer: KeypairSigner;
  reserve: KeypairSigner;
};

export type ResolveMerkleTreeResult = {
  status: "existing" | "created";
  treeAddress: string;
  cluster: string;
  maxDepth: number;
  maxBufferSize: number;
  canopyDepth: number;
  public: boolean;
  txSignature: string;
};

export class MerkleTreeResolver {
  private readonly inFlight =
    new Map<string, Promise<ResolveMerkleTreeResult>>();

  constructor(
    private readonly registry: MerkleTreeRegistryPort,
    private readonly feePayerTopUp: FeePayerTopUpUsecase,
    private readonly config: MerkleTreeResolverConfig,
  ) {}

  async resolve(
    input: ResolveMerkleTreeInput,
  ): Promise<ResolveMerkleTreeResult> {
    this.validateConfig();

    const existingPromise =
      this.inFlight.get(this.config.registryKey);

    if (existingPromise) {
      return existingPromise;
    }

    const promise =
      this.resolveInternal(input)
        .finally(() => {
          this.inFlight.delete(
            this.config.registryKey,
          );
        });

    this.inFlight.set(
      this.config.registryKey,
      promise,
    );

    return promise;
  }

  private async resolveInternal(
    input: ResolveMerkleTreeInput,
  ): Promise<ResolveMerkleTreeResult> {
    const registered =
      await this.registry.getByKey(
        this.config.registryKey,
      );

    if (registered) {
      this.verifyRegisteredConfig(
        registered,
      );

      await this.verifyExistingTree(
        input.umi,
        registered,
      );

      return {
        status: "existing",
        treeAddress: registered.treeAddress,
        cluster: registered.cluster,
        maxDepth: registered.maxDepth,
        maxBufferSize: registered.maxBufferSize,
        canopyDepth: registered.canopyDepth,
        public: registered.public,
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
          "merkle_tree_resolver: fee payer funding unavailable",
          `feePayer=${topUpResult.feePayerAddress}`,
          `reserve=${topUpResult.reserveAddress}`,
          `feePayerBalanceSOL=${topUpResult.feePayerBalanceBeforeSOL}`,
          `reserveBalanceSOL=${topUpResult.reserveBalanceBeforeSOL}`,
        ].join(" "),
      );
    }

    const merkleTreeSigner =
      generateSigner(input.umi);

    const builder =
      await createTreeV2(
        input.umi,
        {
          merkleTree: merkleTreeSigner,
          payer: input.feePayer,
          treeCreator: input.umi.identity,
          maxDepth: this.config.maxDepth,
          maxBufferSize: this.config.maxBufferSize,
          canopyDepth: this.config.canopyDepth,
          public: this.config.public,
        },
      );

    const transactionResult =
      await builder.sendAndConfirm(
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

    const treeAddress =
      String(
        merkleTreeSigner.publicKey,
      );

    await this.verifyCreatedTree(
      input.umi,
      treeAddress,
    );

    const now =
      new Date();

    const record:
      MerkleTreeRegistryRecord = {
        treeAddress,
        cluster: this.config.cluster,
        maxDepth: this.config.maxDepth,
        maxBufferSize: this.config.maxBufferSize,
        canopyDepth: this.config.canopyDepth,
        public: this.config.public,
        txSignature,
        createdAt: now,
        updatedAt: now,
      };

    await this.registry.save(
      this.config.registryKey,
      record,
    );

    return {
      status: "created",
      treeAddress: record.treeAddress,
      cluster: record.cluster,
      maxDepth: record.maxDepth,
      maxBufferSize: record.maxBufferSize,
      canopyDepth: record.canopyDepth,
      public: record.public,
      txSignature: record.txSignature,
    };
  }

  private verifyRegisteredConfig(
    record: MerkleTreeRegistryRecord,
  ): void {
    if (
      record.cluster !==
      this.config.cluster
    ) {
      throw new Error(
        [
          "merkle_tree_resolver: cluster mismatch",
          `key=${this.config.registryKey}`,
          `expected=${this.config.cluster}`,
          `actual=${record.cluster}`,
        ].join(" "),
      );
    }

    if (
      record.maxDepth !==
      this.config.maxDepth
    ) {
      throw new Error(
        [
          "merkle_tree_resolver: maxDepth mismatch",
          `key=${this.config.registryKey}`,
          `expected=${this.config.maxDepth}`,
          `actual=${record.maxDepth}`,
        ].join(" "),
      );
    }

    if (
      record.maxBufferSize !==
      this.config.maxBufferSize
    ) {
      throw new Error(
        [
          "merkle_tree_resolver: maxBufferSize mismatch",
          `key=${this.config.registryKey}`,
          `expected=${this.config.maxBufferSize}`,
          `actual=${record.maxBufferSize}`,
        ].join(" "),
      );
    }

    if (
      record.canopyDepth !==
      this.config.canopyDepth
    ) {
      throw new Error(
        [
          "merkle_tree_resolver: canopyDepth mismatch",
          `key=${this.config.registryKey}`,
          `expected=${this.config.canopyDepth}`,
          `actual=${record.canopyDepth}`,
        ].join(" "),
      );
    }

    if (
      record.public !==
      this.config.public
    ) {
      throw new Error(
        [
          "merkle_tree_resolver: public mismatch",
          `key=${this.config.registryKey}`,
          `expected=${this.config.public}`,
          `actual=${record.public}`,
        ].join(" "),
      );
    }
  }

  private async verifyExistingTree(
    umi: Umi,
    record: MerkleTreeRegistryRecord,
  ): Promise<void> {
    try {
      await fetchTreeConfigFromSeeds(
        umi,
        {
          merkleTree:
            publicKey(
              record.treeAddress,
            ),
        },
        {
          commitment: "finalized",
        },
      );
    } catch {
      throw new Error(
        [
          "merkle_tree_resolver: registered tree not found on-chain",
          `key=${this.config.registryKey}`,
          `treeAddress=${record.treeAddress}`,
        ].join(" "),
      );
    }
  }

  private async verifyCreatedTree(
    umi: Umi,
    treeAddress: string,
  ): Promise<void> {
    let lastError: unknown = null;

    for (
      let attempt = 1;
      attempt <= TREE_VERIFY_ATTEMPTS;
      attempt += 1
    ) {
      try {
        await fetchTreeConfigFromSeeds(
          umi,
          {
            merkleTree:
              publicKey(
                treeAddress,
              ),
          },
          {
            commitment: "finalized",
          },
        );

        return;
      } catch (error) {
        lastError = error;

        if (
          attempt <
          TREE_VERIFY_ATTEMPTS
        ) {
          await sleep(
            TREE_VERIFY_DELAY_MS,
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
        "merkle_tree_resolver: created tree config not found on-chain",
        `key=${this.config.registryKey}`,
        `treeAddress=${treeAddress}`,
        `attempts=${TREE_VERIFY_ATTEMPTS}`,
        `lastError=${detail}`,
      ].join(" "),
    );
  }

  private validateConfig(): void {
    if (!this.config.registryKey) {
      throw new Error(
        "merkle_tree_resolver: registryKey is required",
      );
    }

    if (!this.config.cluster) {
      throw new Error(
        "merkle_tree_resolver: cluster is required",
      );
    }

    if (
      !Number.isInteger(
        this.config.maxDepth,
      ) ||
      this.config.maxDepth <= 0
    ) {
      throw new Error(
        "merkle_tree_resolver: maxDepth is invalid",
      );
    }

    if (
      !Number.isInteger(
        this.config.maxBufferSize,
      ) ||
      this.config.maxBufferSize <= 0
    ) {
      throw new Error(
        "merkle_tree_resolver: maxBufferSize is invalid",
      );
    }

    if (
      !Number.isInteger(
        this.config.canopyDepth,
      ) ||
      this.config.canopyDepth < 0 ||
      this.config.canopyDepth >
        this.config.maxDepth
    ) {
      throw new Error(
        "merkle_tree_resolver: canopyDepth is invalid",
      );
    }

    if (
      typeof this.config.public !==
      "boolean"
    ) {
      throw new Error(
        "merkle_tree_resolver: public is invalid",
      );
    }
  }
}