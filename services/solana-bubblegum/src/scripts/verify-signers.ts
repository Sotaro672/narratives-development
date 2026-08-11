// services/solana-bubblegum/src/scripts/verify-signers.ts

import {
  SecretManagerSignerLoader,
  type LogicalSigner,
} from "../infrastructure/secret-manager/signer-loader.js";

import {
  createSolanaUmi,
} from "../infrastructure/solana/umi-client.js";


type SignerExpectation = {
  logicalSigner: LogicalSigner;
  expectedPublicKey: string;
};


const signerExpectations: SignerExpectation[] = [
  {
    logicalSigner:
      "feePayer",

    expectedPublicKey:
      "CnRBUrN2uZjHwxcTbcNJGJ23WXswQKXYAo4rR55YxioW",
  },
  {
    logicalSigner:
      "reserve",

    expectedPublicKey:
      "9VwuvEExAcBgfy8oupQEUrXTWd1NSdNmiQK7QSKED6Rs",
  },
  {
    logicalSigner:
      "mintAuthority",

    expectedPublicKey:
      "3RpTdVJ5hWeG5ErnARSS7Acwgwa5U4pDUDU1ngn29ayF",
  },
];


function requiredEnv(
  key: string,
): string {
  const value =
    process.env[key];

  if (!value) {
    throw new Error(
      `verify_signers: ${key} is required`,
    );
  }

  return value;
}


async function main(): Promise<void> {
  const googleCloudProject =
    requiredEnv(
      "GOOGLE_CLOUD_PROJECT",
    );

  const solanaRpcURL =
    requiredEnv(
      "SOLANA_RPC_URL",
    );


  const umi =
    createSolanaUmi(
      solanaRpcURL,
    );


  const signerLoader =
    new SecretManagerSignerLoader(
      googleCloudProject,
    );


  for (
    const expectation
    of signerExpectations
  ) {
    const signer =
      await signerLoader.loadSigner(
        umi,
        expectation.logicalSigner,
      );


    const actualPublicKey =
      String(
        signer.publicKey,
      );


    if (
      actualPublicKey !==
      expectation.expectedPublicKey
    ) {
      throw new Error(
        [
          "verify_signers: public key mismatch",
          `logicalSigner=${expectation.logicalSigner}`,
          `expected=${expectation.expectedPublicKey}`,
          `actual=${actualPublicKey}`,
        ].join(
          " ",
        ),
      );
    }


    console.log(
      [
        "Signer: OK",
        `logicalSigner=${expectation.logicalSigner}`,
        `publicKey=${actualPublicKey}`,
      ].join(
        " ",
      ),
    );
  }


  console.log(
    "All Bubblegum signers: OK",
  );
}


main().catch(
  (error: unknown) => {
    console.error(
      "Signer verification: FAILED",
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