// services/solana-bubblegum/src/scripts/verify-core-collection-resolver.ts

import crypto from "node:crypto";

import {
  CoreCollectionResolver,
} from "../application/core-collection-resolver.js";

import {
  FeePayerTopUpUsecase,
} from "../application/fee-payer-top-up.js";

import {
  getBubblegumRuntime,
} from "../bootstrap/container.js";

import {
  env,
} from "../config/env.js";

import {
  FirestoreCoreCollectionRegistryRepository,
} from "../infrastructure/firestore/core-collection-registry-repository.js";


const SAFE_TEST_TARGET_SOL =
  1_000_000;

const SAFE_TEST_RESERVE_MINIMUM_SOL =
  1_000_000;


async function main(): Promise<void> {
  const runtime =
    await getBubblegumRuntime();


  const registry =
    new FirestoreCoreCollectionRegistryRepository();


  const safeTestFeePayerTopUp =
    new FeePayerTopUpUsecase({
      targetSOL:
        SAFE_TEST_TARGET_SOL,

      reserveMinimumSOL:
        SAFE_TEST_RESERVE_MINIMUM_SOL,
    });


  const resolver =
    new CoreCollectionResolver(
      registry,
      safeTestFeePayerTopUp,
      {
        cluster:
          env.solanaCluster,
      },
    );


  const tokenBlueprintId =
    `verify-core-collection-${crypto.randomUUID()}`;


  const before =
    await registry
      .getByTokenBlueprintId(
        tokenBlueprintId,
      );


  if (before !== null) {
    throw new Error(
      [
        "verify_core_collection_resolver: unexpected registry record before test",
        `tokenBlueprintId=${tokenBlueprintId}`,
      ].join(
        " ",
      ),
    );
  }


  let capturedError:
    unknown;


  try {
    await resolver.resolve({
      tokenBlueprintId,

      name:
        "AMOL Core Collection Resolver Verification",

      metadataUri:
        "https://example.com/amol/core-collection-verification.json",

      umi:
        runtime.umi,

      feePayer:
        runtime.feePayer,

      reserve:
        runtime.reserve,
    });
  } catch (error) {
    capturedError =
      error;
  }


  if (
    capturedError ===
    undefined
  ) {
    throw new Error(
      "verify_core_collection_resolver: expected funding failure but resolve succeeded",
    );
  }


  const errorMessage =
    capturedError instanceof Error
      ? capturedError.message
      : String(
          capturedError,
        );


  const expectedPrefix =
    "core_collection_resolver: fee payer funding unavailable";


  if (
    !errorMessage.startsWith(
      expectedPrefix,
    )
  ) {
    throw capturedError;
  }


  const after =
    await registry
      .getByTokenBlueprintId(
        tokenBlueprintId,
      );


  if (after !== null) {
    throw new Error(
      [
        "verify_core_collection_resolver: registry record was created unexpectedly",
        `tokenBlueprintId=${tokenBlueprintId}`,
        `collectionAddress=${after.collectionAddress}`,
      ].join(
        " ",
      ),
    );
  }


  console.log(
    "Core Collection resolver result:",
  );


  console.log(
    JSON.stringify(
      {
        status:
          "funding_blocked",

        tokenBlueprintId,

        cluster:
          env.solanaCluster,

        feePayerAddress:
          String(
            runtime.feePayer.publicKey,
          ),

        reserveAddress:
          String(
            runtime.reserve.publicKey,
          ),

        onChainCollectionCreated:
          false,

        registryRecordCreated:
          false,

        error:
          errorMessage,
      },
      null,
      2,
    ),
  );


  console.log(
    "Core Collection resolver: OK (funding guard blocked collection creation)",
  );
}


main().catch(
  (error: unknown) => {
    console.error(
      "Core Collection resolver verification: FAILED",
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