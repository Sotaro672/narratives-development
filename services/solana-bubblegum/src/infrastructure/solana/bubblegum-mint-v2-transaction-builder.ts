// services/solana-bubblegum/src/infrastructure/solana/bubblegum-mint-v2-transaction-builder.ts

import {
  mintV2,
} from "@metaplex-foundation/mpl-bubblegum";

import {
  publicKey,
  type PublicKey,
  type TransactionBuilder,
  type Umi,
} from "@metaplex-foundation/umi";

import {
  MintV2TransactionError,
  type BuildAndSignMintV2TransactionInput,
  type MintV2Creator,
} from "../../application/ports/mint-v2-transaction-port.js";


const SELLER_FEE_BASIS_POINTS_MIN =
  0;

const SELLER_FEE_BASIS_POINTS_MAX =
  10_000;

const CREATOR_SHARE_MIN =
  0;

const CREATOR_SHARE_MAX =
  100;

const CREATOR_SHARE_TOTAL =
  100;


function fatalError(
  code: string,
  message: string,
  cause?: unknown,
): MintV2TransactionError {
  return new MintV2TransactionError(
    "FATAL",
    code,
    message,
    cause === undefined
      ? undefined
      : {
          cause,
        },
  );
}


function requiredString(
  field: string,
  value: unknown,
): string {
  if (
    typeof value !==
      "string" ||
    value.length ===
      0
  ) {
    throw fatalError(
      "INVALID_INPUT",
      `bubblegum_mint_v2_transaction: ${field} is required`,
    );
  }

  return value;
}


function requiredBoolean(
  field: string,
  value: unknown,
): boolean {
  if (
    typeof value !==
    "boolean"
  ) {
    throw fatalError(
      "INVALID_INPUT",
      `bubblegum_mint_v2_transaction: ${field} must be boolean`,
    );
  }

  return value;
}


function requiredIntegerInRange(
  field: string,
  value: unknown,
  minimum: number,
  maximum: number,
): number {
  if (
    typeof value !==
      "number" ||
    !Number.isInteger(
      value,
    ) ||
    value <
      minimum ||
    value >
      maximum
  ) {
    throw fatalError(
      "INVALID_INPUT",
      [
        `bubblegum_mint_v2_transaction: invalid ${field}`,
        `minimum=${minimum}`,
        `maximum=${maximum}`,
      ].join(
        " ",
      ),
    );
  }

  return value;
}


function parsePublicKey(
  field: string,
  value: string,
): PublicKey {
  requiredString(
    field,
    value,
  );

  try {
    return publicKey(
      value,
    );
  } catch (error) {
    throw fatalError(
      "INVALID_PUBLIC_KEY",
      [
        "bubblegum_mint_v2_transaction: invalid public key",
        `field=${field}`,
        `value=${value}`,
      ].join(
        " ",
      ),
      error,
    );
  }
}


function buildCreators(
  umi: Umi,
  creators: MintV2Creator[],
) {
  if (
    !Array.isArray(
      creators,
    )
  ) {
    throw fatalError(
      "INVALID_CREATORS",
      "bubblegum_mint_v2_transaction: metadata.creators must be an array",
    );
  }

  const seenAddresses =
    new Set<string>();

  let totalShare =
    0;

  const result =
    creators.map(
      (
        creator,
        index,
      ) => {
        if (
          creator ===
            null ||
          typeof creator !==
            "object"
        ) {
          throw fatalError(
            "INVALID_CREATOR",
            [
              "bubblegum_mint_v2_transaction: invalid creator",
              `index=${index}`,
            ].join(
              " ",
            ),
          );
        }

        const address =
          requiredString(
            `metadata.creators[${index}].address`,
            creator.address,
          );

        const publicKeyValue =
          parsePublicKey(
            `metadata.creators[${index}].address`,
            address,
          );

        if (
          seenAddresses.has(
            address,
          )
        ) {
          throw fatalError(
            "DUPLICATE_CREATOR",
            [
              "bubblegum_mint_v2_transaction: duplicate creator address",
              `address=${address}`,
            ].join(
              " ",
            ),
          );
        }

        seenAddresses.add(
          address,
        );

        const verified =
          requiredBoolean(
            `metadata.creators[${index}].verified`,
            creator.verified,
          );

        if (
          verified &&
          address !==
            String(
              umi.identity.publicKey,
            ) &&
          address !==
            String(
              umi.payer.publicKey,
            )
        ) {
          throw fatalError(
            "UNAVAILABLE_VERIFIED_CREATOR_SIGNER",
            [
              "bubblegum_mint_v2_transaction: verified creator signer is unavailable",
              `address=${address}`,
            ].join(
              " ",
            ),
          );
        }

        const share =
          requiredIntegerInRange(
            `metadata.creators[${index}].share`,
            creator.share,
            CREATOR_SHARE_MIN,
            CREATOR_SHARE_MAX,
          );

        totalShare +=
          share;

        return {
          address:
            publicKeyValue,

          verified,

          share,
        };
      },
    );

  if (
    result.length >
      0 &&
    totalShare !==
      CREATOR_SHARE_TOTAL
  ) {
    throw fatalError(
      "INVALID_CREATOR_SHARE_TOTAL",
      [
        "bubblegum_mint_v2_transaction: creator share total must equal 100",
        `actual=${totalShare}`,
      ].join(
        " ",
      ),
    );
  }

  return result;
}


