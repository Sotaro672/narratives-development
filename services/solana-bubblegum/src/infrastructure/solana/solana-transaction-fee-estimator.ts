// services/solana-bubblegum/src/infrastructure/solana/solana-transaction-fee-estimator.ts

import { Buffer } from "node:buffer";

import { createTreeV2 } from "@metaplex-foundation/mpl-bubblegum";
import { createCollection } from "@metaplex-foundation/mpl-core";
import {
  generateSigner,
  type KeypairSigner,
  type TransactionBuilder,
  type Umi,
} from "@metaplex-foundation/umi";

import type {
  MintFundingCostEstimatorInput,
  MintFundingCostEstimatorPort,
  MintFundingCostEstimatorResult,
} from "../../application/mint-funding-estimate.js";

import {
  buildBubblegumMintV2TransactionBuilder,
} from "./bubblegum-mint-v2-transaction-builder.js";

type GetFeeForMessageResponse = {
  context: {
    slot: number;
  };
  value: number | null;
};

function requireFeeLamports(value: number | null): bigint {
  if (value === null) {
    throw new Error(
      "solana_transaction_fee_estimator: getFeeForMessage returned null",
    );
  }

  if (!Number.isSafeInteger(value) || value < 0) {
    throw new Error(
      [
        "solana_transaction_fee_estimator: invalid transaction fee",
        `value=${String(value)}`,
      ].join(" "),
    );
  }

  return BigInt(value);
}

/**
 * TransactionBuilderから現在のblockhashを使ったtransaction messageを生成し、
 * Solana RPC getFeeForMessageでtransaction feeを取得する。
 *
 * sign / sendは行わない。
 */
async function estimateTransactionFeeLamports(
  umi: Umi,
  feePayer: KeypairSigner,
  builder: TransactionBuilder,
): Promise<bigint> {
  const transaction = await builder
    .setFeePayer(feePayer)
    .buildWithLatestBlockhash(umi, {
      commitment: "confirmed",
    });

  const messageBase64 = Buffer.from(
    transaction.serializedMessage,
  ).toString("base64");

  const response = await umi.rpc.call<
    GetFeeForMessageResponse,
    [string]
  >(
    "getFeeForMessage",
    [messageBase64],
    {
      commitment: "confirmed",
    },
  );

  return requireFeeLamports(response.value);
}

/**
 * TransactionBuilderが作成するon-chain accountについて、
 * rent exemptionに必要なlamportsを取得する。
 *
 * account自体は作成しない。
 */
async function estimateRentLamports(
  umi: Umi,
  builder: TransactionBuilder,
): Promise<bigint> {
  const rent = await builder.getRentCreatedOnChain(umi);

  if (rent.basisPoints < 0n) {
    throw new Error(
      "solana_transaction_fee_estimator: estimated rent is negative",
    );
  }

  return rent.basisPoints;
}

/**
 * Shared Merkle Tree新規作成用TransactionBuilderを生成する。
 * signerは見積専用でありtransactionは送信しない。
 */
async function buildMerkleTreeCreationBuilder(
  input: MintFundingCostEstimatorInput,
  merkleTreeSigner: KeypairSigner,
): Promise<TransactionBuilder> {
  return createTreeV2(input.umi, {
    merkleTree: merkleTreeSigner,
    payer: input.feePayer,
    treeCreator: input.umi.identity,
    maxDepth: input.merkleTreeConfig.maxDepth,
    maxBufferSize: input.merkleTreeConfig.maxBufferSize,
    canopyDepth: input.merkleTreeConfig.canopyDepth,
    public: input.merkleTreeConfig.public,
  });
}

/**
 * TokenBlueprint用Core Collection新規作成のTransactionBuilderを生成する。
 *
 * metadataUriは外部入力の実metadata URIではなく、
 * application/mint-funding-estimate.tsから渡された見積専用固定URIを使用する。
 *
 * 実Mintと同じくBubblegumV2 pluginを付与する。
 */
