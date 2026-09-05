// services/solana-bubblegum/src/app.ts

import { Buffer } from "node:buffer";
import { SecretManagerServiceClient } from "@google-cloud/secret-manager";
import { transferV2 } from "@metaplex-foundation/mpl-bubblegum";
import {
  createSignerFromKeypair,
  publicKey,
  type KeypairSigner,
  type PublicKey,
  type Umi,
} from "@metaplex-foundation/umi";
import { base58 } from "@metaplex-foundation/umi/serializers";
import express, {
  type NextFunction,
  type Request,
  type Response,
} from "express";

import {
  MintV2UsecaseInvalidStateError,
  MintV2UsecaseStoredFatalError,
  MintV2UsecaseValidationError,
} from "./application/mint-v2-usecase.js";
import {
  MintOperationNotFoundError,
  MintOperationPayloadConflictError,
  MintOperationSignedTransactionConflictError,
  MintOperationStateConflictError,
} from "./application/ports/mint-operation-registry-port.js";
import { isMintV2TransactionError } from "./application/ports/mint-v2-transaction-port.js";
import {
  getBubblegumRuntime,
  getMintFundingEstimateUsecase,
  getMintV2Usecase,
} from "./bootstrap/container.js";
import { env } from "./config/env.js";
import {
  HttpRequestValidationError,
  MintEstimateExecutionError,
  OwnedAssetsExecutionError,
  TransferExecutionError,
  TransferOwnershipConflictError,
  TransferSignerMismatchError,
} from "./http/errors.js";
import {
  optionalString,
  parseSolanaPublicKey,
  readMintEstimateRequestBody,
  readMintRequestBody,
  readOwnedAssetsRequestBody,
  readTransferRequestBody,
  requiredPositiveInteger,
  requiredString,
  stringValue,
} from "./http/request-validation.js";
import {
  fetchOwnedBubblegumAssetIDs,
  fetchTransferAssetWithProof,
} from "./infrastructure/das/das-client.js";
import type { DasTransferAsset } from "./infrastructure/das/das-types.js";

type TransferExecutionInput = {
  productId: string;
  assetId: string;
  fromAvatarId: string;
  fromBrandId: string;
  toAvatarId: string;
  fromWalletAddress: string;
  toWalletAddress: string;
};

type TransferExecutionResult = {
  signature: string;
  assetId: string;
};

const secretManagerClient = new SecretManagerServiceClient();

function parseSecretKey(
  secretID: string,
  raw: string,
): Uint8Array {
  let parsed: unknown;

  try {
    parsed = JSON.parse(raw);
  } catch (error) {
    throw new Error(
      [
        "transfer: invalid sender secret JSON",
        `secret=${secretID}`,
        `detail=${error instanceof Error ? error.message : String(error)}`,
      ].join(" "),
    );
  }

  if (!Array.isArray(parsed) || parsed.length !== 64) {
    throw new Error(
      [
        "transfer: invalid sender Solana keypair",
        `secret=${secretID}`,
        "expectedLength=64",
      ].join(" "),
    );
  }

  const bytes: number[] = [];

  for (const value of parsed) {
    if (
      typeof value !== "number" ||
      !Number.isInteger(value) ||
      value < 0 ||
      value > 255
    ) {
      throw new Error(
        [
          "transfer: invalid sender Solana keypair byte",
          `secret=${secretID}`,
        ].join(" "),
      );
    }

    bytes.push(value);
  }

  return Uint8Array.from(bytes);
}

