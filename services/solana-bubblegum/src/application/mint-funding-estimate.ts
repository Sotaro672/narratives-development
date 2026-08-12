// services/solana-bubblegum/src/application/mint-funding-estimate.ts

import { type KeypairSigner, type Umi } from "@metaplex-foundation/umi";

import type {
  CoreCollectionRegistryPort,
  CoreCollectionRegistryRecord,
} from "./ports/core-collection-registry-port.js";

import type {
  MerkleTreeRegistryPort,
  MerkleTreeRegistryRecord,
} from "./ports/merkle-tree-registry-port.js";

const LAMPORTS_PER_SOL = 1_000_000_000n;
const DEFAULT_TRANSACTION_FEE_BUFFER_SOL = 0.001;

/**
 * SOL見積専用の固定metadata URI。
 *
 * IMPORTANT:
 * - 実Mintのmetadata URIではない。
 * - GCS / metadata uploadを行わない。
 * - Firestore等へ保存しない。
 * - Core Collection / Mint transactionの費用見積にのみ使用する。
 * - 固定長にすることで、実metadataの内容やGCS容量に依存しない見積とする。
 */
const ESTIMATE_METADATA_URI =
  "https://metadata.invalid/amol-mint-estimate.json";

export type MintFundingEstimateConfig = {
  cluster: string;
  merkleTreeRegistryKey: string;
  merkleTreeMaxDepth: number;
  merkleTreeMaxBufferSize: number;
  merkleTreeCanopyDepth: number;
  merkleTreePublic: boolean;
  feePayerTargetSOL: number;
  reserveMinimumSOL: number;
  transactionFeeBufferSOL?: number;
};

export type MintFundingEstimateInput = {
  tokenBlueprintId: string;
  mintQuantity: number;
  leafOwnerAddress: string;
  name: string;
  symbol: string;
  umi: Umi;
  feePayer: KeypairSigner;
  reserve: KeypairSigner;
};

export type MintFundingMerkleTreeConfig = {
  registryKey: string;
  cluster: string;
  maxDepth: number;
  maxBufferSize: number;
  canopyDepth: number;
  public: boolean;
};

/**
 * Infrastructure側のtransaction/rent estimatorへ渡す内部入力。
 *
 * metadataUriは公開APIやTokenBlueprintから受け取らず、
 * Application層が見積専用固定URIを設定する。
 */
export type MintFundingCostEstimatorInput = {
  tokenBlueprintId: string;
  leafOwnerAddress: string;
  name: string;
  symbol: string;
  metadataUri: string;
  umi: Umi;
  feePayer: KeypairSigner;
  merkleTree: MerkleTreeRegistryRecord | null;
  coreCollection: CoreCollectionRegistryRecord | null;
  merkleTreeConfig: MintFundingMerkleTreeConfig;
};

export type MintFundingCostEstimatorResult = {
  mintTransactionFeePerItemLamports: bigint;
  merkleTreeCreationTransactionFeeLamports: bigint;
  merkleTreeCreationRentLamports: bigint;
  coreCollectionCreationTransactionFeeLamports: bigint;
  coreCollectionCreationRentLamports: bigint;
};

/**
 * SOL費用の算出はInfrastructure側へ委譲する。
 *
 * Application層では、
 * - 現在残高
 * - registry上の既存リソース
 * - funding policy
 * を組み合わせて最終的な見積を構成する。
 *
 * このPortの実装ではread-onlyで見積を行い、
 * transactionの送信やreserveからの送金は行わない。
 */
export interface MintFundingCostEstimatorPort {
  estimate(
    input: MintFundingCostEstimatorInput,
  ): Promise<MintFundingCostEstimatorResult>;
}

