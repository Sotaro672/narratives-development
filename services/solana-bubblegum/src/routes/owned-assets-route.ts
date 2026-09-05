// services/solana-bubblegum/src/routes/owned-assets-route.ts

import { Router } from "express";

import {
  HttpRequestValidationError,
  OwnedAssetsExecutionError,
} from "../http/errors.js";
import {
  parseSolanaPublicKey,
  readOwnedAssetsRequestBody,
  requiredString,
} from "../http/request-validation.js";
import { fetchOwnedBubblegumAssetIDs } from "../infrastructure/das/das-client.js";

export const ownedAssetsRouter = Router();

ownedAssetsRouter.post("/owned-assets", async (req, res, next) => {
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