async function loadSenderSigner(
  umi: Umi,
  input: {
    fromAvatarId: string;
    fromBrandId: string;
    fromWalletAddress: string;
  },
): Promise<KeypairSigner> {
  const hasAvatar = input.fromAvatarId.length > 0;
  const hasBrand = input.fromBrandId.length > 0;

  if (hasAvatar === hasBrand) {
    throw new HttpRequestValidationError(
      "sender",
      "exactly one of fromAvatarId or fromBrandId is required",
    );
  }

  const secretID =
    hasBrand
      ? `brand-wallet-${input.fromBrandId}`
      : `avatar-wallet-${input.fromAvatarId}`;

  const secretName =
    `projects/${env.googleCloudProject}/secrets/${secretID}/versions/latest`;

  let version;

  try {
    [version] = await secretManagerClient.accessSecretVersion({
      name: secretName,
    });
  } catch (error) {
    throw new Error(
      [
        "transfer: failed to load sender secret",
        `secret=${secretID}`,
        `detail=${error instanceof Error ? error.message : String(error)}`,
      ].join(" "),
    );
  }

  const data = version.payload?.data;

  if (!data) {
    throw new Error(
      [
        "transfer: sender secret payload is empty",
        `secret=${secretID}`,
      ].join(" "),
    );
  }

  const raw = Buffer.from(data).toString("utf8");
  const secretKey = parseSecretKey(secretID, raw);
  const keypair = umi.eddsa.createKeypairFromSecretKey(secretKey);
  const signer = createSignerFromKeypair(umi, keypair);
  const signerAddress = String(signer.publicKey);

  if (signerAddress !== input.fromWalletAddress) {
    throw new TransferSignerMismatchError(
      input.fromWalletAddress,
      signerAddress,
    );
  }

  return signer;
}

function parseCoreCollectionPublicKey(
  value: string,
): PublicKey {
  try {
    return publicKey(value);
  } catch (error) {
    throw new Error(
      [
        "DAS getAsset.grouping.collection must be a valid Solana public key",
        `value=${value}`,
        `detail=${error instanceof Error ? error.message : String(error)}`,
      ].join(" "),
    );
  }
}

function resolveCoreCollection(
  asset: DasTransferAsset,
): PublicKey | null {
  const collectionGroup =
    asset.grouping.find(
      (group) =>
        group.groupKey === "collection",
    );

  if (!collectionGroup) {
    return null;
  }

  return parseCoreCollectionPublicKey(
    collectionGroup.groupValue,
  );
}

async function executeBubblegumTransfer(
  input: TransferExecutionInput,
): Promise<TransferExecutionResult> {
  const runtime = await getBubblegumRuntime();

  parseSolanaPublicKey(
    "assetId",
    input.assetId,
  );

  const fromWalletPublicKey =
    parseSolanaPublicKey(
      "fromWalletAddress",
      input.fromWalletAddress,
    );

  const toWalletPublicKey =
    parseSolanaPublicKey(
      "toWalletAddress",
      input.toWalletAddress,
    );

  const canonicalFromWalletAddress =
    String(fromWalletPublicKey);

  const senderSigner =
    await loadSenderSigner(
      runtime.umi,
      {
        fromAvatarId: input.fromAvatarId,
        fromBrandId: input.fromBrandId,
        fromWalletAddress:
          canonicalFromWalletAddress,
      },
    );

  const assetWithProof =
    await fetchTransferAssetWithProof(
      runtime.umi,
      input.assetId,
    );

  const currentOwner =
    String(assetWithProof.leafOwner);

  if (
    currentOwner !==
    canonicalFromWalletAddress
  ) {
    throw new TransferOwnershipConflictError(
      canonicalFromWalletAddress,
      currentOwner,
    );
  }

  const coreCollection =
    resolveCoreCollection(
      assetWithProof.asset,
    );

  console.log(
    [
      "[transfer]",
      "start",
      `productId=${input.productId}`,
      `assetId=${input.assetId}`,
      `fromAvatarId=${input.fromAvatarId}`,
      `fromBrandId=${input.fromBrandId}`,
      `toAvatarId=${input.toAvatarId}`,
      `fromWallet=${canonicalFromWalletAddress}`,
      `toWallet=${String(toWalletPublicKey)}`,
    ].join(" "),
  );

  const transactionResult =
    await transferV2(
      runtime.umi,
      {
        payer: runtime.feePayer,
        authority: senderSigner,
        leafOwner:
          assetWithProof.leafOwner,
        leafDelegate:
          assetWithProof.leafDelegate,
        newLeafOwner:
          toWalletPublicKey,
        merkleTree:
          assetWithProof.merkleTree,
        root:
          assetWithProof.root,
        dataHash:
          assetWithProof.dataHash,
        creatorHash:
          assetWithProof.creatorHash,

        ...(
          assetWithProof.assetDataHash ===
          undefined
            ? {}
            : {
                assetDataHash:
                  assetWithProof.assetDataHash,
              }
        ),

        ...(
          assetWithProof.flags ===
          undefined
            ? {}
            : {
                flags:
                  assetWithProof.flags,
              }
        ),

        nonce:
          assetWithProof.nonce,
        index:
          assetWithProof.index,
        proof:
          assetWithProof.proof,

        ...(
          coreCollection === null
            ? {}
            : {
                coreCollection,
              }
        ),
      },
    ).sendAndConfirm(
      runtime.umi,
      {
        confirm: {
          commitment: "finalized",
        },
      },
    );

  const signature =
    base58.deserialize(
      transactionResult.signature,
    )[0];

  if (!signature) {
    throw new Error(
      "transfer: transaction signature is empty",
    );
  }

  console.log(
    [
      "[transfer]",
      "succeeded",
      `productId=${input.productId}`,
      `assetId=${input.assetId}`,
      `signature=${signature}`,
    ].join(" "),
  );

  return {
    signature,
    assetId: input.assetId,
  };
}

