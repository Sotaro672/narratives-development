// services/solana-bubblegum/src/infrastructure/das/das-types.ts

import type {
  PublicKey,
} from "@metaplex-foundation/umi";

export type DasAsset = {
  id?: unknown;
  compression?: unknown;
};

export type DasGetAssetsByOwnerResult = {
  items?: unknown;
};

export type DasJsonRpcError = {
  code?: unknown;
  message?: unknown;
};

export type DasJsonRpcResponse = {
  result?: unknown;
  error?: unknown;
};

export type DasTransferGrouping = {
  groupKey: string;
  groupValue: string;
};

export type DasTransferCompression = {
  compressed: boolean;
  dataHash: string;
  creatorHash: string;
  assetDataHash?: string;
  flags?: number;
  leafId: number;
};

export type DasTransferOwnership = {
  owner: string;
  delegate: string;
};

export type DasTransferAsset = {
  id: string;
  compression: DasTransferCompression;
  ownership: DasTransferOwnership;
  grouping: DasTransferGrouping[];
};

export type DasTransferProof = {
  root: string;
  proof: string[];
  nodeIndex: number;
  treeId: string;
};

export type BubblegumTransferAssetWithProof = {
  leafOwner: PublicKey;
  leafDelegate: PublicKey;
  merkleTree: PublicKey;
  root: Uint8Array;
  dataHash: Uint8Array;
  creatorHash: Uint8Array;
  assetDataHash?: Uint8Array;
  flags?: number;
  nonce: number;
  index: number;
  proof: PublicKey[];
  asset: DasTransferAsset;
};