function buildCoreCollectionCreationBuilder(
  input: MintFundingCostEstimatorInput,
  collectionSigner: KeypairSigner,
): TransactionBuilder {
  return createCollection(input.umi, {
    collection: collectionSigner,
    payer: input.feePayer,
    updateAuthority: input.umi.identity.publicKey,
    name: input.name,
    uri: input.metadataUri,
    plugins: [
      {
        type: "BubblegumV2",
      },
    ],
  });
}

/**
 * Solana / Bubblegum V2 のtransaction feeとaccount rentをread-onlyで見積もる。
 *
 * IMPORTANT:
 * - transactionは送信しない
 * - reserveからSOLを送金しない
 * - Merkle Treeを作成しない
 * - Core Collectionを作成しない
 * - cNFTをMintしない
 * - metadata uploadを行わない
 * - input.metadataUriは見積専用固定URIを前提とする
 */
export class SolanaTransactionFeeEstimator
  implements MintFundingCostEstimatorPort
{
  async estimate(
    input: MintFundingCostEstimatorInput,
  ): Promise<MintFundingCostEstimatorResult> {
    this.validateInput(input);

    const newMerkleTreeSigner =
      input.merkleTree === null
        ? generateSigner(input.umi)
        : null;

    const newCoreCollectionSigner =
      input.coreCollection === null
        ? generateSigner(input.umi)
        : null;

    const treeAddress =
      input.merkleTree !== null
        ? input.merkleTree.treeAddress
        : String(newMerkleTreeSigner!.publicKey);

    const coreCollectionAddress =
      input.coreCollection !== null
        ? input.coreCollection.collectionAddress
        : String(newCoreCollectionSigner!.publicKey);

    /**
     * 実MintとSOL見積で同一のBubblegum MintV2 TransactionBuilderを使用する。
     *
     * metadata.uriにはapplication層から渡された見積専用固定URIを使う。
     */
    const mintBuilder = buildBubblegumMintV2TransactionBuilder(
      input.umi,
      {
        treeAddress,
        leafOwnerAddress: input.leafOwnerAddress,
        leafDelegateAddress: null,
        coreCollectionAddress,
        metadata: {
          name: input.name,
          symbol: input.symbol,
          uri: input.metadataUri,
          sellerFeeBasisPoints: 0,
          primarySaleHappened: false,
          isMutable: false,
          creators: [],
        },
      },
    );

    const mintTransactionFeePromise =
      estimateTransactionFeeLamports(
        input.umi,
        input.feePayer,
        mintBuilder,
      );

    let merkleTreeCreationTransactionFeePromise: Promise<bigint> =
      Promise.resolve(0n);

    let merkleTreeCreationRentPromise: Promise<bigint> =
      Promise.resolve(0n);

    if (newMerkleTreeSigner !== null) {
      const merkleTreeBuilder =
        await buildMerkleTreeCreationBuilder(
          input,
          newMerkleTreeSigner,
        );

      merkleTreeCreationTransactionFeePromise =
        estimateTransactionFeeLamports(
          input.umi,
          input.feePayer,
          merkleTreeBuilder,
        );

      merkleTreeCreationRentPromise =
        estimateRentLamports(
          input.umi,
          merkleTreeBuilder,
        );
    }

    let coreCollectionCreationTransactionFeePromise: Promise<bigint> =
      Promise.resolve(0n);

    let coreCollectionCreationRentPromise: Promise<bigint> =
      Promise.resolve(0n);

    if (newCoreCollectionSigner !== null) {
      const coreCollectionBuilder =
        buildCoreCollectionCreationBuilder(
          input,
          newCoreCollectionSigner,
        );

      coreCollectionCreationTransactionFeePromise =
        estimateTransactionFeeLamports(
          input.umi,
          input.feePayer,
          coreCollectionBuilder,
        );

      coreCollectionCreationRentPromise =
        estimateRentLamports(
          input.umi,
          coreCollectionBuilder,
        );
    }

    const [
      mintTransactionFeePerItemLamports,
      merkleTreeCreationTransactionFeeLamports,
      merkleTreeCreationRentLamports,
      coreCollectionCreationTransactionFeeLamports,
      coreCollectionCreationRentLamports,
    ] = await Promise.all([
      mintTransactionFeePromise,
      merkleTreeCreationTransactionFeePromise,
      merkleTreeCreationRentPromise,
      coreCollectionCreationTransactionFeePromise,
      coreCollectionCreationRentPromise,
    ]);

    return {
      mintTransactionFeePerItemLamports,
      merkleTreeCreationTransactionFeeLamports,
      merkleTreeCreationRentLamports,
      coreCollectionCreationTransactionFeeLamports,
      coreCollectionCreationRentLamports,
    };
  }

  private validateInput(
    input: MintFundingCostEstimatorInput,
  ): void {
    if (!input) {
      throw new Error(
        "solana_transaction_fee_estimator: input is required",
      );
    }

    if (!input.umi) {
      throw new Error(
        "solana_transaction_fee_estimator: umi is required",
      );
    }

    if (!input.feePayer) {
      throw new Error(
        "solana_transaction_fee_estimator: feePayer is required",
      );
    }

    if (!input.tokenBlueprintId) {
      throw new Error(
        "solana_transaction_fee_estimator: tokenBlueprintId is required",
      );
    }

    if (!input.leafOwnerAddress) {
      throw new Error(
        "solana_transaction_fee_estimator: leafOwnerAddress is required",
      );
    }

    if (!input.name) {
      throw new Error(
        "solana_transaction_fee_estimator: name is required",
      );
    }

    if (typeof input.symbol !== "string") {
      throw new Error(
        "solana_transaction_fee_estimator: symbol must be string",
      );
    }

    /**
     * metadataUri自体はEstimator内部では必要。
     * ただし実metadataUriではなくapplication層が設定した
     * 見積専用固定URIを受け取る。
     */
    if (!input.metadataUri) {
      throw new Error(
        "solana_transaction_fee_estimator: estimate metadataUri is required",
      );
    }

    if (!input.merkleTreeConfig.registryKey) {
      throw new Error(
        "solana_transaction_fee_estimator: merkleTreeConfig.registryKey is required",
      );
    }

    if (!input.merkleTreeConfig.cluster) {
      throw new Error(
        "solana_transaction_fee_estimator: merkleTreeConfig.cluster is required",
      );
    }

    if (
      !Number.isInteger(
        input.merkleTreeConfig.maxDepth,
      ) ||
      input.merkleTreeConfig.maxDepth <= 0
    ) {
      throw new Error(
        "solana_transaction_fee_estimator: merkleTreeConfig.maxDepth is invalid",
      );
    }

    if (
      !Number.isInteger(
        input.merkleTreeConfig.maxBufferSize,
      ) ||
      input.merkleTreeConfig.maxBufferSize <= 0
    ) {
      throw new Error(
        "solana_transaction_fee_estimator: merkleTreeConfig.maxBufferSize is invalid",
      );
    }

    if (
      !Number.isInteger(
        input.merkleTreeConfig.canopyDepth,
      ) ||
      input.merkleTreeConfig.canopyDepth < 0 ||
      input.merkleTreeConfig.canopyDepth >
        input.merkleTreeConfig.maxDepth
    ) {
      throw new Error(
        "solana_transaction_fee_estimator: merkleTreeConfig.canopyDepth is invalid",
      );
    }

    if (
      typeof input.merkleTreeConfig.public !==
      "boolean"
    ) {
      throw new Error(
        "solana_transaction_fee_estimator: merkleTreeConfig.public is invalid",
      );
    }

    /**
     * Runtimeではumi.payer = feePayerの前提。
     * 見積と実Mintのpayerを一致させる。
     */
    if (
      String(input.umi.payer.publicKey) !==
      String(input.feePayer.publicKey)
    ) {
      throw new Error(
        [
          "solana_transaction_fee_estimator: umi payer and fee payer mismatch",
          `umiPayer=${String(input.umi.payer.publicKey)}`,
          `feePayer=${String(input.feePayer.publicKey)}`,
        ].join(" "),
      );
    }
  }
}