// services/solana-bubblegum/src/scripts/verify-merkle-tree-live.ts

import {
  fetchTreeConfigFromSeeds,
  Version,
} from "@metaplex-foundation/mpl-bubblegum";

import {
  publicKey,
} from "@metaplex-foundation/umi";

import {
  getBubblegumRuntime,
  merkleTreeResolver,
} from "../bootstrap/container.js";

import {
  env,
} from "../config/env.js";

import {
  FirestoreMerkleTreeRegistryRepository,
} from "../infrastructure/firestore/merkle-tree-registry-repository.js";


const REGISTRY_KEY =
  "devnet-default";

const MAX_DEPTH =
  14;

const MAX_BUFFER_SIZE =
  64;

const CANOPY_DEPTH =
  8;

const PUBLIC =
  false;

const EXPECTED_TOTAL_MINT_CAPACITY =
  16_384n;


async function main(): Promise<void> {
  if (
    env.solanaCluster !==
    "devnet"
  ) {
    throw new Error(
      [
        "verify_merkle_tree_live: devnet only",
        `cluster=${env.solanaCluster}`,
      ].join(
        " ",
      ),
    );
  }


  const runtime =
    await getBubblegumRuntime();


  const registry =
    new FirestoreMerkleTreeRegistryRepository();


  const before =
    await registry.getByKey(
      REGISTRY_KEY,
    );


  if (
    before !==
    null
  ) {
    throw new Error(
      [
        "verify_merkle_tree_live: registry already exists before test",
        `registryKey=${REGISTRY_KEY}`,
        `treeAddress=${before.treeAddress}`,
      ].join(
        " ",
      ),
    );
  }


  console.log(
    "Merkle Tree live verification:",
  );


  console.log(
    JSON.stringify(
      {
        phase:
          "before_create",

        registryKey:
          REGISTRY_KEY,

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

        expectedTotalMintCapacity:
          Number(
            EXPECTED_TOTAL_MINT_CAPACITY,
          ),

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
    await merkleTreeResolver
      .resolve({
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
        "verify_merkle_tree_live: first resolve did not create tree",
        `status=${first.status}`,
        `registryKey=${REGISTRY_KEY}`,
      ].join(
        " ",
      ),
    );
  }


  if (
    first.treeAddress.length ===
    0
  ) {
    throw new Error(
      "verify_merkle_tree_live: created treeAddress is empty",
    );
  }


  if (
    first.txSignature.length ===
    0
  ) {
    throw new Error(
      "verify_merkle_tree_live: created txSignature is empty",
    );
  }


  if (
    first.cluster !==
    env.solanaCluster
  ) {
    throw new Error(
      [
        "verify_merkle_tree_live: created cluster mismatch",
        `expected=${env.solanaCluster}`,
        `actual=${first.cluster}`,
      ].join(
        " ",
      ),
    );
  }


  if (
    first.maxDepth !==
    MAX_DEPTH
  ) {
    throw new Error(
      [
        "verify_merkle_tree_live: created maxDepth mismatch",
        `expected=${MAX_DEPTH}`,
        `actual=${first.maxDepth}`,
      ].join(
        " ",
      ),
    );
  }


  if (
    first.maxBufferSize !==
    MAX_BUFFER_SIZE
  ) {
    throw new Error(
      [
        "verify_merkle_tree_live: created maxBufferSize mismatch",
        `expected=${MAX_BUFFER_SIZE}`,
        `actual=${first.maxBufferSize}`,
      ].join(
        " ",
      ),
    );
  }


  if (
    first.canopyDepth !==
    CANOPY_DEPTH
  ) {
    throw new Error(
      [
        "verify_merkle_tree_live: created canopyDepth mismatch",
        `expected=${CANOPY_DEPTH}`,
        `actual=${first.canopyDepth}`,
      ].join(
        " ",
      ),
    );
  }


  if (
    first.public !==
    PUBLIC
  ) {
    throw new Error(
      [
        "verify_merkle_tree_live: created public mismatch",
        `expected=${PUBLIC}`,
        `actual=${first.public}`,
      ].join(
        " ",
      ),
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

        registryKey:
          REGISTRY_KEY,

        treeAddress:
          first.treeAddress,

        txSignature:
          first.txSignature,

        cluster:
          first.cluster,

        maxDepth:
          first.maxDepth,

        maxBufferSize:
          first.maxBufferSize,

        canopyDepth:
          first.canopyDepth,

        public:
          first.public,
      },
      null,
      2,
    ),
  );


  const registryAfterCreate =
    await registry.getByKey(
      REGISTRY_KEY,
    );


  if (
    registryAfterCreate ===
    null
  ) {
    throw new Error(
      [
        "verify_merkle_tree_live: registry record missing after create",
        `registryKey=${REGISTRY_KEY}`,
      ].join(
        " ",
      ),
    );
  }


  if (
    registryAfterCreate.treeAddress !==
    first.treeAddress
  ) {
    throw new Error(
      [
        "verify_merkle_tree_live: registry treeAddress mismatch",
        `resolver=${first.treeAddress}`,
        `registry=${registryAfterCreate.treeAddress}`,
      ].join(
        " ",
      ),
    );
  }


  if (
    registryAfterCreate.txSignature !==
    first.txSignature
  ) {
    throw new Error(
      [
        "verify_merkle_tree_live: registry txSignature mismatch",
        `resolver=${first.txSignature}`,
        `registry=${registryAfterCreate.txSignature}`,
      ].join(
        " ",
      ),
    );
  }


  if (
    registryAfterCreate.cluster !==
    env.solanaCluster
  ) {
    throw new Error(
      [
        "verify_merkle_tree_live: registry cluster mismatch",
        `expected=${env.solanaCluster}`,
        `actual=${registryAfterCreate.cluster}`,
      ].join(
        " ",
      ),
    );
  }


  if (
    registryAfterCreate.maxDepth !==
    MAX_DEPTH
  ) {
    throw new Error(
      [
        "verify_merkle_tree_live: registry maxDepth mismatch",
        `expected=${MAX_DEPTH}`,
        `actual=${registryAfterCreate.maxDepth}`,
      ].join(
        " ",
      ),
    );
  }


  if (
    registryAfterCreate.maxBufferSize !==
    MAX_BUFFER_SIZE
  ) {
    throw new Error(
      [
        "verify_merkle_tree_live: registry maxBufferSize mismatch",
        `expected=${MAX_BUFFER_SIZE}`,
        `actual=${registryAfterCreate.maxBufferSize}`,
      ].join(
        " ",
      ),
    );
  }


  if (
    registryAfterCreate.canopyDepth !==
    CANOPY_DEPTH
  ) {
    throw new Error(
      [
        "verify_merkle_tree_live: registry canopyDepth mismatch",
        `expected=${CANOPY_DEPTH}`,
        `actual=${registryAfterCreate.canopyDepth}`,
      ].join(
        " ",
      ),
    );
  }


  if (
    registryAfterCreate.public !==
    PUBLIC
  ) {
    throw new Error(
      [
        "verify_merkle_tree_live: registry public mismatch",
        `expected=${PUBLIC}`,
        `actual=${registryAfterCreate.public}`,
      ].join(
        " ",
      ),
    );
  }


  const treeConfig =
    await fetchTreeConfigFromSeeds(
      runtime.umi,
      {
        merkleTree:
          publicKey(
            first.treeAddress,
          ),
      },
    );


  if (
    treeConfig.version !==
    Version.V2
  ) {
    throw new Error(
      [
        "verify_merkle_tree_live: tree version mismatch",
        `expected=${Version.V2}`,
        `actual=${treeConfig.version}`,
      ].join(
        " ",
      ),
    );
  }


  if (
    treeConfig.isPublic !==
    PUBLIC
  ) {
    throw new Error(
      [
        "verify_merkle_tree_live: on-chain public mismatch",
        `expected=${PUBLIC}`,
        `actual=${treeConfig.isPublic}`,
      ].join(
        " ",
      ),
    );
  }


  if (
    treeConfig.totalMintCapacity !==
    EXPECTED_TOTAL_MINT_CAPACITY
  ) {
    throw new Error(
      [
        "verify_merkle_tree_live: totalMintCapacity mismatch",
        `expected=${EXPECTED_TOTAL_MINT_CAPACITY}`,
        `actual=${treeConfig.totalMintCapacity}`,
      ].join(
        " ",
      ),
    );
  }


  if (
    String(
      treeConfig.treeCreator,
    ) !==
    String(
      runtime.mintAuthority.publicKey,
    )
  ) {
    throw new Error(
      [
        "verify_merkle_tree_live: treeCreator mismatch",
        `expected=${runtime.mintAuthority.publicKey}`,
        `actual=${treeConfig.treeCreator}`,
      ].join(
        " ",
      ),
    );
  }


  if (
    treeConfig.numMinted !==
    0n
  ) {
    throw new Error(
      [
        "verify_merkle_tree_live: newly created tree already has minted leaves",
        `numMinted=${treeConfig.numMinted}`,
      ].join(
        " ",
      ),
    );
  }


  console.log(
    "On-chain TreeConfig:",
  );


  console.log(
    JSON.stringify(
      {
        version:
          "V2",

        treeCreator:
          String(
            treeConfig.treeCreator,
          ),

        treeDelegate:
          String(
            treeConfig.treeDelegate,
          ),

        isPublic:
          treeConfig.isPublic,

        numMinted:
          Number(
            treeConfig.numMinted,
          ),

        totalMintCapacity:
          Number(
            treeConfig.totalMintCapacity,
          ),
      },
      null,
      2,
    ),
  );


  const second =
    await merkleTreeResolver
      .resolve({
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
        "verify_merkle_tree_live: second resolve did not reuse tree",
        `status=${second.status}`,
        `registryKey=${REGISTRY_KEY}`,
      ].join(
        " ",
      ),
    );
  }


  if (
    second.treeAddress !==
    first.treeAddress
  ) {
    throw new Error(
      [
        "verify_merkle_tree_live: treeAddress changed",
        `first=${first.treeAddress}`,
        `second=${second.treeAddress}`,
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
        "verify_merkle_tree_live: txSignature changed",
        `first=${first.txSignature}`,
        `second=${second.txSignature}`,
      ].join(
        " ",
      ),
    );
  }


  const registryAfterSecondResolve =
    await registry.getByKey(
      REGISTRY_KEY,
    );


  if (
    registryAfterSecondResolve ===
    null
  ) {
    throw new Error(
      "verify_merkle_tree_live: registry record disappeared",
    );
  }


  if (
    registryAfterSecondResolve.treeAddress !==
    first.treeAddress
  ) {
    throw new Error(
      [
        "verify_merkle_tree_live: final registry treeAddress mismatch",
        `expected=${first.treeAddress}`,
        `actual=${registryAfterSecondResolve.treeAddress}`,
      ].join(
        " ",
      ),
    );
  }


  if (
    registryAfterSecondResolve.txSignature !==
    first.txSignature
  ) {
    throw new Error(
      [
        "verify_merkle_tree_live: final registry txSignature mismatch",
        `expected=${first.txSignature}`,
        `actual=${registryAfterSecondResolve.txSignature}`,
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

        registryKey:
          REGISTRY_KEY,

        treeAddress:
          second.treeAddress,

        txSignature:
          second.txSignature,

        cluster:
          second.cluster,

        maxDepth:
          second.maxDepth,

        maxBufferSize:
          second.maxBufferSize,

        canopyDepth:
          second.canopyDepth,

        public:
          second.public,

        sameTree:
          second.treeAddress ===
          first.treeAddress,

        sameTransaction:
          second.txSignature ===
          first.txSignature,

        registryStored:
          true,

        onChainVerified:
          true,
      },
      null,
      2,
    ),
  );


  console.log(
    "Merkle Tree live verification: OK",
  );
}


main().catch(
  (error: unknown) => {
    console.error(
      "Merkle Tree live verification: FAILED",
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