// services/solana-bubblegum/src/routes/transfer-route.ts

import { Router } from "express";

import {
  executeBubblegumTransfer,
  type TransferExecutionResult,
} from "../application/execute-bubblegum-transfer.js";
import {
  HttpRequestValidationError,
  TransferExecutionError,
  TransferOwnershipConflictError,
  TransferSignerMismatchError,
} from "../http/errors.js";
import {
  optionalString,
  parseSolanaPublicKey,
  readTransferRequestBody,
  requiredString,
} from "../http/request-validation.js";

export const transferRouter = Router();

transferRouter.post("/transfer", async (req, res, next) => {
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