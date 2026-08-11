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
  env,
} from "../config/env.js";

import {
  FirestoreCoreCollectionRegistryRepository,
} from "../infrastructure/firestore/core-collection-registry-repository.js";

import {
  FirestoreFaucetRateLimitRepository,
} from "../infrastructure/firestore/faucet-rate-limit-repository.js";

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