function buildMetadata(
  umi: Umi,
  input: BuildAndSignMintV2TransactionInput,
  coreCollection: PublicKey | null,
) {
  const name =
    requiredString(
      "metadata.name",
      input.metadata.name,
    );

  if (
    typeof input.metadata.symbol !==
    "string"
  ) {
    throw fatalError(
      "INVALID_INPUT",
      "bubblegum_mint_v2_transaction: metadata.symbol must be string",
    );
  }

  const uri =
    requiredString(
      "metadata.uri",
      input.metadata.uri,
    );

  const sellerFeeBasisPoints =
    requiredIntegerInRange(
      "metadata.sellerFeeBasisPoints",
      input.metadata.sellerFeeBasisPoints,
      SELLER_FEE_BASIS_POINTS_MIN,
      SELLER_FEE_BASIS_POINTS_MAX,
    );

  const primarySaleHappened =
    requiredBoolean(
      "metadata.primarySaleHappened",
      input.metadata.primarySaleHappened,
    );

  const isMutable =
    requiredBoolean(
      "metadata.isMutable",
      input.metadata.isMutable,
    );

  const creators =
    buildCreators(
      umi,
      input.metadata.creators,
    );

  return {
    name,

    symbol:
      input.metadata.symbol,

    uri,

    sellerFeeBasisPoints,

    primarySaleHappened,

    isMutable,

    collection:
      coreCollection,

    creators,
  };
}


/**
 * Bubblegum V2 Mint transactionの
 * TransactionBuilderを生成する。
 *
 * 実MintとSOL見積の双方でこの関数を使用し、
 * transaction構成の差異を発生させない。
 *
 * この関数自体はtransactionの
 * build / sign / sendを行わない。
 */
export function buildBubblegumMintV2TransactionBuilder(
  umi: Umi,
  input: BuildAndSignMintV2TransactionInput,
): TransactionBuilder {
  if (
    !umi
  ) {
    throw fatalError(
      "INVALID_INPUT",
      "bubblegum_mint_v2_transaction: umi is required",
    );
  }

  const merkleTree =
    parsePublicKey(
      "treeAddress",
      input.treeAddress,
    );

  const leafOwner =
    parsePublicKey(
      "leafOwnerAddress",
      input.leafOwnerAddress,
    );

  const leafDelegate =
    input.leafDelegateAddress ===
      null
      ? null
      : parsePublicKey(
          "leafDelegateAddress",
          input.leafDelegateAddress,
        );

  const coreCollection =
    input.coreCollectionAddress ===
      null
      ? null
      : parsePublicKey(
          "coreCollectionAddress",
          input.coreCollectionAddress,
        );

  const metadata =
    buildMetadata(
      umi,
      input,
      coreCollection,
    );

  return mintV2(
    umi,
    {
      merkleTree,

      leafOwner,

      ...(
        leafDelegate ===
          null
          ? {}
          : {
              leafDelegate,
            }
      ),

      payer:
        umi.payer,

      treeCreatorOrDelegate:
        umi.identity,

      ...(
        coreCollection ===
          null
          ? {}
          : {
              coreCollection,

              collectionAuthority:
                umi.identity,
            }
      ),

      metadata,
    },
  );
}