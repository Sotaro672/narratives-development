// services/solana-bubblegum/src/infrastructure/solana/bubblegum-mint-v2-transaction-client.ts

import {
  Buffer,
} from "node:buffer";

import {
  mintV2,
  parseLeafFromMintV2Transaction,
} from "@metaplex-foundation/mpl-bubblegum";

import {
  publicKey,
  type PublicKey,
  type Umi,
} from "@metaplex-foundation/umi";

import {
  base58,
} from "@metaplex-foundation/umi/serializers";

import {
  MintV2TransactionError,
  isMintV2TransactionError,
  type BroadcastMintV2TransactionInput,
  type BroadcastMintV2TransactionResult,
  type BuildAndSignMintV2TransactionInput,
  type BuildAndSignMintV2TransactionResult,
  type MintV2Creator,
  type MintV2TransactionPort,
  type ParseMintV2ResultInput,
  type ParseMintV2ResultResult,
  type WaitForMintV2FinalizedInput,
  type WaitForMintV2FinalizedResult,
} from "../../application/ports/mint-v2-transaction-port.js";


const FINALIZATION_TIMEOUT_MS =
  90_000;

const FINALIZATION_POLL_INTERVAL_MS =
  2_000;

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


function retryableError(
  code: string,
  message: string,
  cause?: unknown,
): MintV2TransactionError {
  return new MintV2TransactionError(
    "RETRYABLE",
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


function parseTransactionSignature(
  signature: string,
): Uint8Array {
  requiredString(
    "signature",
    signature,
  );

  try {
    const bytes =
      base58.serialize(
        signature,
      );

    if (
      bytes.length ===
      0
    ) {
      throw new Error(
        "empty signature",
      );
    }

    return bytes;
  } catch (error) {
    throw fatalError(
      "INVALID_SIGNATURE",
      [
        "bubblegum_mint_v2_transaction: invalid transaction signature",
        `signature=${signature}`,
      ].join(
        " ",
      ),
      error,
    );
  }
}


function transactionSignatureToString(
  signature: Uint8Array,
): string {
  if (
    signature.length ===
    0 ||
    signature.every(
      (value) =>
        value ===
        0,
    )
  ) {
    throw fatalError(
      "MISSING_TRANSACTION_SIGNATURE",
      "bubblegum_mint_v2_transaction: signed transaction has no transaction signature",
    );
  }

  try {
    return base58.deserialize(
      signature,
    )[0];
  } catch (error) {
    throw fatalError(
      "INVALID_TRANSACTION_SIGNATURE",
      "bubblegum_mint_v2_transaction: failed to encode transaction signature",
      error,
    );
  }
}


function encodeSignedTransactionBase64(
  bytes: Uint8Array,
): string {
  if (
    bytes.length ===
    0
  ) {
    throw fatalError(
      "EMPTY_SIGNED_TRANSACTION",
      "bubblegum_mint_v2_transaction: serialized signed transaction is empty",
    );
  }

  return Buffer
    .from(
      bytes,
    )
    .toString(
      "base64",
    );
}


function decodeSignedTransactionBase64(
  value: string,
): Uint8Array {
  requiredString(
    "signedTransactionBase64",
    value,
  );

  if (
    value.length %
      4 !==
    0 ||
    !/^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/.test(
      value,
    )
  ) {
    throw fatalError(
      "INVALID_SIGNED_TRANSACTION_BASE64",
      "bubblegum_mint_v2_transaction: signed transaction is not valid base64",
    );
  }

  const buffer =
    Buffer.from(
      value,
      "base64",
    );

  if (
    buffer.length ===
    0
  ) {
    throw fatalError(
      "EMPTY_SIGNED_TRANSACTION",
      "bubblegum_mint_v2_transaction: decoded signed transaction is empty",
    );
  }

  if (
    buffer.toString(
      "base64",
    ) !==
    value
  ) {
    throw fatalError(
      "INVALID_SIGNED_TRANSACTION_BASE64",
      "bubblegum_mint_v2_transaction: signed transaction base64 is not canonical",
    );
  }

  return new Uint8Array(
    buffer,
  );
}


function sleep(
  milliseconds: number,
): Promise<void> {
  return new Promise<void>(
    (resolve) => {
      setTimeout(
        resolve,
        milliseconds,
      );
    },
  );
}


export class BubblegumMintV2TransactionClient
  implements MintV2TransactionPort {
  constructor(
    private readonly umi:
      Umi,
  ) {}


  async buildAndSign(
    input: BuildAndSignMintV2TransactionInput,
  ): Promise<BuildAndSignMintV2TransactionResult> {
    try {
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
        this.buildMetadata(
          input,
          coreCollection,
        );

      const builder =
        mintV2(
          this.umi,
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
              this.umi.payer,

            treeCreatorOrDelegate:
              this.umi.identity,

            ...(
              coreCollection ===
                null
                ? {}
                : {
                    coreCollection,

                    collectionAuthority:
                      this.umi.identity,
                  }
            ),

            metadata,
          },
        );

      const signedTransaction =
        await builder
          .buildAndSign(
            this.umi,
          );

      const firstSignature =
        signedTransaction
          .signatures[0];

      if (
        firstSignature ===
        undefined
      ) {
        throw fatalError(
          "MISSING_TRANSACTION_SIGNATURE",
          "bubblegum_mint_v2_transaction: signed transaction contains no signatures",
        );
      }

      const signature =
        transactionSignatureToString(
          firstSignature,
        );

      const serializedTransaction =
        this.umi.transactions
          .serialize(
            signedTransaction,
          );

      const signedTransactionBase64 =
        encodeSignedTransactionBase64(
          serializedTransaction,
        );

      return {
        signature,

        signedTransactionBase64,
      };
    } catch (error) {
      if (
        isMintV2TransactionError(
          error,
        )
      ) {
        throw error;
      }

      throw retryableError(
        "BUILD_AND_SIGN_FAILED",
        "bubblegum_mint_v2_transaction: failed to build and sign mintV2 transaction",
        error,
      );
    }
  }


  async broadcast(
    input: BroadcastMintV2TransactionInput,
  ): Promise<BroadcastMintV2TransactionResult> {
    const expectedSignature =
      requiredString(
        "signature",
        input.signature,
      );

    const serializedTransaction =
      decodeSignedTransactionBase64(
        input.signedTransactionBase64,
      );

    let signedTransaction;

    try {
      signedTransaction =
        this.umi.transactions
          .deserialize(
            serializedTransaction,
          );
    } catch (error) {
      throw fatalError(
        "SIGNED_TRANSACTION_DESERIALIZE_FAILED",
        "bubblegum_mint_v2_transaction: failed to deserialize signed transaction",
        error,
      );
    }

    const storedSignature =
      signedTransaction
        .signatures[0];

    if (
      storedSignature ===
      undefined
    ) {
      throw fatalError(
        "MISSING_TRANSACTION_SIGNATURE",
        "bubblegum_mint_v2_transaction: stored signed transaction contains no signatures",
      );
    }

    const serializedSignature =
      transactionSignatureToString(
        storedSignature,
      );

    if (
      serializedSignature !==
      expectedSignature
    ) {
      throw fatalError(
        "SIGNED_TRANSACTION_SIGNATURE_MISMATCH",
        [
          "bubblegum_mint_v2_transaction: stored transaction signature mismatch",
          `expected=${expectedSignature}`,
          `actual=${serializedSignature}`,
        ].join(
          " ",
        ),
      );
    }

    let returnedSignatureBytes:
      Uint8Array;

    try {
      returnedSignatureBytes =
        await this.umi.rpc
          .sendTransaction(
            signedTransaction,
            {
              skipPreflight:
                false,

              preflightCommitment:
                "confirmed",

              maxRetries:
                3,
            },
          );
    } catch (error) {
      throw retryableError(
        "BROADCAST_FAILED",
        [
          "bubblegum_mint_v2_transaction: transaction broadcast failed",
          `signature=${expectedSignature}`,
        ].join(
          " ",
        ),
        error,
      );
    }

    const returnedSignature =
      transactionSignatureToString(
        returnedSignatureBytes,
      );

    if (
      returnedSignature !==
      expectedSignature
    ) {
      throw fatalError(
        "BROADCAST_SIGNATURE_MISMATCH",
        [
          "bubblegum_mint_v2_transaction: RPC returned unexpected signature",
          `expected=${expectedSignature}`,
          `actual=${returnedSignature}`,
        ].join(
          " ",
        ),
      );
    }

    return {
      signature:
        returnedSignature,
    };
  }


  async waitForFinalized(
    input: WaitForMintV2FinalizedInput,
  ): Promise<WaitForMintV2FinalizedResult> {
    const signatureString =
      requiredString(
        "signature",
        input.signature,
      );

    const signature =
      parseTransactionSignature(
        signatureString,
      );

    const deadline =
      Date.now() +
      FINALIZATION_TIMEOUT_MS;

    while (
      Date.now() <
      deadline
    ) {
      let statuses;

      try {
        statuses =
          await this.umi.rpc
            .getSignatureStatuses(
              [
                signature,
              ],
              {
                searchTransactionHistory:
                  true,
              },
            );
      } catch (error) {
        throw retryableError(
          "SIGNATURE_STATUS_FAILED",
          [
            "bubblegum_mint_v2_transaction: failed to fetch transaction status",
            `signature=${signatureString}`,
          ].join(
            " ",
          ),
          error,
        );
      }

      const status =
        statuses[0];

      if (
        status !==
          null &&
        status !==
          undefined
      ) {
        if (
          status.error !==
          null
        ) {
          throw fatalError(
            "TRANSACTION_FAILED",
            [
              "bubblegum_mint_v2_transaction: mintV2 transaction failed",
              `signature=${signatureString}`,
              `slot=${status.slot}`,
            ].join(
              " ",
            ),
          );
        }

        if (
          status.commitment ===
          "finalized"
        ) {
          return {
            slot:
              status.slot,
          };
        }
      }

      await sleep(
        FINALIZATION_POLL_INTERVAL_MS,
      );
    }

    throw retryableError(
      "FINALIZATION_TIMEOUT",
      [
        "bubblegum_mint_v2_transaction: transaction finalization timed out",
        `signature=${signatureString}`,
        `timeoutMs=${FINALIZATION_TIMEOUT_MS}`,
      ].join(
        " ",
      ),
    );
  }


  async parseMintResult(
    input: ParseMintV2ResultInput,
  ): Promise<ParseMintV2ResultResult> {
    const signatureString =
      requiredString(
        "signature",
        input.signature,
      );

    const signature =
      parseTransactionSignature(
        signatureString,
      );

    let leaf;

    try {
      leaf =
        await parseLeafFromMintV2Transaction(
          this.umi,
          signature,
        );
    } catch (error) {
      if (
        isMintV2TransactionError(
          error,
        )
      ) {
        throw error;
      }

      throw retryableError(
        "PARSE_MINT_RESULT_FAILED",
        [
          "bubblegum_mint_v2_transaction: failed to parse mintV2 leaf",
          `signature=${signatureString}`,
        ].join(
          " ",
        ),
        error,
      );
    }

    const assetId =
      String(
        leaf.id,
      );

    if (
      assetId.length ===
      0
    ) {
      throw fatalError(
        "EMPTY_ASSET_ID",
        [
          "bubblegum_mint_v2_transaction: parsed assetId is empty",
          `signature=${signatureString}`,
        ].join(
          " ",
        ),
      );
    }

    const leafIndex =
      Number(
        leaf.nonce,
      );

    if (
      !Number.isSafeInteger(
        leafIndex,
      ) ||
      leafIndex <
        0
    ) {
      throw fatalError(
        "INVALID_LEAF_INDEX",
        [
          "bubblegum_mint_v2_transaction: invalid parsed leaf index",
          `signature=${signatureString}`,
          `leafIndex=${String(leaf.nonce)}`,
        ].join(
          " ",
        ),
      );
    }

    return {
      assetId,

      leafIndex,
    };
  }


  private buildMetadata(
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
      this.buildCreators(
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


  private buildCreators(
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
                this.umi.identity.publicKey,
              ) &&
            address !==
              String(
                this.umi.payer.publicKey,
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
}