export const app = express();

app.disable("x-powered-by");

app.use(
  express.json({
    limit: "32kb",
  }),
);

app.get(
  "/health",
  (
    _req: Request,
    res: Response,
  ) => {
    res.status(200).json({
      status: "ok",
      service: "solana-bubblegum",
    });
  },
);

app.post(
  "/owned-assets",
  async (
    req: Request,
    res: Response,
    next: NextFunction,
  ) => {
    try {
      const body =
        readOwnedAssetsRequestBody(
          req.body,
        );

      const assetStandard =
        requiredString(
          "assetStandard",
          body.assetStandard,
        );

      const walletAddress =
        requiredString(
          "walletAddress",
          body.walletAddress,
        );

      if (
        assetStandard !==
        "BUBBLEGUM_V2"
      ) {
        throw new HttpRequestValidationError(
          "assetStandard",
          "only BUBBLEGUM_V2 is supported",
        );
      }

      parseSolanaPublicKey(
        "walletAddress",
        walletAddress,
      );

      let assetIds: string[];

      try {
        assetIds =
          await fetchOwnedBubblegumAssetIDs(
            walletAddress,
          );
      } catch (error) {
        throw new OwnedAssetsExecutionError(
          error,
        );
      }

      res.status(200).json({
        walletAddress,
        assetIds,
      });
    } catch (error) {
      next(error);
    }
  },
);

app.post(
  "/transfer",
  async (
    req: Request,
    res: Response,
    next: NextFunction,
  ) => {
    try {
      const body =
        readTransferRequestBody(
          req.body,
        );

      const productId =
        requiredString(
          "productId",
          body.productId,
        );

      const assetStandard =
        requiredString(
          "assetStandard",
          body.assetStandard,
        );

      if (
        assetStandard !==
        "BUBBLEGUM_V2"
      ) {
        throw new HttpRequestValidationError(
          "assetStandard",
          "only BUBBLEGUM_V2 is supported",
        );
      }

      const assetId =
        requiredString(
          "assetId",
          body.assetId,
        );

      const fromAvatarId =
        optionalString(
          "fromAvatarId",
          body.fromAvatarId,
        );

      const fromBrandId =
        optionalString(
          "fromBrandId",
          body.fromBrandId,
        );

      const toAvatarId =
        requiredString(
          "toAvatarId",
          body.toAvatarId,
        );

      const fromWalletAddress =
        requiredString(
          "fromWalletAddress",
          body.fromWalletAddress,
        );

      const toWalletAddress =
        requiredString(
          "toWalletAddress",
          body.toWalletAddress,
        );

      if (
        Boolean(fromAvatarId) ===
        Boolean(fromBrandId)
      ) {
        throw new HttpRequestValidationError(
          "sender",
          "exactly one of fromAvatarId or fromBrandId is required",
        );
      }

      parseSolanaPublicKey(
        "assetId",
        assetId,
      );

      parseSolanaPublicKey(
        "fromWalletAddress",
        fromWalletAddress,
      );

      parseSolanaPublicKey(
        "toWalletAddress",
        toWalletAddress,
      );

      let result:
        TransferExecutionResult;

      try {
        result =
          await executeBubblegumTransfer({
            productId,
            assetId,
            fromAvatarId,
            fromBrandId,
            toAvatarId,
            fromWalletAddress,
            toWalletAddress,
          });
      } catch (error) {
        if (
          error instanceof
            HttpRequestValidationError ||
          error instanceof
            TransferOwnershipConflictError ||
          error instanceof
            TransferSignerMismatchError
        ) {
          throw error;
        }

        throw new TransferExecutionError(
          error,
        );
      }

      res.status(200).json({
        signature:
          result.signature,
        assetId:
          result.assetId,
      });
    } catch (error) {
      next(error);
    }
  },
);

