// services/solana-bubblegum/src/application/ports/faucet-rate-limit-port.ts

export type ReserveFaucetSlotResult =
  | {
      allowed: true;
      reservationId: string;
    }
  | {
      allowed: false;
      nextEligibleAt: Date;
    };


export type CompleteFaucetRequestInput = {
  reservationId: string;
  outcome:
    | "succeeded"
    | "failed"
    | "rate_limited";
  completedAt: Date;
  retryAfterSeconds?: number;
};


export interface FaucetRateLimitPort {
  reserveRequestSlot(
    now: Date,
  ): Promise<ReserveFaucetSlotResult>;


  completeRequestSlot(
    input: CompleteFaucetRequestInput,
  ): Promise<void>;
}