// services/solana-bubblegum/src/scripts/verify-mint-v2-live.ts

import {
  fetchTreeConfigFromSeeds,
} from "@metaplex-foundation/mpl-bubblegum";

import {
  fetchCollection,
} from "@metaplex-foundation/mpl-core";

import {
  publicKey,
  type Umi,
} from "@metaplex-foundation/umi";

import {
  base58,
} from "@metaplex-foundation/umi/serializers";

import {
  type MintV2UsecaseInput,
} from "../application/mint-v2-usecase.js";

import {
  type MintOperationRecord,
  type MintOperationResult,
} from "../application/ports/mint-operation-registry-port.js";

import {
  getBubblegumRuntime,
  getMintV2Usecase,
  mintOperationRegistry,
} from "../bootstrap/container.js";

import {
  env,
} from "../config/env.js";

import {
  type BubblegumRuntime,
} from "../infrastructure/solana/bubblegum-runtime.js";


const PRODUCT_ID =
  "verify-mint-v2-live-product-v1";

const TOKEN_BLUEPRINT_ID =
  "verify-mint-v2-live-token-blueprint-v1";

const BRAND_ID =
  "verify-mint-v2-live-brand-v1";

const COLLECTION_NAME =
  "AMOL Bubblegum V2 Live Verification";

const COLLECTION_METADATA_URI =
  "https://example.com/amol/bubblegum-v2-live-collection.json";

const ASSET_NAME =
  "AMOL Bubblegum V2 Live Verification Asset";

const ASSET_SYMBOL =
  "AMOLV";

const ASSET_METADATA_URI =
  "https://example.com/amol/bubblegum-v2-live-asset.json";

const EXPECTED_ASSET_STANDARD =
  "bubblegum-v2";

const EXPECTED_TREE_CAPACITY =
  16_384n;


function assertCondition(
  condition: unknown,
  message: string,
): asserts condition {
  if (!condition) {
    throw new Error(
      [
        "verify_mint_v2_live:",
        message,
      ].join(
        " ",
      ),
    );
  }
}


function createInput(
  runtime: BubblegumRuntime,
): MintV2UsecaseInput {
  return {
    productId:
      PRODUCT_ID,

    tokenBlueprintId:
      TOKEN_BLUEPRINT_ID,

    brandId:
      BRAND_ID,

    leafOwnerAddress:
      String(
        runtime.mintAuthority.publicKey,
      ),

    leafDelegateAddress:
      null,

    coreCollection: {
      name:
        COLLECTION_NAME,

      metadataUri:
        COLLECTION_METADATA_URI,
    },

    metadata: {
      name:
        ASSET_NAME,

      symbol:
        ASSET_SYMBOL,

      uri:
        ASSET_METADATA_URI,

      sellerFeeBasisPoints:
        0,

      primarySaleHappened:
        false,

      isMutable:
        false,

      creators:
        [],
    },

    umi:
      runtime.umi,

    feePayer:
      runtime.feePayer,

    reserve:
      runtime.reserve,
  };
}


function assertResultValid(
  result: MintOperationResult,
): void {
  assertCondition(
    result.signature.length >
      0,
    "signature is empty",
  );

  assertCondition(
    result.assetStandard ===
      EXPECTED_ASSET_STANDARD,
    [
      "assetStandard mismatch",
      `expected=${EXPECTED_ASSET_STANDARD}`,
      `actual=${result.assetStandard}`,
    ].join(
      " ",
    ),
  );

  assertCondition(
    result.cluster ===
      env.solanaCluster,
    [
      "cluster mismatch",
      `expected=${env.solanaCluster}`,
      `actual=${result.cluster}`,
    ].join(
      " ",
    ),
  );

  assertCondition(
    result.assetId.length >
      0,
    "assetId is empty",
  );

  assertCondition(
    result.treeAddress.length >
      0,
    "treeAddress is empty",
  );

  assertCondition(
    Number.isSafeInteger(
      result.leafIndex,
    ) &&
      result.leafIndex >=
        0,
    [
      "leafIndex is invalid",
      `leafIndex=${result.leafIndex}`,
    ].join(
      " ",
    ),
  );

  assertCondition(
    result.coreCollectionAddress.length >
      0,
    "coreCollectionAddress is empty",
  );

  assertCondition(
    Number.isSafeInteger(
      result.slot,
    ) &&
      result.slot >
        0,
    [
      "slot is invalid",
      `slot=${result.slot}`,
    ].join(
      " ",
    ),
  );
}


