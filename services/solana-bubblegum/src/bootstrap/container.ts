// services/solana-bubblegum/src/bootstrap/container.ts

import {
  DevnetReserveRefillUsecase,
} from "../application/devnet-reserve-refill.js";

import {
  env,
} from "../config/env.js";

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