export type MintFundingEstimateResult = {
  cluster: string;
  mintQuantity: number;

  reserve: {
    address: string;
    balanceLamports: string;
    balanceSol: number;
    minimumLamports: string;
    minimumSol: number;
  };

  feePayer: {
    address: string;
    balanceLamports: string;
    balanceSol: number;
    targetLamports: string;
    targetSol: number;
  };

  resources: {
    sharedMerkleTreeExists: boolean;
    sharedMerkleTreeAddress: string | null;
    coreCollectionExists: boolean;
    coreCollectionAddress: string | null;
  };

  estimate: {
    mintTransactionFeePerItemLamports: string;
    mintTransactionFeePerItemSol: number;
    mintTransactionFeeTotalLamports: string;
    mintTransactionFeeTotalSol: number;

    merkleTreeCreationTransactionFeeLamports: string;
    merkleTreeCreationTransactionFeeSol: number;
    merkleTreeCreationRentLamports: string;
    merkleTreeCreationRentSol: number;
    merkleTreeCreationCostLamports: string;
    merkleTreeCreationCostSol: number;

    coreCollectionCreationTransactionFeeLamports: string;
    coreCollectionCreationTransactionFeeSol: number;
    coreCollectionCreationRentLamports: string;
    coreCollectionCreationRentSol: number;
    coreCollectionCreationCostLamports: string;
    coreCollectionCreationCostSol: number;

    provisioningCostLamports: string;
    provisioningCostSol: number;

    estimatedNetworkCostLamports: string;
    estimatedNetworkCostSol: number;

    requiredFeePayerBalanceLamports: string;
    requiredFeePayerBalanceSol: number;

    estimatedReserveTopUpLamports: string;
    estimatedReserveTopUpSol: number;

    reserveTransferFeeBufferLamports: string;
    reserveTransferFeeBufferSol: number;

    requiredReserveForTopUpLamports: string;
    requiredReserveForTopUpSol: number;

    sufficient: boolean;
  };
};

function solToLamports(value: number): bigint {
  if (!Number.isFinite(value) || value < 0) {
    throw new Error(
      "mint_funding_estimate: invalid SOL amount",
    );
  }

  return BigInt(
    Math.floor(
      value * Number(LAMPORTS_PER_SOL),
    ),
  );
}

function lamportsToSol(value: bigint): number {
  return Number(value) / Number(LAMPORTS_PER_SOL);
}

function maxLamports(
  left: bigint,
  right: bigint,
): bigint {
  return left >= right ? left : right;
}

function requireNonNegativeLamports(
  field: string,
  value: bigint,
): bigint {
  if (
    typeof value !== "bigint" ||
    value < 0n
  ) {
    throw new Error(
      [
        "mint_funding_estimate: invalid estimator result",
        `field=${field}`,
      ].join(" "),
    );
  }

  return value;
}

export class MintFundingEstimateUsecase {
  constructor(
    private readonly merkleTreeRegistry: MerkleTreeRegistryPort,
    private readonly coreCollectionRegistry: CoreCollectionRegistryPort,
    private readonly costEstimator: MintFundingCostEstimatorPort,
    private readonly config: MintFundingEstimateConfig,
  ) {}