function assertResultsEqual(
  first: MintOperationResult,
  second: MintOperationResult,
): void {
  assertCondition(
    second.signature ===
      first.signature,
    [
      "signature changed between executions",
      `first=${first.signature}`,
      `second=${second.signature}`,
    ].join(
      " ",
    ),
  );

  assertCondition(
    second.assetStandard ===
      first.assetStandard,
    [
      "assetStandard changed between executions",
      `first=${first.assetStandard}`,
      `second=${second.assetStandard}`,
    ].join(
      " ",
    ),
  );

  assertCondition(
    second.cluster ===
      first.cluster,
    [
      "cluster changed between executions",
      `first=${first.cluster}`,
      `second=${second.cluster}`,
    ].join(
      " ",
    ),
  );

  assertCondition(
    second.assetId ===
      first.assetId,
    [
      "assetId changed between executions",
      `first=${first.assetId}`,
      `second=${second.assetId}`,
    ].join(
      " ",
    ),
  );

  assertCondition(
    second.treeAddress ===
      first.treeAddress,
    [
      "treeAddress changed between executions",
      `first=${first.treeAddress}`,
      `second=${second.treeAddress}`,
    ].join(
      " ",
    ),
  );

  assertCondition(
    second.leafIndex ===
      first.leafIndex,
    [
      "leafIndex changed between executions",
      `first=${first.leafIndex}`,
      `second=${second.leafIndex}`,
    ].join(
      " ",
    ),
  );

  assertCondition(
    second.coreCollectionAddress ===
      first.coreCollectionAddress,
    [
      "coreCollectionAddress changed between executions",
      `first=${first.coreCollectionAddress}`,
      `second=${second.coreCollectionAddress}`,
    ].join(
      " ",
    ),
  );

  assertCondition(
    second.slot ===
      first.slot,
    [
      "slot changed between executions",
      `first=${first.slot}`,
      `second=${second.slot}`,
    ].join(
      " ",
    ),
  );
}


function assertRegistryRecord(
  record: MintOperationRecord | null,
  result: MintOperationResult,
): asserts record is MintOperationRecord {
  assertCondition(
    record !==
      null,
    "mint operation registry record does not exist",
  );

  assertCondition(
    record.productId ===
      PRODUCT_ID,
    [
      "registry productId mismatch",
      `expected=${PRODUCT_ID}`,
      `actual=${record.productId}`,
    ].join(
      " ",
    ),
  );

  assertCondition(
    record.status ===
      "CONFIRMED",
    [
      "registry operation is not CONFIRMED",
      `status=${record.status}`,
    ].join(
      " ",
    ),
  );

  assertCondition(
    record.signature !==
      null,
    "CONFIRMED registry record has no signature",
  );

  assertCondition(
    record.signature ===
      result.signature,
    [
      "registry signature mismatch",
      `expected=${result.signature}`,
      `actual=${record.signature}`,
    ].join(
      " ",
    ),
  );

  assertCondition(
    record.signedTransactionBase64 !==
      null,
    "CONFIRMED registry record has no signed transaction",
  );

  assertCondition(
    record.signedTransactionBase64.length >
      0,
    "CONFIRMED signed transaction is empty",
  );

  assertCondition(
    record.result !==
      null,
    "CONFIRMED registry record has no result",
  );

  assertResultsEqual(
    result,
    record.result,
  );
}