app.post(
  "/estimate",
  async (
    req: Request,
    res: Response,
    next: NextFunction,
  ) => {
    try {
      const body =
        readMintEstimateRequestBody(
          req.body,
        );

      const tokenBlueprintId =
        requiredString(
          "tokenBlueprintId",
          body.tokenBlueprintId,
        );

      const mintQuantity =
        requiredPositiveInteger(
          "mintQuantity",
          body.mintQuantity,
        );

      const toAddress =
        requiredString(
          "toAddress",
          body.toAddress,
        );

      const name =
        requiredString(
          "name",
          body.name,
        );

      const symbol =
        stringValue(
          "symbol",
          body.symbol,
        );

      let result;

      try {
        const runtime =
          await getBubblegumRuntime();

        const mintFundingEstimateUsecase =
          getMintFundingEstimateUsecase();

        result =
          await mintFundingEstimateUsecase.execute({
            tokenBlueprintId,
            mintQuantity,
            leafOwnerAddress:
              toAddress,
            name,
            symbol,
            umi:
              runtime.umi,
            feePayer:
              runtime.feePayer,
            reserve:
              runtime.reserve,
          });
      } catch (error) {
        if (
          isMintV2TransactionError(
            error,
          )
        ) {
          throw error;
        }

        throw new MintEstimateExecutionError(
          error,
        );
      }

      res.status(200).json(
        result,
      );
    } catch (error) {
      next(error);
    }
  },
);

app.post(
  "/mint",
  async (
    req: Request,
    res: Response,
    next: NextFunction,
  ) => {
    try {
      const body =
        readMintRequestBody(
          req.body,
        );

      const productId =
        requiredString(
          "productId",
          body.productId,
        );

      const idempotencyKey =
        req.get(
          "Idempotency-Key",
        );

      if (!idempotencyKey) {
        throw new HttpRequestValidationError(
          "Idempotency-Key",
          "header is required",
        );
      }

      if (
        idempotencyKey !==
        productId
      ) {
        throw new HttpRequestValidationError(
          "Idempotency-Key",
          "header must equal productId",
        );
      }

      const tokenBlueprintId =
        requiredString(
          "tokenBlueprintId",
          body.tokenBlueprintId,
        );

      const brandId =
        requiredString(
          "brandId",
          body.brandId,
        );

      const toAddress =
        requiredString(
          "toAddress",
          body.toAddress,
        );

      const name =
        requiredString(
          "name",
          body.name,
        );

      const symbol =
        stringValue(
          "symbol",
          body.symbol,
        );

      const metadataUri =
        requiredString(
          "metadataUri",
          body.metadataUri,
        );

      const [
        runtime,
        mintV2Usecase,
      ] =
        await Promise.all([
          getBubblegumRuntime(),
          getMintV2Usecase(),
        ]);

      const result =
        await mintV2Usecase.execute({
          productId,
          tokenBlueprintId,
          brandId,
          leafOwnerAddress:
            toAddress,
          leafDelegateAddress:
            null,
          coreCollection: {
            name,
            metadataUri,
          },
          metadata: {
            name,
            symbol,
            uri: metadataUri,
            sellerFeeBasisPoints: 0,
            primarySaleHappened:
              false,
            isMutable: false,
            creators: [],
          },
          umi:
            runtime.umi,
          feePayer:
            runtime.feePayer,
          reserve:
            runtime.reserve,
        });

      res.status(200).json(
        result,
      );
    } catch (error) {
      next(error);
    }
  },
);

