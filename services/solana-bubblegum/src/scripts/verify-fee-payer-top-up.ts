// services/solana-bubblegum/src/scripts/verify-fee-payer-top-up.ts

import {
  feePayerTopUpUsecase,
  getBubblegumRuntime,
} from "../bootstrap/container.js";


const expectedFeePayer =
  "CnRBUrN2uZjHwxcTbcNJGJ23WXswQKXYAo4rR55YxioW";

const expectedReserve =
  "9VwuvEExAcBgfy8oupQEUrXTWd1NSdNmiQK7QSKED6Rs";


async function main(): Promise<void> {
  const runtime =
    await getBubblegumRuntime();


  const result =
    await feePayerTopUpUsecase.execute({
      umi:
        runtime.umi,

      feePayer:
        runtime.feePayer,

      reserve:
        runtime.reserve,
    });


  if (
    result.feePayerAddress !==
    expectedFeePayer
  ) {
    throw new Error(
      [
        "verify_fee_payer_top_up: unexpected fee payer",
        `expected=${expectedFeePayer}`,
        `actual=${result.feePayerAddress}`,
      ].join(
        " ",
      ),
    );
  }


  if (
    result.reserveAddress !==
    expectedReserve
  ) {
    throw new Error(
      [
        "verify_fee_payer_top_up: unexpected reserve",
        `expected=${expectedReserve}`,
        `actual=${result.reserveAddress}`,
      ].join(
        " ",
      ),
    );
  }


  console.log(
    "Fee payer top-up result:",
  );


  console.log(
    JSON.stringify(
      {
        status:
          result.status,

        feePayerAddress:
          result.feePayerAddress,

        reserveAddress:
          result.reserveAddress,

        feePayerBalanceBeforeSOL:
          result.feePayerBalanceBeforeSOL,

        feePayerBalanceAfterSOL:
          result.feePayerBalanceAfterSOL,

        reserveBalanceBeforeSOL:
          result.reserveBalanceBeforeSOL,

        reserveBalanceAfterSOL:
          result.reserveBalanceAfterSOL,

        transferredSOL:
          result.transferredSOL,

        hasSignature:
          result.signature !==
          undefined,
      },
      null,
      2,
    ),
  );


  if (
    result.status ===
    "reserve_insufficient"
  ) {
    console.log(
      "Fee payer top-up: OK (reserve insufficient, no transfer executed)",
    );

    return;
  }


  if (
    result.status ===
    "balance_sufficient"
  ) {
    console.log(
      "Fee payer top-up: OK (fee payer balance already sufficient)",
    );

    return;
  }


  if (
    result.status ===
    "topped_up"
  ) {
    console.log(
      "Fee payer top-up: OK (SOL transferred)",
    );

    return;
  }


  throw new Error(
    "verify_fee_payer_top_up: unknown status",
  );
}


main().catch(
  (error: unknown) => {
    console.error(
      "Fee payer top-up verification: FAILED",
    );


    if (
      error instanceof Error
    ) {
      console.error(
        error.message,
      );
    } else {
      console.error(
        String(
          error,
        ),
      );
    }


    process.exitCode =
      1;
  },
);