  async execute(
    input: MintFundingEstimateInput,
  ): Promise<MintFundingEstimateResult> {
    this.validateConfig();
    this.validateInput(input);

    const transactionFeeBufferSOL =
      this.config.transactionFeeBufferSOL ??
      DEFAULT_TRANSACTION_FEE_BUFFER_SOL;

    const [
      feePayerBalance,
      reserveBalance,
      merkleTree,
      coreCollection,
    ] = await Promise.all([
      input.umi.rpc.getBalance(
        input.feePayer.publicKey,
        { commitment: "finalized" },
      ),
      input.umi.rpc.getBalance(
        input.reserve.publicKey,
        { commitment: "finalized" },
      ),
      this.merkleTreeRegistry.getByKey(
        this.config.merkleTreeRegistryKey,
      ),
      this.coreCollectionRegistry.getByTokenBlueprintId(
        input.tokenBlueprintId,
      ),
    ]);

    this.validateRegisteredMerkleTree(
      merkleTree,
    );

    this.validateRegisteredCoreCollection(
      input.tokenBlueprintId,
      coreCollection,
    );

    /**
     * metadataUriは実metadataを参照しない。
     *
     * 初回Mint前はmetadataUriがまだ存在しないため、
     * transaction/rent見積に必要なURIには固定値を使用する。
     */
    const costEstimate =
      await this.costEstimator.estimate({
        tokenBlueprintId:
          input.tokenBlueprintId,
        leafOwnerAddress:
          input.leafOwnerAddress,
        name:
          input.name,
        symbol:
          input.symbol,
        metadataUri:
          ESTIMATE_METADATA_URI,
        umi:
          input.umi,
        feePayer:
          input.feePayer,
        merkleTree,
        coreCollection,
        merkleTreeConfig: {
          registryKey:
            this.config.merkleTreeRegistryKey,
          cluster:
            this.config.cluster,
          maxDepth:
            this.config.merkleTreeMaxDepth,
          maxBufferSize:
            this.config.merkleTreeMaxBufferSize,
          canopyDepth:
            this.config.merkleTreeCanopyDepth,
          public:
            this.config.merkleTreePublic,
        },
      });

    const mintTransactionFeePerItemLamports =
      requireNonNegativeLamports(
        "mintTransactionFeePerItemLamports",
        costEstimate
          .mintTransactionFeePerItemLamports,
      );

    const rawMerkleTreeCreationTransactionFeeLamports =
      requireNonNegativeLamports(
        "merkleTreeCreationTransactionFeeLamports",
        costEstimate
          .merkleTreeCreationTransactionFeeLamports,
      );

    const rawMerkleTreeCreationRentLamports =
      requireNonNegativeLamports(
        "merkleTreeCreationRentLamports",
        costEstimate
          .merkleTreeCreationRentLamports,
      );

    const rawCoreCollectionCreationTransactionFeeLamports =
      requireNonNegativeLamports(
        "coreCollectionCreationTransactionFeeLamports",
        costEstimate
          .coreCollectionCreationTransactionFeeLamports,
      );

    const rawCoreCollectionCreationRentLamports =
      requireNonNegativeLamports(
        "coreCollectionCreationRentLamports",
        costEstimate
          .coreCollectionCreationRentLamports,
      );

    const mintTransactionFeeTotalLamports =
      mintTransactionFeePerItemLamports *
      BigInt(input.mintQuantity);

    const merkleTreeCreationTransactionFeeLamports =
      merkleTree === null
        ? rawMerkleTreeCreationTransactionFeeLamports
        : 0n;

    const merkleTreeCreationRentLamports =
      merkleTree === null
        ? rawMerkleTreeCreationRentLamports
        : 0n;

    const merkleTreeCreationCostLamports =
      merkleTreeCreationTransactionFeeLamports +
      merkleTreeCreationRentLamports;

    const coreCollectionCreationTransactionFeeLamports =
      coreCollection === null
        ? rawCoreCollectionCreationTransactionFeeLamports
        : 0n;

    const coreCollectionCreationRentLamports =
      coreCollection === null
        ? rawCoreCollectionCreationRentLamports
        : 0n;

    const coreCollectionCreationCostLamports =
      coreCollectionCreationTransactionFeeLamports +
      coreCollectionCreationRentLamports;

    const provisioningCostLamports =
      merkleTreeCreationCostLamports +
      coreCollectionCreationCostLamports;

    const estimatedNetworkCostLamports =
      mintTransactionFeeTotalLamports +
      provisioningCostLamports;

    const feePayerBalanceLamports =
      feePayerBalance.basisPoints;

    const reserveBalanceLamports =
      reserveBalance.basisPoints;

    const feePayerTargetLamports =
      solToLamports(
        this.config.feePayerTargetSOL,
      );

    const reserveMinimumLamports =
      solToLamports(
        this.config.reserveMinimumSOL,
      );

    const reserveTransferFeeBufferLamports =
      solToLamports(
        transactionFeeBufferSOL,
      );

    /**
     * fee payerは通常のtarget残高を満たしつつ、
     * 今回の見積ネットワーク費用以上を保持できる状態を
     * 必要残高として扱う。
     */
    const requiredFeePayerBalanceLamports =
      maxLamports(
        feePayerTargetLamports,
        estimatedNetworkCostLamports,
      );

    const estimatedReserveTopUpLamports =
      feePayerBalanceLamports >=
      requiredFeePayerBalanceLamports
        ? 0n
        : requiredFeePayerBalanceLamports -
          feePayerBalanceLamports;

    /**
     * reserve minimumとtransfer fee bufferは、
     * 実際にreserveからfee payerへ補充が必要な場合のみ
     * funding可否判定へ含める。
     */
    const requiredReserveForTopUpLamports =
      estimatedReserveTopUpLamports === 0n
        ? 0n
        : estimatedReserveTopUpLamports +
          reserveMinimumLamports +
          reserveTransferFeeBufferLamports;

    const sufficient =
      estimatedReserveTopUpLamports === 0n ||
      reserveBalanceLamports >=
        requiredReserveForTopUpLamports;

    return {
      cluster:
        this.config.cluster,
      mintQuantity:
        input.mintQuantity,

      reserve: {
        address:
          String(
            input.reserve.publicKey,
          ),
        balanceLamports:
          reserveBalanceLamports.toString(),
        balanceSol:
          lamportsToSol(
            reserveBalanceLamports,
          ),
        minimumLamports:
          reserveMinimumLamports.toString(),
        minimumSol:
          this.config.reserveMinimumSOL,
      },

      feePayer: {
        address:
          String(
            input.feePayer.publicKey,
          ),
        balanceLamports:
          feePayerBalanceLamports.toString(),
        balanceSol:
          lamportsToSol(
            feePayerBalanceLamports,
          ),
        targetLamports:
          feePayerTargetLamports.toString(),
        targetSol:
          this.config.feePayerTargetSOL,
      },

      resources: {
        sharedMerkleTreeExists:
          merkleTree !== null,
        sharedMerkleTreeAddress:
          merkleTree?.treeAddress ??
          null,
        coreCollectionExists:
          coreCollection !== null,
        coreCollectionAddress:
          coreCollection
            ?.collectionAddress ??
          null,
      },

      estimate: {
        mintTransactionFeePerItemLamports:
          mintTransactionFeePerItemLamports
            .toString(),
        mintTransactionFeePerItemSol:
          lamportsToSol(
            mintTransactionFeePerItemLamports,
          ),

        mintTransactionFeeTotalLamports:
          mintTransactionFeeTotalLamports
            .toString(),
        mintTransactionFeeTotalSol:
          lamportsToSol(
            mintTransactionFeeTotalLamports,
          ),

        merkleTreeCreationTransactionFeeLamports:
          merkleTreeCreationTransactionFeeLamports
            .toString(),
        merkleTreeCreationTransactionFeeSol:
          lamportsToSol(
            merkleTreeCreationTransactionFeeLamports,
          ),

        merkleTreeCreationRentLamports:
          merkleTreeCreationRentLamports
            .toString(),
        merkleTreeCreationRentSol:
          lamportsToSol(
            merkleTreeCreationRentLamports,
          ),

        merkleTreeCreationCostLamports:
          merkleTreeCreationCostLamports
            .toString(),
        merkleTreeCreationCostSol:
          lamportsToSol(
            merkleTreeCreationCostLamports,
          ),

        coreCollectionCreationTransactionFeeLamports:
          coreCollectionCreationTransactionFeeLamports
            .toString(),
        coreCollectionCreationTransactionFeeSol:
          lamportsToSol(
            coreCollectionCreationTransactionFeeLamports,
          ),

        coreCollectionCreationRentLamports:
          coreCollectionCreationRentLamports
            .toString(),
        coreCollectionCreationRentSol:
          lamportsToSol(
            coreCollectionCreationRentLamports,
          ),

        coreCollectionCreationCostLamports:
          coreCollectionCreationCostLamports
            .toString(),
        coreCollectionCreationCostSol:
          lamportsToSol(
            coreCollectionCreationCostLamports,
          ),

        provisioningCostLamports:
          provisioningCostLamports
            .toString(),
        provisioningCostSol:
          lamportsToSol(
            provisioningCostLamports,
          ),

        estimatedNetworkCostLamports:
          estimatedNetworkCostLamports
            .toString(),
        estimatedNetworkCostSol:
          lamportsToSol(
            estimatedNetworkCostLamports,
          ),

        requiredFeePayerBalanceLamports:
          requiredFeePayerBalanceLamports
            .toString(),
        requiredFeePayerBalanceSol:
          lamportsToSol(
            requiredFeePayerBalanceLamports,
          ),

        estimatedReserveTopUpLamports:
          estimatedReserveTopUpLamports
            .toString(),
        estimatedReserveTopUpSol:
          lamportsToSol(
            estimatedReserveTopUpLamports,
          ),

        reserveTransferFeeBufferLamports:
          reserveTransferFeeBufferLamports
            .toString(),
        reserveTransferFeeBufferSol:
          transactionFeeBufferSOL,

        requiredReserveForTopUpLamports:
          requiredReserveForTopUpLamports
            .toString(),
        requiredReserveForTopUpSol:
          lamportsToSol(
            requiredReserveForTopUpLamports,
          ),

        sufficient,
      },
    };
  }