app.use(
  (
    _req: Request,
    res: Response,
  ) => {
    res.status(404).json({
      error: "not found",
    });
  },
);

app.use(
  (
    error: unknown,
    _req: Request,
    res: Response,
    _next: NextFunction,
  ) => {
    console.error(
      "[http]",
      error,
    );

    if (
      error instanceof
      SyntaxError
    ) {
      res.status(400).json({
        error:
          "invalid JSON body",
        message:
          error.message,
      });
      return;
    }

    if (
      error instanceof
      HttpRequestValidationError
    ) {
      res.status(400).json({
        error:
          "invalid request",
        field:
          error.field,
        message:
          error.message,
      });
      return;
    }

    if (
      error instanceof
      TransferOwnershipConflictError
    ) {
      res.status(409).json({
        error:
          "transfer ownership conflict",
        message:
          error.message,
        expectedOwner:
          error.expectedOwner,
        actualOwner:
          error.actualOwner,
      });
      return;
    }

    if (
      error instanceof
      TransferSignerMismatchError
    ) {
      res.status(409).json({
        error:
          "transfer signer mismatch",
        message:
          error.message,
        expectedAddress:
          error.expectedAddress,
        signerAddress:
          error.signerAddress,
      });
      return;
    }

    if (
      error instanceof
      TransferExecutionError
    ) {
      res.status(503).json({
        error:
          "transfer unavailable",
        message:
          error.message,
      });
      return;
    }

    if (
      error instanceof
      OwnedAssetsExecutionError
    ) {
      res.status(503).json({
        error:
          "owned assets unavailable",
        message:
          error.message,
      });
      return;
    }

    if (
      error instanceof
      MintV2UsecaseValidationError
    ) {
      res.status(400).json({
        error:
          "invalid mint request",
        field:
          error.field,
        message:
          error.message,
      });
      return;
    }

    if (
      isMintV2TransactionError(
        error,
      )
    ) {
      if (
        error.code ===
          "INVALID_INPUT" ||
        error.code ===
          "INVALID_PUBLIC_KEY" ||
        error.code ===
          "INVALID_SIGNATURE" ||
        error.code ===
          "INVALID_TRANSACTION_SIGNATURE"
      ) {
        res.status(400).json({
          error:
            "invalid mint transaction request",
          code:
            error.code,
          message:
            error.message,
        });
        return;
      }

      if (
        error.kind ===
        "FATAL"
      ) {
        res.status(422).json({
          error:
            "mint transaction failed fatally",
          code:
            error.code,
          message:
            error.message,
        });
        return;
      }

      res.status(503).json({
        error:
          "mint transaction failed retryably",
        code:
          error.code,
        message:
          error.message,
      });
      return;
    }

    if (
      error instanceof
      MintEstimateExecutionError
    ) {
      res.status(503).json({
        error:
          "mint funding estimate unavailable",
        message:
          error.message,
      });
      return;
    }

    if (
      error instanceof
      MintOperationPayloadConflictError
    ) {
      res.status(409).json({
        error:
          "idempotency conflict",
        productId:
          error.productId,
      });
      return;
    }

    if (
      error instanceof
        MintOperationStateConflictError ||
      error instanceof
        MintOperationSignedTransactionConflictError
    ) {
      res.status(409).json({
        error:
          "mint operation conflict",
        productId:
          error.productId,
      });
      return;
    }

    if (
      error instanceof
      MintV2UsecaseInvalidStateError
    ) {
      res.status(409).json({
        error:
          "invalid mint operation state",
        productId:
          error.productId,
        status:
          error.status,
      });
      return;
    }

    if (
      error instanceof
      MintV2UsecaseStoredFatalError
    ) {
      res.status(422).json({
        error:
          "mint operation failed fatally",
        productId:
          error.productId,
        errorCode:
          error.errorCode,
      });
      return;
    }

    if (
      error instanceof
      MintOperationNotFoundError
    ) {
      res.status(404).json({
        error:
          "mint operation not found",
        productId:
          error.productId,
      });
      return;
    }

    res.status(500).json({
      error:
        "internal server error",
      message:
        error instanceof Error
          ? error.message
          : String(error),
    });
  },
);