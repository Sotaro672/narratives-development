// services/solana-bubblegum/src/app.ts

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

import {
  isMintV2TransactionError,
} from "./application/ports/mint-v2-transaction-port.js";

import {
  getBubblegumRuntime,
  getMintV2Usecase,
} from "./bootstrap/container.js";

type MintRequestBody = {
  productId?: unknown;
  tokenBlueprintId?: unknown;
  brandId?: unknown;
  name?: unknown;
  symbol?: unknown;
  metadataUri?: unknown;
};

class HttpRequestValidationError extends Error {
  readonly name = "HttpRequestValidationError";

  constructor(
    readonly field: string,
    message: string,
  ) {
    super([
      "http: invalid request",
      `field=${field}`,
      message,
    ].join(" "));
  }
}

function readMintRequestBody(value: unknown): MintRequestBody {
  if (
    value === null ||
    typeof value !== "object" ||
    Array.isArray(value)
  ) {
    throw new HttpRequestValidationError(
      "body",
      "JSON object is required",
    );
  }

  return value as MintRequestBody;
}

function requiredString(
  field: string,
  value: unknown,
): string {
  if (
    typeof value !== "string" ||
    value.length === 0
  ) {
    throw new HttpRequestValidationError(
      field,
      "value is required",
    );
  }

  return value;
}

function stringValue(
  field: string,
  value: unknown,
): string {
  if (typeof value !== "string") {
    throw new HttpRequestValidationError(
      field,
      "value must be string",
    );
  }

  return value;
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
  (_req: Request, res: Response) => {
    res.status(200).json({
      status: "ok",
      service: "solana-bubblegum",
    });
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
      const body = readMintRequestBody(req.body);

      const productId = requiredString(
        "productId",
        body.productId,
      );

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

      const [
        runtime,
        mintV2Usecase,
      ] = await Promise.all([
        getBubblegumRuntime(),
        getMintV2Usecase(),
      ]);

      const mintAuthorityAddress = String(
        runtime.mintAuthority.publicKey,
      );

      const result = await mintV2Usecase.execute({
        productId,
        tokenBlueprintId,
        brandId,
        leafOwnerAddress: mintAuthorityAddress,
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