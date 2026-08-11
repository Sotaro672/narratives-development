// services/solana-bubblegum/src/application/fee-payer-top-up.ts

import {
  transferSol,
} from "@metaplex-foundation/mpl-toolbox";

import {
  sol,
  type KeypairSigner,
  type Umi,
} from "@metaplex-foundation/umi";


const LAMPORTS_PER_SOL =
  1_000_000_000n;


const DEFAULT_TRANSACTION_FEE_BUFFER_SOL =
  0.001;


export type FeePayerTopUpConfig = {
  targetSOL: number;

  reserveMinimumSOL: number;

  transactionFeeBufferSOL?: number;
};


export type FeePayerTopUpInput = {
  umi: Umi;

  feePayer:
    KeypairSigner;

  reserve:
    KeypairSigner;
};


export type FeePayerTopUpResult = {
  status:
    | "topped_up"
    | "balance_sufficient"
    | "reserve_insufficient";

  feePayerAddress: string;

  reserveAddress: string;

  feePayerBalanceBeforeSOL: number;

  feePayerBalanceAfterSOL: number;

  reserveBalanceBeforeSOL: number;

  reserveBalanceAfterSOL: number;

  transferredSOL: number;

  signature?: Uint8Array;
};


function solToLamports(
  value: number,
): bigint {
  if (
    !Number.isFinite(
      value,
    ) ||
    value < 0
  ) {
    throw new Error(
      "fee_payer_top_up: invalid SOL amount",
    );
  }


  return BigInt(
    Math.floor(
      value *
        Number(
          LAMPORTS_PER_SOL,
        ),
    ),
  );
}


function lamportsToSOL(
  value: bigint,
): number {
  return (
    Number(
      value,
    ) /
    Number(
      LAMPORTS_PER_SOL,
    )
  );
}


export class FeePayerTopUpUsecase {
  constructor(
    private readonly config:
      FeePayerTopUpConfig,
  ) {}


  async execute(
    input: FeePayerTopUpInput,
  ): Promise<FeePayerTopUpResult> {
    if (
      this.config.targetSOL <= 0
    ) {
      throw new Error(
        "fee_payer_top_up: targetSOL must be greater than 0",
      );
    }


    if (
      this.config.reserveMinimumSOL < 0
    ) {
      throw new Error(
        "fee_payer_top_up: reserveMinimumSOL must not be negative",
      );
    }


    const transactionFeeBufferSOL =
      this.config
        .transactionFeeBufferSOL ??
      DEFAULT_TRANSACTION_FEE_BUFFER_SOL;


    if (
      !Number.isFinite(
        transactionFeeBufferSOL,
      ) ||
      transactionFeeBufferSOL < 0
    ) {
      throw new Error(
        "fee_payer_top_up: transactionFeeBufferSOL is invalid",
      );
    }


    const feePayerAddress =
      String(
        input.feePayer.publicKey,
      );


    const reserveAddress =
      String(
        input.reserve.publicKey,
      );


    if (
      feePayerAddress ===
      reserveAddress
    ) {
      throw new Error(
        "fee_payer_top_up: fee payer and reserve must be different",
      );
    }


    const [
      feePayerBalanceBefore,
      reserveBalanceBefore,
    ] =
      await Promise.all([
        input.umi.rpc.getBalance(
          input.feePayer.publicKey,
        ),

        input.umi.rpc.getBalance(
          input.reserve.publicKey,
        ),
      ]);


    const feePayerBalanceBeforeLamports =
      feePayerBalanceBefore
        .basisPoints;


    const reserveBalanceBeforeLamports =
      reserveBalanceBefore
        .basisPoints;


    const targetLamports =
      solToLamports(
        this.config.targetSOL,
      );


    const reserveMinimumLamports =
      solToLamports(
        this.config.reserveMinimumSOL,
      );


    const transactionFeeBufferLamports =
      solToLamports(
        transactionFeeBufferSOL,
      );


    if (
      feePayerBalanceBeforeLamports >=
      targetLamports
    ) {
      return {
        status:
          "balance_sufficient",

        feePayerAddress,

        reserveAddress,

        feePayerBalanceBeforeSOL:
          lamportsToSOL(
            feePayerBalanceBeforeLamports,
          ),

        feePayerBalanceAfterSOL:
          lamportsToSOL(
            feePayerBalanceBeforeLamports,
          ),

        reserveBalanceBeforeSOL:
          lamportsToSOL(
            reserveBalanceBeforeLamports,
          ),

        reserveBalanceAfterSOL:
          lamportsToSOL(
            reserveBalanceBeforeLamports,
          ),

        transferredSOL: 0,
      };
    }


    const requiredTransferLamports =
      targetLamports -
      feePayerBalanceBeforeLamports;


    const requiredReserveLamports =
      requiredTransferLamports +
      reserveMinimumLamports +
      transactionFeeBufferLamports;


    if (
      reserveBalanceBeforeLamports <
      requiredReserveLamports
    ) {
      return {
        status:
          "reserve_insufficient",

        feePayerAddress,

        reserveAddress,

        feePayerBalanceBeforeSOL:
          lamportsToSOL(
            feePayerBalanceBeforeLamports,
          ),

        feePayerBalanceAfterSOL:
          lamportsToSOL(
            feePayerBalanceBeforeLamports,
          ),

        reserveBalanceBeforeSOL:
          lamportsToSOL(
            reserveBalanceBeforeLamports,
          ),

        reserveBalanceAfterSOL:
          lamportsToSOL(
            reserveBalanceBeforeLamports,
          ),

        transferredSOL: 0,
      };
    }


    const transferSOL =
      lamportsToSOL(
        requiredTransferLamports,
      );


    const transactionResult =
      await transferSol(
        input.umi,
        {
          source:
            input.reserve,

          destination:
            input.feePayer
              .publicKey,

          amount:
            sol(
              transferSOL,
            ),
        },
      )
        .setFeePayer(
          input.reserve,
        )
        .sendAndConfirm(
          input.umi,
        );


    const [
      feePayerBalanceAfter,
      reserveBalanceAfter,
    ] =
      await Promise.all([
        input.umi.rpc.getBalance(
          input.feePayer.publicKey,
        ),

        input.umi.rpc.getBalance(
          input.reserve.publicKey,
        ),
      ]);


    return {
      status:
        "topped_up",

      feePayerAddress,

      reserveAddress,

      feePayerBalanceBeforeSOL:
        lamportsToSOL(
          feePayerBalanceBeforeLamports,
        ),

      feePayerBalanceAfterSOL:
        lamportsToSOL(
          feePayerBalanceAfter
            .basisPoints,
        ),

      reserveBalanceBeforeSOL:
        lamportsToSOL(
          reserveBalanceBeforeLamports,
        ),

      reserveBalanceAfterSOL:
        lamportsToSOL(
          reserveBalanceAfter
            .basisPoints,
        ),

      transferredSOL:
        transferSOL,

      signature:
        transactionResult.signature,
    };
  }
}