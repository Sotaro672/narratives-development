//services\solana-bubblegum\src\http\errors.ts
export class HttpRequestValidationError extends Error {
  readonly name = "HttpRequestValidationError";

  constructor(
    readonly field: string,
    message: string,
  ) {
    super(["http: invalid request", `field=${field}`, message].join(" "));
  }
}

export class MintEstimateExecutionError extends Error {
  readonly name = "MintEstimateExecutionError";

  constructor(readonly cause: unknown) {
    super(cause instanceof Error ? cause.message : String(cause));
  }
}

export class OwnedAssetsExecutionError extends Error {
  readonly name = "OwnedAssetsExecutionError";

  constructor(readonly cause: unknown) {
    super(cause instanceof Error ? cause.message : String(cause));
  }
}

export class TransferExecutionError extends Error {
  readonly name = "TransferExecutionError";

  constructor(readonly cause: unknown) {
    super(cause instanceof Error ? cause.message : String(cause));
  }
}

export class TransferOwnershipConflictError extends Error {
  readonly name = "TransferOwnershipConflictError";

  constructor(
    readonly expectedOwner: string,
    readonly actualOwner: string,
  ) {
    super(
      [
        "transfer: asset owner mismatch",
        `expectedOwner=${expectedOwner}`,
        `actualOwner=${actualOwner}`,
      ].join(" "),
    );
  }
}

export class TransferSignerMismatchError extends Error {
  readonly name = "TransferSignerMismatchError";

  constructor(
    readonly expectedAddress: string,
    readonly signerAddress: string,
  ) {
    super(
      [
        "transfer: sender signer address mismatch",
        `expectedAddress=${expectedAddress}`,
        `signerAddress=${signerAddress}`,
      ].join(" "),
    );
  }
}