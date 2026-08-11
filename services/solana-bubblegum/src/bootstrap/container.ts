// services/solana-bubblegum/src/bootstrap/container.ts

import {
  CoreCollectionResolver,
} from "../application/core-collection-resolver.js";

import {
  DevnetReserveRefillUsecase,
} from "../application/devnet-reserve-refill.js";

import {
  FeePayerTopUpUsecase,
} from "../application/fee-payer-top-up.js";

import {
  MerkleTreeResolver,
} from "../application/merkle-tree-resolver.js";

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
  FirestoreFaucetRateLimitRepository,
} from "../infrastructure/firestore/faucet-rate-limit-repository.js";

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
  SolanaRpcClient,
} from "../infrastructure/solana/solana-rpc-client.js";


const solanaRpc =
  new SolanaRpcClient(
    env.devnetAirdropRpcURL,
  );


const faucetRateLimit =
  new FirestoreFaucetRateLimitRepository();


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


export const devnetReserveRefillUsecase =
  new DevnetReserveRefillUsecase(
    solanaRpc,
    faucetRateLimit,
    {
      cluster:
        env.solanaCluster,

      reserveAddress:
        env.reservePublicKey,

      targetSOL:
        env.devnetReserveTargetSOL,

      requestSOL:
        env.devnetAirdropSOL,
    },
  );


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
        "devnet-default",

      cluster:
        env.solanaCluster,

      maxDepth:
        14,

      maxBufferSize:
        64,

      canopyDepth:
        8,

      public:
        false,
    },
  );


const mintV2UsecasePromise =
  bubblegumRuntimePromise
    .then(
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