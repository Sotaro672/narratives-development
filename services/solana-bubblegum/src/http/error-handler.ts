// services/solana-bubblegum/src/http/error-handler.ts

import type {
  NextFunction,
  Request,
  Response,
} from "express";

import {
  MintV2UsecaseInvalidStateError,
  MintV2UsecaseStoredFatalError,
  MintV2UsecaseValidationError,
} from "../application/mint-v2-usecase.js";
import {
  MintOperationNotFoundError,
  MintOperationPayloadConflictError,
  MintOperationSignedTransactionConflictError,
  MintOperationStateConflictError,
} from "../application/ports/mint-operation-registry-port.js";
import { isMintV2TransactionError } from "../application/ports/mint-v2-transaction-port.js";
import {
  HttpRequestValidationError,
  MintEstimateExecutionError,
  OwnedAssetsExecutionError,
  TransferExecutionError,
  TransferOwnershipConflictError,
  TransferSignerMismatchError,
} from "./errors.js";

export function errorHandler(
  error: unknown,
  _req: Request,
  res: Response,
  _next: NextFunction,
): void {
  if (error instanceof SyntaxError) {
    res.status(400).json({
      error: "invalid JSON body",
      message: error.message,
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

  if (error instanceof TransferOwnershipConflictError) {
    res.status(409).json({
      error: "transfer ownership conflict",
      message: error.message,
      expectedOwner: error.expectedOwner,
      actualOwner: error.actualOwner,
    });
    return;
  }

  if (error instanceof TransferSignerMismatchError) {
    res.status(409).json({
      error: "transfer signer mismatch",
      message: error.message,
      expectedAddress: error.expectedAddress,
      signerAddress: error.signerAddress,
    });
    return;
  }

  if (error instanceof TransferExecutionError) {
    res.status(503).json({
      error: "transfer unavailable",
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
    message: error instanceof Error ? error.message : String(error),
  });
}