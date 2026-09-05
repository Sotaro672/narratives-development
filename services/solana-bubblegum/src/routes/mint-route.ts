// services/solana-bubblegum/src/routes/mint-route.ts

import { Router } from "express";

import {
  getBubblegumRuntime,
  getMintV2Usecase,
} from "../bootstrap/container.js";
import { HttpRequestValidationError } from "../http/errors.js";
import {
  readMintRequestBody,
  requiredString,
  stringValue,
} from "../http/request-validation.js";

export const mintRouter = Router();

mintRouter.post("/mint", async (req, res, next) => {
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