  private validateConfig(): void {
    if (!this.config.cluster) {
      throw new Error(
        "mint_funding_estimate: cluster is required",
      );
    }

    if (!this.config.merkleTreeRegistryKey) {
      throw new Error(
        "mint_funding_estimate: merkleTreeRegistryKey is required",
      );
    }

    if (
      !Number.isInteger(
        this.config.merkleTreeMaxDepth,
      ) ||
      this.config.merkleTreeMaxDepth <= 0
    ) {
      throw new Error(
        "mint_funding_estimate: merkleTreeMaxDepth is invalid",
      );
    }

    if (
      !Number.isInteger(
        this.config.merkleTreeMaxBufferSize,
      ) ||
      this.config.merkleTreeMaxBufferSize <= 0
    ) {
      throw new Error(
        "mint_funding_estimate: merkleTreeMaxBufferSize is invalid",
      );
    }

    if (
      !Number.isInteger(
        this.config.merkleTreeCanopyDepth,
      ) ||
      this.config.merkleTreeCanopyDepth < 0 ||
      this.config.merkleTreeCanopyDepth >
        this.config.merkleTreeMaxDepth
    ) {
      throw new Error(
        "mint_funding_estimate: merkleTreeCanopyDepth is invalid",
      );
    }

    if (
      typeof this.config.merkleTreePublic !==
      "boolean"
    ) {
      throw new Error(
        "mint_funding_estimate: merkleTreePublic is invalid",
      );
    }

    if (
      !Number.isFinite(
        this.config.feePayerTargetSOL,
      ) ||
      this.config.feePayerTargetSOL <= 0
    ) {
      throw new Error(
        "mint_funding_estimate: feePayerTargetSOL is invalid",
      );
    }

    if (
      !Number.isFinite(
        this.config.reserveMinimumSOL,
      ) ||
      this.config.reserveMinimumSOL < 0
    ) {
      throw new Error(
        "mint_funding_estimate: reserveMinimumSOL is invalid",
      );
    }

    const transactionFeeBufferSOL =
      this.config.transactionFeeBufferSOL ??
      DEFAULT_TRANSACTION_FEE_BUFFER_SOL;

    if (
      !Number.isFinite(
        transactionFeeBufferSOL,
      ) ||
      transactionFeeBufferSOL < 0
    ) {
      throw new Error(
        "mint_funding_estimate: transactionFeeBufferSOL is invalid",
      );
    }
  }