async function verifySignatureFinalized(
  umi: Umi,
  result: MintOperationResult,
): Promise<void> {
  const signature =
    base58.serialize(
      result.signature,
    );

  const statuses =
    await umi.rpc
      .getSignatureStatuses(
        [
          signature,
        ],
        {
          searchTransactionHistory:
            true,
        },
      );

  const status =
    statuses[0];

  assertCondition(
    status !==
      null &&
    status !==
      undefined,
    [
      "transaction signature not found on-chain",
      `signature=${result.signature}`,
    ].join(
      " ",
    ),
  );

  assertCondition(
    status.error ===
      null,
    [
      "transaction contains an on-chain error",
      `signature=${result.signature}`,
      `slot=${status.slot}`,
    ].join(
      " ",
    ),
  );

  assertCondition(
    status.commitment ===
      "finalized",
    [
      "transaction is not finalized",
      `signature=${result.signature}`,
      `commitment=${String(status.commitment)}`,
    ].join(
      " ",
    ),
  );

  assertCondition(
    status.slot ===
      result.slot,
    [
      "transaction slot mismatch",
      `registrySlot=${result.slot}`,
      `rpcSlot=${status.slot}`,
    ].join(
      " ",
    ),
  );
}


async function verifyMerkleTree(
  runtime: BubblegumRuntime,
  result: MintOperationResult,
): Promise<void> {
  const treeConfig =
    await fetchTreeConfigFromSeeds(
      runtime.umi,
      {
        merkleTree:
          publicKey(
            result.treeAddress,
          ),
      },
    );

  assertCondition(
    treeConfig.isPublic ===
      false,
    [
      "Merkle Tree must be private",
      `isPublic=${treeConfig.isPublic}`,
    ].join(
      " ",
    ),
  );

  assertCondition(
    String(
      treeConfig.treeCreator,
    ) ===
      String(
        runtime.mintAuthority.publicKey,
      ),
    [
      "Merkle Tree creator mismatch",
      `expected=${runtime.mintAuthority.publicKey}`,
      `actual=${treeConfig.treeCreator}`,
    ].join(
      " ",
    ),
  );

  assertCondition(
    treeConfig.totalMintCapacity ===
      EXPECTED_TREE_CAPACITY,
    [
      "Merkle Tree capacity mismatch",
      `expected=${EXPECTED_TREE_CAPACITY}`,
      `actual=${treeConfig.totalMintCapacity}`,
    ].join(
      " ",
    ),
  );

  assertCondition(
    treeConfig.numMinted >
      BigInt(
        result.leafIndex,
      ),
    [
      "Merkle Tree minted count does not include returned leafIndex",
      `numMinted=${treeConfig.numMinted}`,
      `leafIndex=${result.leafIndex}`,
    ].join(
      " ",
    ),
  );

  console.log(
    "Merkle Tree:",
  );

  console.log(
    JSON.stringify(
      {
        treeAddress:
          result.treeAddress,

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

        totalMintCapacity:
          Number(
            treeConfig.totalMintCapacity,
          ),

        numMinted:
          Number(
            treeConfig.numMinted,
          ),
      },
      null,
      2,
    ),
  );
}


async function verifyCoreCollection(
  runtime: BubblegumRuntime,
  result: MintOperationResult,
): Promise<void> {
  await fetchCollection(
    runtime.umi,
    publicKey(
      result.coreCollectionAddress,
    ),
  );

  console.log(
    "Core Collection:",
  );

  console.log(
    JSON.stringify(
      {
        address:
          result.coreCollectionAddress,

        onChainVerified:
          true,
      },
      null,
      2,
    ),
  );
}


