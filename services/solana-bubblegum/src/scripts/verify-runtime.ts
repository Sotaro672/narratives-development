// services/solana-bubblegum/src/scripts/verify-runtime.ts

import {
  createBubblegumRuntime,
} from "../infrastructure/solana/bubblegum-runtime.js";


const expectedMintAuthority =
  "3RpTdVJ5hWeG5ErnARSS7Acwgwa5U4pDUDU1ngn29ayF";

const expectedFeePayer =
  "CnRBUrN2uZjHwxcTbcNJGJ23WXswQKXYAo4rR55YxioW";

const expectedReserve =
  "9VwuvEExAcBgfy8oupQEUrXTWd1NSdNmiQK7QSKED6Rs";


function requiredEnv(
  key: string,
): string {
  const value =
    process.env[key];

  if (!value) {
    throw new Error(
      `verify_runtime: ${key} is required`,
    );
  }

  return value;
}


function assertPublicKey(
  name: string,
  actual: string,
  expected: string,
): void {
  if (
    actual !==
    expected
  ) {
    throw new Error(
      [
        "verify_runtime: public key mismatch",
        `name=${name}`,
        `expected=${expected}`,
        `actual=${actual}`,
      ].join(
        " ",
      ),
    );
  }


  console.log(
    [
      "Runtime signer: OK",
      `name=${name}`,
      `publicKey=${actual}`,
    ].join(
      " ",
    ),
  );
}


async function main(): Promise<void> {
  const runtime =
    await createBubblegumRuntime({
      rpcURL:
        requiredEnv(
          "SOLANA_RPC_URL",
        ),

      googleCloudProject:
        requiredEnv(
          "GOOGLE_CLOUD_PROJECT",
        ),
    });


  assertPublicKey(
    "umi.identity",
    String(
      runtime.umi.identity.publicKey,
    ),
    expectedMintAuthority,
  );


  assertPublicKey(
    "umi.payer",
    String(
      runtime.umi.payer.publicKey,
    ),
    expectedFeePayer,
  );


  assertPublicKey(
    "mintAuthority",
    String(
      runtime.mintAuthority.publicKey,
    ),
    expectedMintAuthority,
  );


  assertPublicKey(
    "feePayer",
    String(
      runtime.feePayer.publicKey,
    ),
    expectedFeePayer,
  );


  assertPublicKey(
    "reserve",
    String(
      runtime.reserve.publicKey,
    ),
    expectedReserve,
  );


  console.log(
    "Bubblegum runtime: OK",
  );
}


main().catch(
  (error: unknown) => {
    console.error(
      "Bubblegum runtime verification: FAILED",
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