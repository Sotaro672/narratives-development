// services/solana-bubblegum/src/bootstrap/container.ts

import {
  CoreCollectionResolver,
} from "../application/core-collection-resolver.js";

import {
  FeePayerTopUpUsecase,
} from "../application/fee-payer-top-up.js";

import {
  MerkleTreeResolver,
} from "../application/merkle-tree-resolver.js";

import {
  MintFundingEstimateUsecase,
} from "../application/mint-funding-estimate.js";

import {
  MintV2Usecase,
} from "../application/mint-v2-usecase.js";

import {
  env,
} from "../config/env.js";

import {
  FirestoreCoreCollectionRegistryRepository,
} from "../infrastructure/firestore/core-collection-registry-repository.js";

import {
  FirestoreMerkleTreeRegistryRepository,
} from "../infrastructure/firestore/merkle-tree-registry-repository.js";

import {
  FirestoreMintOperationRegistryRepository,
} from "../infrastructure/firestore/mint-operation-registry-repository.js";

import {
  BubblegumMintV2TransactionClient,
} from "../infrastructure/solana/bubblegum-mint-v2-transaction-client.js";

import {
  createBubblegumRuntime,
  type BubblegumRuntime,
} from "../infrastructure/solana/bubblegum-runtime.js";

import {
  SolanaTransactionFeeEstimator,
} from "../infrastructure/solana/solana-transaction-fee-estimator.js";

const MERKLE_TREE_REGISTRY_KEY =
  "devnet-default";

const MERKLE_TREE_MAX_DEPTH =
  14;

const MERKLE_TREE_MAX_BUFFER_SIZE =
  64;

const MERKLE_TREE_CANOPY_DEPTH =
  8;

const MERKLE_TREE_PUBLIC =
  false;

const coreCollectionRegistry =
  new FirestoreCoreCollectionRegistryRepository();

const merkleTreeRegistry =
  new FirestoreMerkleTreeRegistryRepository();

export const mintOperationRegistry =
  new FirestoreMintOperationRegistryRepository();

const bubblegumRuntimePromise =
  createBubblegumRuntime({
    rpcURL:
      env.solanaRpcURL,
    googleCloudProject:
      env.googleCloudProject,
  });

export async function getBubblegumRuntime(): Promise<BubblegumRuntime> {
  return bubblegumRuntimePromise;
}

export const feePayerTopUpUsecase =
  new FeePayerTopUpUsecase({
    targetSOL:
      env.feePayerTargetSOL,
    reserveMinimumSOL:
      env.reserveMinimumSOL,
  });

export const coreCollectionResolver =
  new CoreCollectionResolver(
    coreCollectionRegistry,
    feePayerTopUpUsecase,
    {
      cluster:
        env.solanaCluster,
    },
  );

export const merkleTreeResolver =
  new MerkleTreeResolver(
    merkleTreeRegistry,
    feePayerTopUpUsecase,
    {
      registryKey:
        MERKLE_TREE_REGISTRY_KEY,
      cluster:
        env.solanaCluster,
      maxDepth:
        MERKLE_TREE_MAX_DEPTH,
      maxBufferSize:
        MERKLE_TREE_MAX_BUFFER_SIZE,
      canopyDepth:
        MERKLE_TREE_CANOPY_DEPTH,
      public:
        MERKLE_TREE_PUBLIC,
    },
  );

const solanaTransactionFeeEstimator =
  new SolanaTransactionFeeEstimator();

export const mintFundingEstimateUsecase =
  new MintFundingEstimateUsecase(
    merkleTreeRegistry,
    coreCollectionRegistry,
    solanaTransactionFeeEstimator,
    {
      cluster:
        env.solanaCluster,
      merkleTreeRegistryKey:
        MERKLE_TREE_REGISTRY_KEY,
      merkleTreeMaxDepth:
        MERKLE_TREE_MAX_DEPTH,
      merkleTreeMaxBufferSize:
        MERKLE_TREE_MAX_BUFFER_SIZE,
      merkleTreeCanopyDepth:
        MERKLE_TREE_CANOPY_DEPTH,
      merkleTreePublic:
        MERKLE_TREE_PUBLIC,
      feePayerTargetSOL:
        env.feePayerTargetSOL,
      reserveMinimumSOL:
        env.reserveMinimumSOL,
    },
  );

export function getMintFundingEstimateUsecase(): MintFundingEstimateUsecase {
  return mintFundingEstimateUsecase;
}

const mintV2UsecasePromise =
  bubblegumRuntimePromise.then(
    (runtime) => {
      const mintV2Transaction =
        new BubblegumMintV2TransactionClient(
          runtime.umi,
        );

      return new MintV2Usecase(
        mintOperationRegistry,
        mintV2Transaction,
        merkleTreeResolver,
        coreCollectionResolver,
        feePayerTopUpUsecase,
        {
          cluster:
            env.solanaCluster,
        },
      );
    },
  );

export async function getMintV2Usecase(): Promise<MintV2Usecase> {
  return mintV2UsecasePromise;
}