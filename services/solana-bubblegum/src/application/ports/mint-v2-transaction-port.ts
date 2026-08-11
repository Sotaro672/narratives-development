// services/solana-bubblegum/src/application/ports/mint-v2-transaction-port.ts

export type MintV2Creator = {
  address: string;

  verified: boolean;

  share: number;
};


export type MintV2Metadata = {
  name: string;

  symbol: string;

  uri: string;

  sellerFeeBasisPoints: number;

  primarySaleHappened: boolean;

  isMutable: boolean;

  creators:
    MintV2Creator[];
};


export type BuildAndSignMintV2TransactionInput = {
  treeAddress: string;

  leafOwnerAddress: string;

  leafDelegateAddress:
    string | null;

  coreCollectionAddress:
    string | null;

  metadata:
    MintV2Metadata;
};


export type BuildAndSignMintV2TransactionResult = {
  signature: string;

  signedTransactionBase64: string;
};


export type BroadcastMintV2TransactionInput = {
  signature: string;

  signedTransactionBase64: string;
};


export type BroadcastMintV2TransactionResult = {
  signature: string;
};


export type WaitForMintV2FinalizedInput = {
  signature: string;
};


export type WaitForMintV2FinalizedResult = {
  slot: number;
};


export type ParseMintV2ResultInput = {
  signature: string;
};


export type ParseMintV2ResultResult = {
  assetId: string;

  leafIndex: number;
};


export interface MintV2TransactionPort {
  buildAndSign(
    input: BuildAndSignMintV2TransactionInput,
  ): Promise<BuildAndSignMintV2TransactionResult>;

  broadcast(
    input: BroadcastMintV2TransactionInput,
  ): Promise<BroadcastMintV2TransactionResult>;

  waitForFinalized(
    input: WaitForMintV2FinalizedInput,
  ): Promise<WaitForMintV2FinalizedResult>;

  parseMintResult(
    input: ParseMintV2ResultInput,
  ): Promise<ParseMintV2ResultResult>;
}


export type MintV2TransactionErrorKind =
  | "RETRYABLE"
  | "FATAL";


export class MintV2TransactionError
  extends Error {
  readonly name =
    "MintV2TransactionError";

  constructor(
    readonly kind:
      MintV2TransactionErrorKind,

    readonly code:
      string,

    message:
      string,

    options?: {
      cause?: unknown;
    },
  ) {
    super(
      message,
      options,
    );
  }
}


export function isMintV2TransactionError(
  error: unknown,
): error is MintV2TransactionError {
  return (
    error instanceof
    MintV2TransactionError
  );
}