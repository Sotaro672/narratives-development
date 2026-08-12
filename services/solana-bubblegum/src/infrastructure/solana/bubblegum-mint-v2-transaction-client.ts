// services/solana-bubblegum/src/infrastructure/solana/bubblegum-mint-v2-transaction-client.ts

import {
  Buffer,
} from "node:buffer";

import {
  parseLeafFromMintV2Transaction,
} from "@metaplex-foundation/mpl-bubblegum";

import {
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
  type MintV2TransactionPort,
  type ParseMintV2ResultInput,
  type ParseMintV2ResultResult,
  type WaitForMintV2FinalizedInput,
  type WaitForMintV2FinalizedResult,
} from "../../application/ports/mint-v2-transaction-port.js";

import {
  buildBubblegumMintV2TransactionBuilder,
} from "./bubblegum-mint-v2-transaction-builder.js";


const FINALIZATION_TIMEOUT_MS =
  90_000;

const FINALIZATION_POLL_INTERVAL_MS =
  2_000;


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
      const builder =
        buildBubblegumMintV2TransactionBuilder(
          this.umi,
          input,
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
}