async function main(): Promise<void> {
  if (
    env.solanaCluster !==
    "devnet"
  ) {
    throw new Error(
      [
        "verify_mint_v2_live: devnet only",
        `cluster=${env.solanaCluster}`,
      ].join(
        " ",
      ),
    );
  }


  const runtime =
    await getBubblegumRuntime();

  const usecase =
    await getMintV2Usecase();

  const input =
    createInput(
      runtime,
    );


  const before =
    await mintOperationRegistry
      .getByProductId(
        PRODUCT_ID,
      );


  console.log(
    "MintV2 live verification:",
  );

  console.log(
    JSON.stringify(
      {
        cluster:
          env.solanaCluster,

        productId:
          PRODUCT_ID,

        tokenBlueprintId:
          TOKEN_BLUEPRINT_ID,

        brandId:
          BRAND_ID,

        leafOwnerAddress:
          input.leafOwnerAddress,

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

        previousOperation:
          before ===
            null
            ? null
            : {
                status:
                  before.status,

                signature:
                  before.signature,

                hasSignedTransaction:
                  before.signedTransactionBase64 !==
                  null,

                hasResult:
                  before.result !==
                  null,
              },

        realDevnetMint:
          true,

        cleanupOperation:
          false,
      },
      null,
      2,
    ),
  );


  const first =
    await usecase.execute(
      input,
    );


  assertResultValid(
    first,
  );


  const afterFirst =
    await mintOperationRegistry
      .getByProductId(
        PRODUCT_ID,
      );


  assertRegistryRecord(
    afterFirst,
    first,
  );


  await verifySignatureFinalized(
    runtime.umi,
    first,
  );


  await verifyMerkleTree(
    runtime,
    first,
  );


  await verifyCoreCollection(
    runtime,
    first,
  );


  console.log(
    "First execution:",
  );

  console.log(
    JSON.stringify(
      {
        signature:
          first.signature,

        assetStandard:
          first.assetStandard,

        cluster:
          first.cluster,

        assetId:
          first.assetId,

        treeAddress:
          first.treeAddress,

        leafIndex:
          first.leafIndex,

        coreCollectionAddress:
          first.coreCollectionAddress,

        slot:
          first.slot,

        registryStatus:
          afterFirst.status,

        signedTransactionStored:
          afterFirst.signedTransactionBase64 !==
          null,
      },
      null,
      2,
    ),
  );


  const second =
    await usecase.execute(
      input,
    );


  assertResultValid(
    second,
  );


  assertResultsEqual(
    first,
    second,
  );


  const afterSecond =
    await mintOperationRegistry
      .getByProductId(
        PRODUCT_ID,
      );


  assertRegistryRecord(
    afterSecond,
    second,
  );


  await verifySignatureFinalized(
    runtime.umi,
    second,
  );


  console.log(
    "Second execution:",
  );

  console.log(
    JSON.stringify(
      {
        signature:
          second.signature,

        assetId:
          second.assetId,

        treeAddress:
          second.treeAddress,

        leafIndex:
          second.leafIndex,

        coreCollectionAddress:
          second.coreCollectionAddress,

        slot:
          second.slot,

        sameSignature:
          second.signature ===
          first.signature,

        sameAssetId:
          second.assetId ===
          first.assetId,

        sameTree:
          second.treeAddress ===
          first.treeAddress,

        sameLeafIndex:
          second.leafIndex ===
          first.leafIndex,

        sameCoreCollection:
          second.coreCollectionAddress ===
          first.coreCollectionAddress,

        registryStatus:
          afterSecond.status,

        idempotencyVerified:
          true,
      },
      null,
      2,
    ),
  );


  console.log(
    JSON.stringify(
      {
        status:
          "OK",

        realDevnetMint:
          true,

        productId:
          PRODUCT_ID,

        signature:
          first.signature,

        assetId:
          first.assetId,

        treeAddress:
          first.treeAddress,

        leafIndex:
          first.leafIndex,

        coreCollectionAddress:
          first.coreCollectionAddress,

        slot:
          first.slot,

        transactionFinalized:
          true,

        merkleTreeVerified:
          true,

        coreCollectionVerified:
          true,

        registryConfirmed:
          true,

        signedTransactionStored:
          true,

        secondExecutionReturnedSameResult:
          true,

        operationRetained:
          true,
      },
      null,
      2,
    ),
  );


  console.log(
    "MintV2 live verification: OK",
  );


  console.log(
    [
      "MintV2 live verification:",
      "bubblegumMintOperations record was intentionally retained",
      `productId=${PRODUCT_ID}`,
    ].join(
      " ",
    ),
  );
}


main().catch(
  (error: unknown) => {
    console.error(
      "MintV2 live verification: FAILED",
    );

    if (
      error instanceof
      Error
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

    console.error(
      [
        "MintV2 live verification:",
        "do not delete the mint operation automatically",
        `productId=${PRODUCT_ID}`,
        "inspect its signature/status before any manual recovery",
      ].join(
        " ",
      ),
    );

    process.exitCode =
      1;
  },
);