// services/solana-bubblegum/src/routes/estimate-route.ts

import { Router } from "express";

import { isMintV2TransactionError } from "../application/ports/mint-v2-transaction-port.js";
import {
  getBubblegumRuntime,
  getMintFundingEstimateUsecase,
} from "../bootstrap/container.js";
import { MintEstimateExecutionError } from "../http/errors.js";
import {
  readMintEstimateRequestBody,
  requiredPositiveInteger,
  requiredString,
  stringValue,
} from "../http/request-validation.js";

export const estimateRouter = Router();

estimateRouter.post("/estimate", async (req, res, next) => {
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