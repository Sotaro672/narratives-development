// services/solana-bubblegum/src/app.ts

import express, { type NextFunction, type Request, type Response } from "express";

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
import { env } from "./config/env.js";

import {
  getBubblegumRuntime,
  getMintFundingEstimateUsecase,
  getMintV2Usecase,
} from "./bootstrap/container.js";

type MintRequestBody = {
  productId?: unknown;
  tokenBlueprintId?: unknown;
  brandId?: unknown;
  toAddress?: unknown;
  name?: unknown;
  symbol?: unknown;
  metadataUri?: unknown;
};

type MintEstimateRequestBody = {
  tokenBlueprintId?: unknown;
  mintQuantity?: unknown;
  toAddress?: unknown;
  name?: unknown;
  symbol?: unknown;
};

type OwnedAssetsRequestBody = {
  assetStandard?: unknown;
  walletAddress?: unknown;
};

type DasAsset = {
  id?: unknown;
  compression?: unknown;
};

type DasGetAssetsByOwnerResult = {
  items?: unknown;
};

type DasJsonRpcError = {
  code?: unknown;
  message?: unknown;
};

type DasJsonRpcResponse = {
  result?: unknown;
  error?: unknown;
};

class HttpRequestValidationError extends Error {
  readonly name = "HttpRequestValidationError";

  constructor(readonly field: string, message: string) {
    super(["http: invalid request", `field=${field}`, message].join(" "));
  }
}

class MintEstimateExecutionError extends Error {
  readonly name = "MintEstimateExecutionError";

  constructor(readonly cause: unknown) {
    super("mint funding estimate is temporarily unavailable");
  }
}

class OwnedAssetsExecutionError extends Error {
  readonly name = "OwnedAssetsExecutionError";

  constructor(readonly cause: unknown) {
    super("owned assets lookup is temporarily unavailable");
  }
}

function readMintRequestBody(value: unknown): MintRequestBody {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    throw new HttpRequestValidationError("body", "JSON object is required");
  }

  return value as MintRequestBody;
}

function readMintEstimateRequestBody(value: unknown): MintEstimateRequestBody {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    throw new HttpRequestValidationError("body", "JSON object is required");
  }

  return value as MintEstimateRequestBody;
}

function readOwnedAssetsRequestBody(value: unknown): OwnedAssetsRequestBody {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    throw new HttpRequestValidationError("body", "JSON object is required");
  }

  return value as OwnedAssetsRequestBody;
}

function requiredString(field: string, value: unknown): string {
  if (typeof value !== "string" || value.length === 0) {
    throw new HttpRequestValidationError(field, "value is required");
  }

  return value;
}

function stringValue(field: string, value: unknown): string {
  if (typeof value !== "string") {
    throw new HttpRequestValidationError(field, "value must be string");
  }

  return value;
}

function requiredPositiveInteger(field: string, value: unknown): number {
  if (
    typeof value !== "number" ||
    !Number.isSafeInteger(value) ||
    value <= 0
  ) {
    throw new HttpRequestValidationError(
      field,
      "value must be a positive integer",
    );
  }

  return value;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

function readDasAsset(value: unknown): DasAsset | null {
  if (!isRecord(value)) {
    return null;
  }

  return {
    id: value.id,
    compression: value.compression,
  };
}

function isCompressedDasAsset(asset: DasAsset): boolean {
  if (!isRecord(asset.compression)) {
    return false;
  }

  return asset.compression.compressed === true;
}

function readDasErrorMessage(value: unknown): string {
  if (!isRecord(value)) {
    return "";
  }

  const error = value as DasJsonRpcError;

  if (typeof error.message === "string" && error.message.length > 0) {
    return error.message;
  }

  if (typeof error.code === "number") {
    return `DAS RPC error code=${error.code}`;
  }

  return "";
}

async function fetchOwnedBubblegumAssetIDs(
  walletAddress: string,
): Promise<string[]> {
  const pageSize = 1000;
  const maxPages = 100;
  const assetIDs: string[] = [];
  const seen = new Set<string>();

  for (let page = 1; page <= maxPages; page += 1) {
    const response = await fetch(env.solanaRpcURL, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
      },
      body: JSON.stringify({
        jsonrpc: "2.0",
        id: `owned-assets-${page}`,
        method: "getAssetsByOwner",
        params: {
          ownerAddress: walletAddress,
          page,
          limit: pageSize,
        },
      }),
    });

    if (!response.ok) {
      throw new Error(
        `DAS getAssetsByOwner returned HTTP ${response.status}`,
      );
    }

    let payload: unknown;

    try {
      payload = await response.json();
    } catch (error) {
      throw new Error(
        `DAS getAssetsByOwner returned invalid JSON: ${
          error instanceof Error ? error.message : String(error)
        }`,
      );
    }

    if (!isRecord(payload)) {
      throw new Error("DAS getAssetsByOwner returned invalid response");
    }

    const rpcResponse = payload as DasJsonRpcResponse;

    if (rpcResponse.error !== undefined) {
      const message = readDasErrorMessage(rpcResponse.error);

      throw new Error(
        message
          ? `DAS getAssetsByOwner failed: ${message}`
          : "DAS getAssetsByOwner failed",
      );
    }

    if (!isRecord(rpcResponse.result)) {
      throw new Error("DAS getAssetsByOwner result is missing");
    }

    const result = rpcResponse.result as DasGetAssetsByOwnerResult;

    if (!Array.isArray(result.items)) {
      throw new Error("DAS getAssetsByOwner items is missing");
    }

    for (const value of result.items) {
      const asset = readDasAsset(value);

      if (!asset || !isCompressedDasAsset(asset)) {
        continue;
      }

      if (typeof asset.id !== "string" || asset.id.length === 0) {
        continue;
      }

      if (seen.has(asset.id)) {
        continue;
      }

      seen.add(asset.id);
      assetIDs.push(asset.id);
    }

    if (result.items.length < pageSize) {
      return assetIDs;
    }
  }

  throw new Error(
    `DAS getAssetsByOwner exceeded pagination limit pages=${maxPages}`,
  );
}

