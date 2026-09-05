// services/solana-bubblegum/src/app.ts

import express, { type NextFunction, type Request, type Response } from "express";

import {
  executeBubblegumTransfer,
  type TransferExecutionResult,
} from "./application/execute-bubblegum-transfer.js";
import { isMintV2TransactionError } from "./application/ports/mint-v2-transaction-port.js";
import {
  getBubblegumRuntime,
  getMintFundingEstimateUsecase,
  getMintV2Usecase,
} from "./bootstrap/container.js";
import { errorHandler } from "./http/error-handler.js";
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
import { fetchOwnedBubblegumAssetIDs } from "./infrastructure/das/das-client.js";

export const app = express();

app.disable("x-powered-by");
app.use(express.json({ limit: "32kb" }));

app.get("/health", (_req: Request, res: Response) => {
  res.status(200).json({
    status: "ok",
    service: "solana-bubblegum",
  });
});

app.post("/owned-assets", async (req: Request, res: Response, next: NextFunction) => {
  try {
    const body = readOwnedAssetsRequestBody(req.body);
    const assetStandard = requiredString("assetStandard", body.assetStandard);
    const walletAddress = requiredString("walletAddress", body.walletAddress);

    if (assetStandard !== "BUBBLEGUM_V2") {
      throw new HttpRequestValidationError(
        "assetStandard",
        "only BUBBLEGUM_V2 is supported",
      );
    }

    parseSolanaPublicKey("walletAddress", walletAddress);

    let assetIds: string[];

    try {
      assetIds = await fetchOwnedBubblegumAssetIDs(walletAddress);
    } catch (error) {
      throw new OwnedAssetsExecutionError(error);
    }

    res.status(200).json({
      walletAddress,
      assetIds,
    });
  } catch (error) {
    next(error);
  }
});

app.post("/transfer", async (req: Request, res: Response, next: NextFunction) => {
  try {
    const body = readTransferRequestBody(req.body);
    const productId = requiredString("productId", body.productId);
    const assetStandard = requiredString("assetStandard", body.assetStandard);

    if (assetStandard !== "BUBBLEGUM_V2") {
      throw new HttpRequestValidationError(
        "assetStandard",
        "only BUBBLEGUM_V2 is supported",
      );
    }

    const assetId = requiredString("assetId", body.assetId);
    const fromAvatarId = optionalString("fromAvatarId", body.fromAvatarId);
    const fromBrandId = optionalString("fromBrandId", body.fromBrandId);
    const toAvatarId = requiredString("toAvatarId", body.toAvatarId);
    const fromWalletAddress = requiredString("fromWalletAddress", body.fromWalletAddress);
    const toWalletAddress = requiredString("toWalletAddress", body.toWalletAddress);

    if (Boolean(fromAvatarId) === Boolean(fromBrandId)) {
      throw new HttpRequestValidationError(
        "sender",
        "exactly one of fromAvatarId or fromBrandId is required",
      );
    }

    parseSolanaPublicKey("assetId", assetId);
    parseSolanaPublicKey("fromWalletAddress", fromWalletAddress);
    parseSolanaPublicKey("toWalletAddress", toWalletAddress);

    let result: TransferExecutionResult;

    try {
      result = await executeBubblegumTransfer({
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
        error instanceof HttpRequestValidationError ||
        error instanceof TransferOwnershipConflictError ||
        error instanceof TransferSignerMismatchError
      ) {
        throw error;
      }

      throw new TransferExecutionError(error);
    }

    res.status(200).json({
      signature: result.signature,
      assetId: result.assetId,
    });
  } catch (error) {
    next(error);
  }
});

app.post("/estimate", async (req: Request, res: Response, next: NextFunction) => {
  try {
    const body = readMintEstimateRequestBody(req.body);
    const tokenBlueprintId = requiredString("tokenBlueprintId", body.tokenBlueprintId);
    const mintQuantity = requiredPositiveInteger("mintQuantity", body.mintQuantity);
    const toAddress = requiredString("toAddress", body.toAddress);
    const name = requiredString("name", body.name);
    const symbol = stringValue("symbol", body.symbol);

    let result;

    try {
      const runtime = await getBubblegumRuntime();
      const mintFundingEstimateUsecase = getMintFundingEstimateUsecase();

      result = await mintFundingEstimateUsecase.execute({
        tokenBlueprintId,
        mintQuantity,
        leafOwnerAddress: toAddress,
        name,
        symbol,
        umi: runtime.umi,
        feePayer: runtime.feePayer,
        reserve: runtime.reserve,
      });
    } catch (error) {
      if (isMintV2TransactionError(error)) {
        throw error;
      }

      throw new MintEstimateExecutionError(error);
    }

    res.status(200).json(result);
  } catch (error) {
    next(error);
  }
});

app.post("/mint", async (req: Request, res: Response, next: NextFunction) => {
  try {
    const body = readMintRequestBody(req.body);
    const productId = requiredString("productId", body.productId);
    const idempotencyKey = req.get("Idempotency-Key");

    if (!idempotencyKey) {
      throw new HttpRequestValidationError(
        "Idempotency-Key",
        "header is required",
      );
    }

    if (idempotencyKey !== productId) {
      throw new HttpRequestValidationError(
        "Idempotency-Key",
        "header must equal productId",
      );
    }

    const tokenBlueprintId = requiredString("tokenBlueprintId", body.tokenBlueprintId);
    const brandId = requiredString("brandId", body.brandId);
    const toAddress = requiredString("toAddress", body.toAddress);
    const name = requiredString("name", body.name);
    const symbol = stringValue("symbol", body.symbol);
    const metadataUri = requiredString("metadataUri", body.metadataUri);

    const [runtime, mintV2Usecase] = await Promise.all([
      getBubblegumRuntime(),
      getMintV2Usecase(),
    ]);

    const result = await mintV2Usecase.execute({
      productId,
      tokenBlueprintId,
      brandId,
      leafOwnerAddress: toAddress,
      leafDelegateAddress: null,
      coreCollection: {
        name,
        metadataUri,
      },
      metadata: {
        name,
        symbol,
        uri: metadataUri,
        sellerFeeBasisPoints: 0,
        primarySaleHappened: false,
        isMutable: false,
        creators: [],
      },
      umi: runtime.umi,
      feePayer: runtime.feePayer,
      reserve: runtime.reserve,
    });

    res.status(200).json(result);
  } catch (error) {
    next(error);
  }
});

app.use((_req: Request, res: Response) => {
  res.status(404).json({
    error: "not found",
  });
});

app.use(errorHandler);