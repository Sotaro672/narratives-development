// services/solana-bubblegum/src/application/devnet-reserve-refill.ts

import type {
  SolanaRpcPort,
} from "./ports/solana-rpc-port.js";


import type {
  FaucetRateLimitPort,
} from "./ports/faucet-rate-limit-port.js";


const LAMPORTS_PER_SOL =
  1_000_000_000;


export type DevnetReserveRefillConfig = {
  cluster: string;
  reserveAddress: string;
  targetSOL: number;
  requestSOL: number;
};


export type DevnetReserveRefillResult = {
  status:
    | "refilled"
    | "balance_sufficient"
    | "rate_limited";


  reserveAddress: string;


  balanceBeforeSol: number;
  balanceAfterSol: number;


  requestedSol: number;


  signature?: string;


  nextEligibleAt?: string;


  retryAfterSeconds?: number;
};


type SolanaRateLimitErrorLike = Error & {
  retryAfterSeconds?: number;
};


function isSolanaRateLimitError(
  error: unknown,
): error is SolanaRateLimitErrorLike {
  if (!(error instanceof Error)) {
    return false;
  }


  if (
    error.name !==
    "SolanaRpcRateLimitError"
  ) {
    return false;
  }


  const candidate =
    error as Error & {
      retryAfterSeconds?: unknown;
    };


  return (
    candidate.retryAfterSeconds ===
      undefined ||
    (
      typeof candidate.retryAfterSeconds ===
        "number" &&
      Number.isFinite(
        candidate.retryAfterSeconds,
      ) &&
      candidate.retryAfterSeconds >= 0
    )
  );
}


export class DevnetReserveRefillUsecase {
  constructor(
    private readonly solanaRpc:
      SolanaRpcPort,


    private readonly rateLimit:
      FaucetRateLimitPort,


    private readonly config:
      DevnetReserveRefillConfig,
  ) {}


  async execute():
  Promise<DevnetReserveRefillResult> {
    if (
      this.config.cluster !==
      "devnet"
    ) {
      throw new Error(
        "devnet_reserve_refill: only devnet is allowed",
      );
    }


    if (
      !this.config.reserveAddress
    ) {
      throw new Error(
        "devnet_reserve_refill: reserveAddress is required",
      );
    }


    if (
      this.config.targetSOL <= 0
    ) {
      throw new Error(
        "devnet_reserve_refill: targetSOL must be greater than 0",
      );
    }


    if (
      this.config.requestSOL <= 0 ||
      this.config.requestSOL > 5
    ) {
      throw new Error(
        "devnet_reserve_refill: requestSOL must be greater than 0 and less than or equal to 5",
      );
    }


    const balanceBeforeLamports =
      await this.solanaRpc
        .getBalanceLamports(
          this.config.reserveAddress,
        );


    const balanceBeforeSol =
      balanceBeforeLamports /
      LAMPORTS_PER_SOL;


    if (
      balanceBeforeSol >=
      this.config.targetSOL
    ) {
      return {
        status:
          "balance_sufficient",


        reserveAddress:
          this.config.reserveAddress,


        balanceBeforeSol,
        balanceAfterSol:
          balanceBeforeSol,


        requestedSol: 0,
      };
    }


    const reservation =
      await this.rateLimit
        .reserveRequestSlot(
          new Date(),
        );


    if (!reservation.allowed) {
      return {
        status:
          "rate_limited",


        reserveAddress:
          this.config.reserveAddress,


        balanceBeforeSol,
        balanceAfterSol:
          balanceBeforeSol,


        requestedSol: 0,


        nextEligibleAt:
          reservation.nextEligibleAt
            .toISOString(),
      };
    }


    const reservationId =
      reservation.reservationId;


    const remainingSOL =
      Math.max(
        this.config.targetSOL -
          balanceBeforeSol,
        0,
      );


    const actualRequestSOL =
      Math.min(
        this.config.requestSOL,
        remainingSOL,
      );


    const lamports =
      Math.floor(
        actualRequestSOL *
          LAMPORTS_PER_SOL,
      );


    if (lamports <= 0) {
      await this.rateLimit
        .completeRequestSlot({
          reservationId,
          outcome:
            "rate_limited",
          completedAt:
            new Date(),
        });


      return {
        status:
          "balance_sufficient",


        reserveAddress:
          this.config.reserveAddress,


        balanceBeforeSol,
        balanceAfterSol:
          balanceBeforeSol,


        requestedSol: 0,
      };
    }


    let signature:
      string | undefined;


    try {
      signature =
        await this.solanaRpc
          .requestAirdrop(
            this.config.reserveAddress,
            lamports,
          );


      await this.solanaRpc
        .waitForConfirmation(
          signature,
        );
    } catch (error) {
      if (
        signature === undefined &&
        isSolanaRateLimitError(
          error,
        )
      ) {
        const retryAfterSeconds =
          error.retryAfterSeconds;


        await this.rateLimit
          .completeRequestSlot({
            reservationId,
            outcome:
              "rate_limited",
            completedAt:
              new Date(),
            retryAfterSeconds,
          });


        const nextEligibleAt =
          retryAfterSeconds ===
            undefined
            ? undefined
            : new Date(
                Date.now() +
                  retryAfterSeconds *
                    1000,
              ).toISOString();


        return {
          status:
            "rate_limited",


          reserveAddress:
            this.config.reserveAddress,


          balanceBeforeSol,
          balanceAfterSol:
            balanceBeforeSol,


          requestedSol:
            actualRequestSOL,


          nextEligibleAt,


          retryAfterSeconds,
        };
      }


      await this.rateLimit
        .completeRequestSlot({
          reservationId,
          outcome:
            "failed",
          completedAt:
            new Date(),
        });


      throw error;
    }


    await this.rateLimit
      .completeRequestSlot({
        reservationId,
        outcome:
          "succeeded",
        completedAt:
          new Date(),
      });


    const balanceAfterLamports =
      await this.solanaRpc
        .getBalanceLamports(
          this.config.reserveAddress,
        );


    return {
      status:
        "refilled",


      reserveAddress:
        this.config.reserveAddress,


      balanceBeforeSol,


      balanceAfterSol:
        balanceAfterLamports /
        LAMPORTS_PER_SOL,


      requestedSol:
        actualRequestSOL,


      signature,
    };
  }
}