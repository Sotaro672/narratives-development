// services/solana-bubblegum/src/scripts/verify-core-collection-live.ts

import crypto from "node:crypto";

import {
  coreCollectionResolver,
  getBubblegumRuntime,
} from "../bootstrap/container.js";

import {
  env,
} from "../config/env.js";

import {
  FirestoreCoreCollectionRegistryRepository,
} from "../infrastructure/firestore/core-collection-registry-repository.js";


const COLLECTION_NAME =
  "AMOL Bubblegum V2 Core Collection Live Verification";

const COLLECTION_METADATA_URI =
  "https://example.com/amol/bubblegum-v2-core-collection-live.json";


async function main(): Promise<void> {
  if (
    env.solanaCluster !==
    "devnet"
  ) {
    throw new Error(
      [
        "verify_core_collection_live: devnet only",
        `cluster=${env.solanaCluster}`,
      ].join(
        " ",
      ),
    );
  }


  const runtime =
    await getBubblegumRuntime();


  const registry =
    new FirestoreCoreCollectionRegistryRepository();


  const tokenBlueprintId =
    `verify-core-collection-live-${crypto.randomUUID()}`;


  const before =
    await registry
      .getByTokenBlueprintId(
        tokenBlueprintId,
      );


  if (
    before !==
    null
  ) {
    throw new Error(
      [
        "verify_core_collection_live: unexpected registry record before test",
        `tokenBlueprintId=${tokenBlueprintId}`,
      ].join(
        " ",
      ),
    );
  }


  console.log(
    "Core Collection live verification:",
  );


  console.log(
    JSON.stringify(
      {
        phase:
          "before_create",

        cluster:
          env.solanaCluster,

        tokenBlueprintId,

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
      },
      null,
      2,
    ),
  );


  const first =
    await coreCollectionResolver
      .resolve({
        tokenBlueprintId,

        name:
          COLLECTION_NAME,

        metadataUri:
          COLLECTION_METADATA_URI,

        umi:
          runtime.umi,

        feePayer:
          runtime.feePayer,

        reserve:
          runtime.reserve,
      });


  if (
    first.status !==
    "created"
  ) {
    throw new Error(
      [
        "verify_core_collection_live: first resolve did not create collection",
        `status=${first.status}`,
        `tokenBlueprintId=${tokenBlueprintId}`,
      ].join(
        " ",
      ),
    );
  }


  if (
    first.collectionAddress.length ===
    0
  ) {
    throw new Error(
      "verify_core_collection_live: created collectionAddress is empty",
    );
  }


  if (
    first.txSignature.length ===
    0
  ) {
    throw new Error(
      "verify_core_collection_live: created txSignature is empty",
    );
  }


  console.log(
    "First resolve:",
  );


  console.log(
    JSON.stringify(
      {
        status:
          first.status,

        tokenBlueprintId:
          first.tokenBlueprintId,

        collectionAddress:
          first.collectionAddress,

        txSignature:
          first.txSignature,

        name:
          first.name,

        metadataUri:
          first.metadataUri,

        cluster:
          first.cluster,
      },
      null,
      2,
    ),
  );


  const registryAfterCreate =
    await registry
      .getByTokenBlueprintId(
        tokenBlueprintId,
      );


  if (
    registryAfterCreate ===
    null
  ) {
    throw new Error(
      [
        "verify_core_collection_live: registry record missing after create",
        `tokenBlueprintId=${tokenBlueprintId}`,
      ].join(
        " ",
      ),
    );
  }


  if (
    registryAfterCreate
      .collectionAddress !==
    first.collectionAddress
  ) {
    throw new Error(
      [
        "verify_core_collection_live: registry collectionAddress mismatch",
        `resolver=${first.collectionAddress}`,
        `registry=${registryAfterCreate.collectionAddress}`,
      ].join(
        " ",
      ),
    );
  }


  if (
    registryAfterCreate
      .txSignature !==
    first.txSignature
  ) {
    throw new Error(
      [
        "verify_core_collection_live: registry txSignature mismatch",
        `resolver=${first.txSignature}`,
        `registry=${registryAfterCreate.txSignature}`,
      ].join(
        " ",
      ),
    );
  }


  const second =
    await coreCollectionResolver
      .resolve({
        tokenBlueprintId,

        name:
          COLLECTION_NAME,

        metadataUri:
          COLLECTION_METADATA_URI,

        umi:
          runtime.umi,

        feePayer:
          runtime.feePayer,

        reserve:
          runtime.reserve,
      });


  if (
    second.status !==
    "existing"
  ) {
    throw new Error(
      [
        "verify_core_collection_live: second resolve did not reuse collection",
        `status=${second.status}`,
        `tokenBlueprintId=${tokenBlueprintId}`,
      ].join(
        " ",
      ),
    );
  }


  if (
    second.collectionAddress !==
    first.collectionAddress
  ) {
    throw new Error(
      [
        "verify_core_collection_live: collectionAddress changed",
        `first=${first.collectionAddress}`,
        `second=${second.collectionAddress}`,
      ].join(
        " ",
      ),
    );
  }


  if (
    second.txSignature !==
    first.txSignature
  ) {
    throw new Error(
      [
        "verify_core_collection_live: txSignature changed",
        `first=${first.txSignature}`,
        `second=${second.txSignature}`,
      ].join(
        " ",
      ),
    );
  }


  const registryAfterSecondResolve =
    await registry
      .getByTokenBlueprintId(
        tokenBlueprintId,
      );


  if (
    registryAfterSecondResolve ===
    null
  ) {
    throw new Error(
      "verify_core_collection_live: registry record disappeared",
    );
  }


  if (
    registryAfterSecondResolve
      .collectionAddress !==
    first.collectionAddress
  ) {
    throw new Error(
      [
        "verify_core_collection_live: final registry collectionAddress mismatch",
        `expected=${first.collectionAddress}`,
        `actual=${registryAfterSecondResolve.collectionAddress}`,
      ].join(
        " ",
      ),
    );
  }


  console.log(
    "Second resolve:",
  );


  console.log(
    JSON.stringify(
      {
        status:
          second.status,

        tokenBlueprintId:
          second.tokenBlueprintId,

        collectionAddress:
          second.collectionAddress,

        txSignature:
          second.txSignature,

        sameCollection:
          second.collectionAddress ===
          first.collectionAddress,

        registryStored:
          true,
      },
      null,
      2,
    ),
  );


  console.log(
    "Core Collection live verification: OK",
  );
}


main().catch(
  (error: unknown) => {
    console.error(
      "Core Collection live verification: FAILED",
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