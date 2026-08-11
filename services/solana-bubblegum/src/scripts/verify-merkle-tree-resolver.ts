// services/solana-bubblegum/src/scripts/verify-merkle-tree-resolver.ts

import crypto from "node:crypto";

import {
  FeePayerTopUpUsecase,
} from "../application/fee-payer-top-up.js";

import {
  MerkleTreeResolver,
} from "../application/merkle-tree-resolver.js";

import {
  getBubblegumRuntime,
} from "../bootstrap/container.js";

import {
  env,
} from "../config/env.js";

import {
  FirestoreMerkleTreeRegistryRepository,
} from "../infrastructure/firestore/merkle-tree-registry-repository.js";


const SAFE_TEST_TARGET_SOL =
  1_000_000;

const SAFE_TEST_RESERVE_MINIMUM_SOL =
  1_000_000;

const MAX_DEPTH =
  14;

const MAX_BUFFER_SIZE =
  64;

const CANOPY_DEPTH =
  8;

const PUBLIC =
  false;


async function main(): Promise<void> {
  const runtime =
    await getBubblegumRuntime();

  const registry =
    new FirestoreMerkleTreeRegistryRepository();

  const safeTestFeePayerTopUp =
    new FeePayerTopUpUsecase({
      targetSOL:
        SAFE_TEST_TARGET_SOL,

      reserveMinimumSOL:
        SAFE_TEST_RESERVE_MINIMUM_SOL,
    });

  const registryKey =
    `verify-merkle-tree-${crypto.randomUUID()}`;

  const resolver =
    new MerkleTreeResolver(
      registry,
      safeTestFeePayerTopUp,
      {
        registryKey,

        cluster:
          env.solanaCluster,

        maxDepth:
          MAX_DEPTH,

        maxBufferSize:
          MAX_BUFFER_SIZE,

        canopyDepth:
          CANOPY_DEPTH,

        public:
          PUBLIC,
      },
    );

  const before =
    await registry.getByKey(
      registryKey,
    );

  if (
    before !==
    null
  ) {
    throw new Error(
      [
        "verify_merkle_tree_resolver: unexpected registry record before test",
        `registryKey=${registryKey}`,
        `treeAddress=${before.treeAddress}`,
      ].join(
        " ",
      ),
    );
  }

  let capturedError:
    unknown;

  try {
    await resolver.resolve({
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
      "verify_merkle_tree_resolver: expected funding failure but resolve succeeded",
    );
  }

  const errorMessage =
    capturedError instanceof Error
      ? capturedError.message
      : String(
          capturedError,
        );

  const expectedPrefix =
    "merkle_tree_resolver: fee payer funding unavailable";

  if (
    !errorMessage.startsWith(
      expectedPrefix,
    )
  ) {
    throw capturedError;
  }

  const after =
    await registry.getByKey(
      registryKey,
    );

  if (
    after !==
    null
  ) {
    throw new Error(
      [
        "verify_merkle_tree_resolver: registry record was created unexpectedly",
        `registryKey=${registryKey}`,
        `treeAddress=${after.treeAddress}`,
      ].join(
        " ",
      ),
    );
  }

  console.log(
    "Merkle Tree resolver result:",
  );

  console.log(
    JSON.stringify(
      {
        status:
          "funding_blocked",

        registryKey,

        cluster:
          env.solanaCluster,

        maxDepth:
          MAX_DEPTH,

        maxBufferSize:
          MAX_BUFFER_SIZE,

        canopyDepth:
          CANOPY_DEPTH,

        public:
          PUBLIC,

        feePayerAddress:
          String(
            runtime.feePayer.publicKey,
          ),

        reserveAddress:
          String(
            runtime.reserve.publicKey,
          ),

        mintAuthorityAddress:
          String(
            runtime.mintAuthority.publicKey,
          ),

        onChainTreeCreated:
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
    "Merkle Tree resolver: OK (funding guard blocked tree creation)",
  );
}


main().catch(
  (error: unknown) => {
    console.error(
      "Merkle Tree resolver verification: FAILED",
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