  private validateInput(
    input: MintFundingEstimateInput,
  ): void {
    if (!input.tokenBlueprintId) {
      throw new Error(
        "mint_funding_estimate: tokenBlueprintId is required",
      );
    }

    if (
      !Number.isInteger(
        input.mintQuantity,
      ) ||
      input.mintQuantity <= 0
    ) {
      throw new Error(
        "mint_funding_estimate: mintQuantity must be a positive integer",
      );
    }

    if (!input.leafOwnerAddress) {
      throw new Error(
        "mint_funding_estimate: leafOwnerAddress is required",
      );
    }

    if (!input.name) {
      throw new Error(
        "mint_funding_estimate: name is required",
      );
    }

    if (
      typeof input.symbol !==
      "string"
    ) {
      throw new Error(
        "mint_funding_estimate: symbol must be string",
      );
    }

    if (!input.umi) {
      throw new Error(
        "mint_funding_estimate: umi is required",
      );
    }

    if (!input.feePayer) {
      throw new Error(
        "mint_funding_estimate: feePayer is required",
      );
    }

    if (!input.reserve) {
      throw new Error(
        "mint_funding_estimate: reserve is required",
      );
    }

    if (
      String(
        input.feePayer.publicKey,
      ) ===
      String(
        input.reserve.publicKey,
      )
    ) {
      throw new Error(
        "mint_funding_estimate: fee payer and reserve must be different",
      );
    }
  }

  private validateRegisteredMerkleTree(
    record: MerkleTreeRegistryRecord | null,
  ): void {
    if (record === null) {
      return;
    }

    if (
      record.cluster !==
      this.config.cluster
    ) {
      throw new Error(
        [
          "mint_funding_estimate: merkle tree cluster mismatch",
          `expected=${this.config.cluster}`,
          `actual=${record.cluster}`,
        ].join(" "),
      );
    }

    if (
      record.maxDepth !==
      this.config.merkleTreeMaxDepth
    ) {
      throw new Error(
        "mint_funding_estimate: merkle tree maxDepth mismatch",
      );
    }

    if (
      record.maxBufferSize !==
      this.config.merkleTreeMaxBufferSize
    ) {
      throw new Error(
        "mint_funding_estimate: merkle tree maxBufferSize mismatch",
      );
    }

    if (
      record.canopyDepth !==
      this.config.merkleTreeCanopyDepth
    ) {
      throw new Error(
        "mint_funding_estimate: merkle tree canopyDepth mismatch",
      );
    }

    if (
      record.public !==
      this.config.merkleTreePublic
    ) {
      throw new Error(
        "mint_funding_estimate: merkle tree public mismatch",
      );
    }
  }

  private validateRegisteredCoreCollection(
    tokenBlueprintId: string,
    record: CoreCollectionRegistryRecord | null,
  ): void {
    if (record === null) {
      return;
    }

    if (
      record.tokenBlueprintId !==
      tokenBlueprintId
    ) {
      throw new Error(
        "mint_funding_estimate: core collection tokenBlueprintId mismatch",
      );
    }

    if (
      record.cluster !==
      this.config.cluster
    ) {
      throw new Error(
        [
          "mint_funding_estimate: core collection cluster mismatch",
          `expected=${this.config.cluster}`,
          `actual=${record.cluster}`,
        ].join(" "),
      );
    }
  }
}