export const app = express();

app.disable("x-powered-by");
app.use(express.json({ limit: "32kb" }));

app.get("/health", (_req: Request, res: Response) => {
  res.status(200).json({
    status: "ok",
    service: "solana-bubblegum",
  });
});

app.post(
  "/owned-assets",
  async (req: Request, res: Response, next: NextFunction) => {
    try {
      const body = readOwnedAssetsRequestBody(req.body);

      const assetStandard = requiredString(
        "assetStandard",
        body.assetStandard,
      );

      const walletAddress = requiredString(
        "walletAddress",
        body.walletAddress,
      );

      if (assetStandard !== "BUBBLEGUM_V2") {
        throw new HttpRequestValidationError(
          "assetStandard",
          "only BUBBLEGUM_V2 is supported",
        );
      }

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
  },
);

app.post(
  "/estimate",
  async (req: Request, res: Response, next: NextFunction) => {
    try {
      const body = readMintEstimateRequestBody(req.body);

      const tokenBlueprintId = requiredString(
        "tokenBlueprintId",
        body.tokenBlueprintId,
      );

      const mintQuantity = requiredPositiveInteger(
        "mintQuantity",
        body.mintQuantity,
      );

      const toAddress = requiredString(
        "toAddress",
        body.toAddress,
      );

      const name = requiredString(
        "name",
        body.name,
      );

      const symbol = stringValue(
        "symbol",
        body.symbol,
      );

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
  },
);

app.post(
  "/mint",
  async (req: Request, res: Response, next: NextFunction) => {
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

      const tokenBlueprintId = requiredString(
        "tokenBlueprintId",
        body.tokenBlueprintId,
      );

      const brandId = requiredString(
        "brandId",
        body.brandId,
      );

      const toAddress = requiredString(
        "toAddress",
        body.toAddress,
      );

      const name = requiredString(
        "name",
        body.name,
      );

      const symbol = stringValue(
        "symbol",
        body.symbol,
      );

      const metadataUri = requiredString(
        "metadataUri",
        body.metadataUri,
      );

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
  },
);

app.use((_req: Request, res: Response) => {
  res.status(404).json({
    error: "not found",
  });
});

app.use(
  (
    error: unknown,
    _req: Request,
    res: Response,
    _next: NextFunction,
  ) => {
    console.error("[http]", error);

    if (error instanceof SyntaxError) {
      res.status(400).json({
        error: "invalid JSON body",
      });
      return;
    }

    if (error instanceof HttpRequestValidationError) {
      res.status(400).json({
        error: "invalid request",
        field: error.field,
        message: error.message,
      });
      return;
    }

    if (error instanceof OwnedAssetsExecutionError) {
      res.status(503).json({
        error: "owned assets unavailable",
        message: error.message,
      });
      return;
    }

    if (error instanceof MintV2UsecaseValidationError) {
      res.status(400).json({
        error: "invalid mint request",
        field: error.field,
        message: error.message,
      });
      return;
    }

    if (isMintV2TransactionError(error)) {
      if (
        error.code === "INVALID_INPUT" ||
        error.code === "INVALID_PUBLIC_KEY" ||
        error.code === "INVALID_SIGNATURE" ||
        error.code === "INVALID_TRANSACTION_SIGNATURE"
      ) {
        res.status(400).json({
          error: "invalid mint transaction request",
          code: error.code,
          message: error.message,
        });
        return;
      }

      if (error.kind === "FATAL") {
        res.status(422).json({
          error: "mint transaction failed fatally",
          code: error.code,
          message: error.message,
        });
        return;
      }

      res.status(503).json({
        error: "mint transaction failed retryably",
        code: error.code,
        message: error.message,
      });
      return;
    }

    if (error instanceof MintEstimateExecutionError) {
      res.status(503).json({
        error: "mint funding estimate unavailable",
        message: error.message,
      });
      return;
    }

    if (error instanceof MintOperationPayloadConflictError) {
      res.status(409).json({
        error: "idempotency conflict",
        productId: error.productId,
      });
      return;
    }

    if (
      error instanceof MintOperationStateConflictError ||
      error instanceof MintOperationSignedTransactionConflictError
    ) {
      res.status(409).json({
        error: "mint operation conflict",
        productId: error.productId,
      });
      return;
    }

    if (error instanceof MintV2UsecaseInvalidStateError) {
      res.status(409).json({
        error: "invalid mint operation state",
        productId: error.productId,
        status: error.status,
      });
      return;
    }

    if (error instanceof MintV2UsecaseStoredFatalError) {
      res.status(422).json({
        error: "mint operation failed fatally",
        productId: error.productId,
        errorCode: error.errorCode,
      });
      return;
    }

    if (error instanceof MintOperationNotFoundError) {
      res.status(404).json({
        error: "mint operation not found",
        productId: error.productId,
      });
      return;
    }

    res.status(500).json({
      error: "internal server error",
    });
  },
);