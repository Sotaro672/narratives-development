// services/solana-bubblegum/src/app.ts

import { Buffer } from "node:buffer";
import { SecretManagerServiceClient } from "@google-cloud/secret-manager";
import {
  isValidLeafSchemaV2Flags,
  transferV2,
} from "@metaplex-foundation/mpl-bubblegum";
import { fetchMerkleTree } from "@metaplex-foundation/spl-account-compression";
import {
  createSignerFromKeypair,
  publicKey,
  publicKeyBytes,
  type KeypairSigner,
  type PublicKey,
  type Umi,
} from "@metaplex-foundation/umi";
import { base58 } from "@metaplex-foundation/umi/serializers";
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

type TransferRequestBody = {
  productId?: unknown;
  assetStandard?: unknown;
  assetId?: unknown;
  fromAvatarId?: unknown;
  fromBrandId?: unknown;
  toAvatarId?: unknown;
  brandId?: unknown;
  modelId?: unknown;
  tokenBlueprintId?: unknown;
  fromWalletAddress?: unknown;
  toWalletAddress?: unknown;
};

type TransferExecutionInput = {
  productId: string;
  assetId: string;
  fromAvatarId: string;
  fromBrandId: string;
  toAvatarId: string;
  fromWalletAddress: string;
  toWalletAddress: string;
};

type TransferExecutionResult = {
  signature: string;
  assetId: string;
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

type DasTransferGrouping = {
  groupKey: string;
  groupValue: string;
};

type DasTransferCompression = {
  compressed: boolean;
  dataHash: string;
  creatorHash: string;
  assetDataHash?: string;
  flags?: number;
  leafId: number;
};

type DasTransferOwnership = {
  owner: string;
  delegate: string;
};

type DasTransferAsset = {
  id: string;
  compression: DasTransferCompression;
  ownership: DasTransferOwnership;
  grouping: DasTransferGrouping[];
};

type DasTransferProof = {
  root: string;
  proof: string[];
  nodeIndex: number;
  treeId: string;
};

type BubblegumTransferAssetWithProof = {
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

class HttpRequestValidationError extends Error {
  readonly name = "HttpRequestValidationError";

  constructor(
    readonly field: string,
    message: string,
  ) {
    super(["http: invalid request", `field=${field}`, message].join(" "));
  }
}

class MintEstimateExecutionError extends Error {
  readonly name = "MintEstimateExecutionError";

  constructor(readonly cause: unknown) {
    super(cause instanceof Error ? cause.message : String(cause));
  }
}

class OwnedAssetsExecutionError extends Error {
  readonly name = "OwnedAssetsExecutionError";

  constructor(readonly cause: unknown) {
    super(cause instanceof Error ? cause.message : String(cause));
  }
}

class TransferExecutionError extends Error {
  readonly name = "TransferExecutionError";

  constructor(readonly cause: unknown) {
    super(cause instanceof Error ? cause.message : String(cause));
  }
}

class TransferOwnershipConflictError extends Error {
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

class TransferSignerMismatchError extends Error {
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

const secretManagerClient = new SecretManagerServiceClient();

function readMintRequestBody(value: unknown): MintRequestBody {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    throw new HttpRequestValidationError("body", "JSON object is required");
  }

  return value as MintRequestBody;
}

function readMintEstimateRequestBody(
  value: unknown,
): MintEstimateRequestBody {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    throw new HttpRequestValidationError("body", "JSON object is required");
  }

  return value as MintEstimateRequestBody;
}

function readOwnedAssetsRequestBody(
  value: unknown,
): OwnedAssetsRequestBody {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    throw new HttpRequestValidationError("body", "JSON object is required");
  }

  return value as OwnedAssetsRequestBody;
}

function readTransferRequestBody(value: unknown): TransferRequestBody {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    throw new HttpRequestValidationError("body", "JSON object is required");
  }

  return value as TransferRequestBody;
}

function requiredString(field: string, value: unknown): string {
  if (typeof value !== "string" || value.length === 0) {
    throw new HttpRequestValidationError(field, "value is required");
  }

  return value;
}

function optionalString(field: string, value: unknown): string {
  if (value === undefined || value === null) {
    return "";
  }

  if (typeof value !== "string") {
    throw new HttpRequestValidationError(field, "value must be string");
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

function parseSolanaPublicKey(
  field: string,
  value: string,
): PublicKey {
  try {
    return publicKey(value);
  } catch (error) {
    const detail =
      error instanceof Error
        ? error.message
        : String(error);

    throw new HttpRequestValidationError(
      field,
      `value must be a valid Solana public key detail=${detail}`,
    );
  }
}

function isRecord(
  value: unknown,
): value is Record<string, unknown> {
  return (
    value !== null &&
    typeof value === "object" &&
    !Array.isArray(value)
  );
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

  if (
    typeof error.message === "string" &&
    error.message.length > 0
  ) {
    return error.message;
  }

  if (typeof error.code === "number") {
    return `DAS RPC error code=${error.code}`;
  }

  return "";
}

function parseSecretKey(
  secretID: string,
  raw: string,
): Uint8Array {
  let parsed: unknown;

  try {
    parsed = JSON.parse(raw);
  } catch (error) {
    throw new Error(
      [
        "transfer: invalid sender secret JSON",
        `secret=${secretID}`,
        `detail=${
          error instanceof Error
            ? error.message
            : String(error)
        }`,
      ].join(" "),
    );
  }

  if (
    !Array.isArray(parsed) ||
    parsed.length !== 64
  ) {
    throw new Error(
      [
        "transfer: invalid sender Solana keypair",
        `secret=${secretID}`,
        "expectedLength=64",
      ].join(" "),
    );
  }

  const bytes: number[] = [];

  for (const value of parsed) {
    if (
      typeof value !== "number" ||
      !Number.isInteger(value) ||
      value < 0 ||
      value > 255
    ) {
      throw new Error(
        [
          "transfer: invalid sender Solana keypair byte",
          `secret=${secretID}`,
        ].join(" "),
      );
    }

    bytes.push(value);
  }

  return Uint8Array.from(bytes);
}

async function loadSenderSigner(
  umi: Umi,
  input: {
    fromAvatarId: string;
    fromBrandId: string;
    fromWalletAddress: string;
  },
): Promise<KeypairSigner> {
  const hasAvatar = input.fromAvatarId.length > 0;
  const hasBrand = input.fromBrandId.length > 0;

  if (hasAvatar === hasBrand) {
    throw new HttpRequestValidationError(
      "sender",
      "exactly one of fromAvatarId or fromBrandId is required",
    );
  }

  const secretID =
    hasBrand
      ? `brand-wallet-${input.fromBrandId}`
      : `avatar-wallet-${input.fromAvatarId}`;

  const secretName =
    `projects/${env.googleCloudProject}/secrets/${secretID}/versions/latest`;

  let version;

  try {
    [version] = await secretManagerClient.accessSecretVersion({
      name: secretName,
    });
  } catch (error) {
    throw new Error(
      [
        "transfer: failed to load sender secret",
        `secret=${secretID}`,
        `detail=${
          error instanceof Error
            ? error.message
            : String(error)
        }`,
      ].join(" "),
    );
  }

  const data = version.payload?.data;

  if (!data) {
    throw new Error(
      [
        "transfer: sender secret payload is empty",
        `secret=${secretID}`,
      ].join(" "),
    );
  }

  const raw = Buffer
    .from(data)
    .toString("utf8");

  const secretKey = parseSecretKey(secretID, raw);
  const keypair =
    umi.eddsa.createKeypairFromSecretKey(secretKey);
  const signer =
    createSignerFromKeypair(umi, keypair);
  const signerAddress =
    String(signer.publicKey);

  if (signerAddress !== input.fromWalletAddress) {
    throw new TransferSignerMismatchError(
      input.fromWalletAddress,
      signerAddress,
    );
  }

  return signer;
}

async function fetchOwnedBubblegumAssetIDs(
  walletAddress: string,
): Promise<string[]> {
  const pageSize = 1000;
  const maxPages = 100;
  const assetIDs: string[] = [];
  const seen = new Set<string>();

  for (
    let page = 1;
    page <= maxPages;
    page += 1
  ) {
    const response = await fetch(
      env.solanaRpcURL,
      {
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
      },
    );

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
          error instanceof Error
            ? error.message
            : String(error)
        }`,
      );
    }

    if (!isRecord(payload)) {
      throw new Error(
        "DAS getAssetsByOwner returned invalid response",
      );
    }

    const rpcResponse =
      payload as DasJsonRpcResponse;

    if (rpcResponse.error !== undefined) {
      const message =
        readDasErrorMessage(
          rpcResponse.error,
        );

      throw new Error(
        message
          ? `DAS getAssetsByOwner failed: ${message}`
          : "DAS getAssetsByOwner failed",
      );
    }

    if (!isRecord(rpcResponse.result)) {
      throw new Error(
        "DAS getAssetsByOwner result is missing",
      );
    }

    const result =
      rpcResponse.result as DasGetAssetsByOwnerResult;

    if (!Array.isArray(result.items)) {
      throw new Error(
        "DAS getAssetsByOwner items is missing",
      );
    }

    for (const value of result.items) {
      const asset =
        readDasAsset(value);

      if (
        !asset ||
        !isCompressedDasAsset(asset)
      ) {
        continue;
      }

      if (
        typeof asset.id !== "string" ||
        asset.id.length === 0
      ) {
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

function requiredRecord(
  field: string,
  value: unknown,
): Record<string, unknown> {
  if (!isRecord(value)) {
    throw new Error(`DAS ${field} must be an object`);
  }

  return value;
}

function requiredDasString(
  field: string,
  value: unknown,
): string {
  if (typeof value !== "string" || value.length === 0) {
    throw new Error(
      `DAS ${field} must be a non-empty string`,
    );
  }

  return value;
}

function requiredDasSafeInteger(
  field: string,
  value: unknown,
): number {
  if (
    typeof value !== "number" ||
    !Number.isSafeInteger(value) ||
    value < 0
  ) {
    throw new Error(
      `DAS ${field} must be a non-negative safe integer`,
    );
  }

  return value;
}

function parseDasPublicKey(
  field: string,
  value: string,
): PublicKey {
  try {
    return publicKey(value);
  } catch (error) {
    throw new Error(
      [
        `DAS ${field} must be a valid Solana public key`,
        `value=${value}`,
        `detail=${
          error instanceof Error
            ? error.message
            : String(error)
        }`,
      ].join(" "),
    );
  }
}

async function callDasRpc(
  method: string,
  params: Record<string, unknown>,
  requestID: string,
): Promise<unknown> {
  const response = await fetch(
    env.solanaRpcURL,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
      },
      body: JSON.stringify({
        jsonrpc: "2.0",
        id: requestID,
        method,
        params,
      }),
    },
  );

  if (!response.ok) {
    throw new Error(
      `DAS ${method} returned HTTP ${response.status}`,
    );
  }

  let payload: unknown;

  try {
    payload = await response.json();
  } catch (error) {
    throw new Error(
      `DAS ${method} returned invalid JSON: ${
        error instanceof Error
          ? error.message
          : String(error)
      }`,
    );
  }

  if (!isRecord(payload)) {
    throw new Error(
      `DAS ${method} returned invalid response`,
    );
  }

  const rpcResponse =
    payload as DasJsonRpcResponse;

  if (rpcResponse.error !== undefined) {
    const message =
      readDasErrorMessage(
        rpcResponse.error,
      );

    throw new Error(
      message
        ? `DAS ${method} failed: ${message}`
        : `DAS ${method} failed`,
    );
  }

  if (
    rpcResponse.result === undefined ||
    rpcResponse.result === null
  ) {
    throw new Error(
      `DAS ${method} result is missing`,
    );
  }

  return rpcResponse.result;
}

function parseTransferDasAsset(
  value: unknown,
  expectedAssetID: string,
): DasTransferAsset {
  const asset =
    requiredRecord(
      "getAsset.result",
      value,
    );

  const assetID =
    requiredDasString(
      "getAsset.id",
      asset.id,
    );

  if (assetID !== expectedAssetID) {
    throw new Error(
      `DAS getAsset returned unexpected asset id expected=${expectedAssetID} actual=${assetID}`,
    );
  }

  const compression =
    requiredRecord(
      "getAsset.compression",
      asset.compression,
    );

  if (compression.compressed !== true) {
    throw new Error(
      `DAS asset is not compressed assetId=${assetID}`,
    );
  }

  const ownership =
    requiredRecord(
      "getAsset.ownership",
      asset.ownership,
    );

  const owner =
    requiredDasString(
      "getAsset.ownership.owner",
      ownership.owner,
    );

  const delegate =
    ownership.delegate === undefined ||
    ownership.delegate === null
      ? owner
      : requiredDasString(
          "getAsset.ownership.delegate",
          ownership.delegate,
        );

  const grouping: DasTransferGrouping[] = [];

  if (
    asset.grouping !== undefined &&
    asset.grouping !== null
  ) {
    if (!Array.isArray(asset.grouping)) {
      throw new Error(
        "DAS getAsset.grouping must be an array",
      );
    }

    for (const value of asset.grouping) {
      if (!isRecord(value)) {
        continue;
      }

      if (
        typeof value.group_key !== "string" ||
        typeof value.group_value !== "string" ||
        value.group_key.length === 0 ||
        value.group_value.length === 0
      ) {
        continue;
      }

      grouping.push({
        groupKey: value.group_key,
        groupValue: value.group_value,
      });
    }
  }

  const rawFlags =
    compression.flags;

  if (
    rawFlags !== undefined &&
    rawFlags !== null &&
    !isValidLeafSchemaV2Flags(rawFlags)
  ) {
    throw new Error(
      `DAS getAsset.compression.flags is invalid value=${String(rawFlags)}`,
    );
  }

  const assetDataHash =
    compression.asset_data_hash === undefined ||
    compression.asset_data_hash === null
      ? undefined
      : requiredDasString(
          "getAsset.compression.asset_data_hash",
          compression.asset_data_hash,
        );

  return {
    id: assetID,
    compression: {
      compressed: true,
      dataHash: requiredDasString(
        "getAsset.compression.data_hash",
        compression.data_hash,
      ),
      creatorHash: requiredDasString(
        "getAsset.compression.creator_hash",
        compression.creator_hash,
      ),
      assetDataHash,
      flags:
        rawFlags === undefined ||
        rawFlags === null
          ? undefined
          : rawFlags,
      leafId: requiredDasSafeInteger(
        "getAsset.compression.leaf_id",
        compression.leaf_id,
      ),
    },
    ownership: {
      owner,
      delegate,
    },
    grouping,
  };
}

function parseTransferDasProof(
  value: unknown,
): DasTransferProof {
  const result =
    requiredRecord(
      "getAssetProof.result",
      value,
    );

  if (!Array.isArray(result.proof)) {
    throw new Error(
      "DAS getAssetProof.proof must be an array",
    );
  }

  const proof =
    result.proof.map(
      (value, index) =>
        requiredDasString(
          `getAssetProof.proof[${index}]`,
          value,
        ),
    );

  return {
    root: requiredDasString(
      "getAssetProof.root",
      result.root,
    ),
    proof,
    nodeIndex: requiredDasSafeInteger(
      "getAssetProof.node_index",
      result.node_index,
    ),
    treeId: requiredDasString(
      "getAssetProof.tree_id",
      result.tree_id,
    ),
  };
}

async function fetchTransferAssetWithProof(
  umi: Umi,
  assetID: string,
): Promise<BubblegumTransferAssetWithProof> {
  const [
    assetValue,
    proofValue,
  ] =
    await Promise.all([
      callDasRpc(
        "getAsset",
        {
          id: assetID,
          options: {
            showUnverifiedCollections: true,
          },
        },
        `transfer-asset-${assetID}`,
      ),
      callDasRpc(
        "getAssetProof",
        {
          id: assetID,
        },
        `transfer-proof-${assetID}`,
      ),
    ]);

  const asset =
    parseTransferDasAsset(
      assetValue,
      assetID,
    );

  const rpcProof =
    parseTransferDasProof(
      proofValue,
    );

  const merkleTree =
    parseDasPublicKey(
      "getAssetProof.tree_id",
      rpcProof.treeId,
    );

  const fullProof =
    rpcProof.proof.map(
      (value, index) =>
        parseDasPublicKey(
          `getAssetProof.proof[${index}]`,
          value,
        ),
    );

  const merkleTreeAccount =
    await fetchMerkleTree(
      umi,
      merkleTree,
    );

  const canopyDepth =
    Math.log2(
      merkleTreeAccount.canopy.length + 2,
    ) - 1;

  if (
    !Number.isInteger(canopyDepth) ||
    canopyDepth < 0 ||
    canopyDepth > fullProof.length
  ) {
    throw new Error(
      [
        "transfer: invalid merkle tree canopy depth",
        `tree=${rpcProof.treeId}`,
        `canopyDepth=${canopyDepth}`,
        `proofLength=${fullProof.length}`,
      ].join(" "),
    );
  }

  const proof =
    canopyDepth === 0
      ? fullProof
      : fullProof.slice(
          0,
          -canopyDepth,
        );

  const leafIndex =
    rpcProof.nodeIndex -
    2 ** rpcProof.proof.length;

  if (
    !Number.isSafeInteger(leafIndex) ||
    leafIndex < 0
  ) {
    throw new Error(
      [
        "transfer: invalid leaf index",
        `nodeIndex=${rpcProof.nodeIndex}`,
        `proofLength=${rpcProof.proof.length}`,
        `index=${leafIndex}`,
      ].join(" "),
    );
  }

  return {
    leafOwner: parseDasPublicKey(
      "getAsset.ownership.owner",
      asset.ownership.owner,
    ),
    leafDelegate: parseDasPublicKey(
      "getAsset.ownership.delegate",
      asset.ownership.delegate,
    ),
    merkleTree,
    root: publicKeyBytes(
      parseDasPublicKey(
        "getAssetProof.root",
        rpcProof.root,
      ),
    ),
    dataHash: publicKeyBytes(
      parseDasPublicKey(
        "getAsset.compression.data_hash",
        asset.compression.dataHash,
      ),
    ),
    creatorHash: publicKeyBytes(
      parseDasPublicKey(
        "getAsset.compression.creator_hash",
        asset.compression.creatorHash,
      ),
    ),
    assetDataHash:
      asset.compression.assetDataHash === undefined
        ? undefined
        : publicKeyBytes(
            parseDasPublicKey(
              "getAsset.compression.asset_data_hash",
              asset.compression.assetDataHash,
            ),
          ),
    flags: asset.compression.flags,
    nonce: asset.compression.leafId,
    index: leafIndex,
    proof,
    asset,
  };
}

function resolveCoreCollection(
  asset: DasTransferAsset,
): PublicKey | null {
  const collectionGroup =
    asset.grouping.find(
      (group) =>
        group.groupKey === "collection",
    );

  if (!collectionGroup) {
    return null;
  }

  return parseDasPublicKey(
    "getAsset.grouping.collection",
    collectionGroup.groupValue,
  );
}

async function executeBubblegumTransfer(
  input: TransferExecutionInput,
): Promise<TransferExecutionResult> {
  const runtime =
    await getBubblegumRuntime();

  parseSolanaPublicKey(
    "assetId",
    input.assetId,
  );

  const fromWalletPublicKey =
    parseSolanaPublicKey(
      "fromWalletAddress",
      input.fromWalletAddress,
    );

  const toWalletPublicKey =
    parseSolanaPublicKey(
      "toWalletAddress",
      input.toWalletAddress,
    );

  const canonicalFromWalletAddress =
    String(fromWalletPublicKey);

  const senderSigner =
    await loadSenderSigner(
      runtime.umi,
      {
        fromAvatarId:
          input.fromAvatarId,
        fromBrandId:
          input.fromBrandId,
        fromWalletAddress:
          canonicalFromWalletAddress,
      },
    );

  const assetWithProof =
    await fetchTransferAssetWithProof(
      runtime.umi,
      input.assetId,
    );

  const currentOwner =
    String(
      assetWithProof.leafOwner,
    );

  if (
    currentOwner !==
    canonicalFromWalletAddress
  ) {
    throw new TransferOwnershipConflictError(
      canonicalFromWalletAddress,
      currentOwner,
    );
  }

  const coreCollection =
    resolveCoreCollection(
      assetWithProof.asset,
    );

  console.log(
    [
      "[transfer]",
      "start",
      `productId=${input.productId}`,
      `assetId=${input.assetId}`,
      `fromAvatarId=${input.fromAvatarId}`,
      `fromBrandId=${input.fromBrandId}`,
      `toAvatarId=${input.toAvatarId}`,
      `fromWallet=${canonicalFromWalletAddress}`,
      `toWallet=${String(toWalletPublicKey)}`,
    ].join(" "),
  );

  const transactionResult =
    await transferV2(
      runtime.umi,
      {
        payer: runtime.feePayer,
        authority: senderSigner,
        leafOwner:
          assetWithProof.leafOwner,
        leafDelegate:
          assetWithProof.leafDelegate,
        newLeafOwner:
          toWalletPublicKey,
        merkleTree:
          assetWithProof.merkleTree,
        root:
          assetWithProof.root,
        dataHash:
          assetWithProof.dataHash,
        creatorHash:
          assetWithProof.creatorHash,

        ...(
          assetWithProof.assetDataHash ===
          undefined
            ? {}
            : {
                assetDataHash:
                  assetWithProof.assetDataHash,
              }
        ),

        ...(
          assetWithProof.flags ===
          undefined
            ? {}
            : {
                flags:
                  assetWithProof.flags,
              }
        ),

        nonce:
          assetWithProof.nonce,
        index:
          assetWithProof.index,
        proof:
          assetWithProof.proof,

        ...(
          coreCollection === null
            ? {}
            : {
                coreCollection,
              }
        ),
      },
    ).sendAndConfirm(
      runtime.umi,
      {
        confirm: {
          commitment: "finalized",
        },
      },
    );

  const signature =
    base58.deserialize(
      transactionResult.signature,
    )[0];

  if (!signature) {
    throw new Error(
      "transfer: transaction signature is empty",
    );
  }

  console.log(
    [
      "[transfer]",
      "succeeded",
      `productId=${input.productId}`,
      `assetId=${input.assetId}`,
      `signature=${signature}`,
    ].join(" "),
  );

  return {
    signature,
    assetId: input.assetId,
  };
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
  (
    _req: Request,
    res: Response,
  ) => {
    res.status(200).json({
      status: "ok",
      service: "solana-bubblegum",
    });
  },
);

app.post(
  "/owned-assets",
  async (
    req: Request,
    res: Response,
    next: NextFunction,
  ) => {
    try {
      const body =
        readOwnedAssetsRequestBody(
          req.body,
        );

      const assetStandard =
        requiredString(
          "assetStandard",
          body.assetStandard,
        );

      const walletAddress =
        requiredString(
          "walletAddress",
          body.walletAddress,
        );

      if (
        assetStandard !==
        "BUBBLEGUM_V2"
      ) {
        throw new HttpRequestValidationError(
          "assetStandard",
          "only BUBBLEGUM_V2 is supported",
        );
      }

      parseSolanaPublicKey(
        "walletAddress",
        walletAddress,
      );

      let assetIds: string[];

      try {
        assetIds =
          await fetchOwnedBubblegumAssetIDs(
            walletAddress,
          );
      } catch (error) {
        throw new OwnedAssetsExecutionError(
          error,
        );
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
  "/transfer",
  async (
    req: Request,
    res: Response,
    next: NextFunction,
  ) => {
    try {
      const body =
        readTransferRequestBody(
          req.body,
        );

      const productId =
        requiredString(
          "productId",
          body.productId,
        );

      const assetStandard =
        requiredString(
          "assetStandard",
          body.assetStandard,
        );

      if (
        assetStandard !==
        "BUBBLEGUM_V2"
      ) {
        throw new HttpRequestValidationError(
          "assetStandard",
          "only BUBBLEGUM_V2 is supported",
        );
      }

      const assetId =
        requiredString(
          "assetId",
          body.assetId,
        );

      const fromAvatarId =
        optionalString(
          "fromAvatarId",
          body.fromAvatarId,
        );

      const fromBrandId =
        optionalString(
          "fromBrandId",
          body.fromBrandId,
        );

      const toAvatarId =
        requiredString(
          "toAvatarId",
          body.toAvatarId,
        );

      const fromWalletAddress =
        requiredString(
          "fromWalletAddress",
          body.fromWalletAddress,
        );

      const toWalletAddress =
        requiredString(
          "toWalletAddress",
          body.toWalletAddress,
        );

      if (
        Boolean(fromAvatarId) ===
        Boolean(fromBrandId)
      ) {
        throw new HttpRequestValidationError(
          "sender",
          "exactly one of fromAvatarId or fromBrandId is required",
        );
      }

      parseSolanaPublicKey(
        "assetId",
        assetId,
      );

      parseSolanaPublicKey(
        "fromWalletAddress",
        fromWalletAddress,
      );

      parseSolanaPublicKey(
        "toWalletAddress",
        toWalletAddress,
      );

      let result:
        TransferExecutionResult;

      try {
        result =
          await executeBubblegumTransfer({
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
          error instanceof
            HttpRequestValidationError ||
          error instanceof
            TransferOwnershipConflictError ||
          error instanceof
            TransferSignerMismatchError
        ) {
          throw error;
        }

        throw new TransferExecutionError(
          error,
        );
      }

      res.status(200).json({
        signature:
          result.signature,
        assetId:
          result.assetId,
      });
    } catch (error) {
      next(error);
    }
  },
);

app.post(
  "/estimate",
  async (
    req: Request,
    res: Response,
    next: NextFunction,
  ) => {
    try {
      const body =
        readMintEstimateRequestBody(
          req.body,
        );

      const tokenBlueprintId =
        requiredString(
          "tokenBlueprintId",
          body.tokenBlueprintId,
        );

      const mintQuantity =
        requiredPositiveInteger(
          "mintQuantity",
          body.mintQuantity,
        );

      const toAddress =
        requiredString(
          "toAddress",
          body.toAddress,
        );

      const name =
        requiredString(
          "name",
          body.name,
        );

      const symbol =
        stringValue(
          "symbol",
          body.symbol,
        );

      let result;

      try {
        const runtime =
          await getBubblegumRuntime();

        const mintFundingEstimateUsecase =
          getMintFundingEstimateUsecase();

        result =
          await mintFundingEstimateUsecase.execute({
            tokenBlueprintId,
            mintQuantity,
            leafOwnerAddress:
              toAddress,
            name,
            symbol,
            umi:
              runtime.umi,
            feePayer:
              runtime.feePayer,
            reserve:
              runtime.reserve,
          });
      } catch (error) {
        if (
          isMintV2TransactionError(
            error,
          )
        ) {
          throw error;
        }

        throw new MintEstimateExecutionError(
          error,
        );
      }

      res.status(200).json(
        result,
      );
    } catch (error) {
      next(error);
    }
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
      const body =
        readMintRequestBody(
          req.body,
        );

      const productId =
        requiredString(
          "productId",
          body.productId,
        );

      const idempotencyKey =
        req.get(
          "Idempotency-Key",
        );

      if (!idempotencyKey) {
        throw new HttpRequestValidationError(
          "Idempotency-Key",
          "header is required",
        );
      }

      if (
        idempotencyKey !==
        productId
      ) {
        throw new HttpRequestValidationError(
          "Idempotency-Key",
          "header must equal productId",
        );
      }

      const tokenBlueprintId =
        requiredString(
          "tokenBlueprintId",
          body.tokenBlueprintId,
        );

      const brandId =
        requiredString(
          "brandId",
          body.brandId,
        );

      const toAddress =
        requiredString(
          "toAddress",
          body.toAddress,
        );

      const name =
        requiredString(
          "name",
          body.name,
        );

      const symbol =
        stringValue(
          "symbol",
          body.symbol,
        );

      const metadataUri =
        requiredString(
          "metadataUri",
          body.metadataUri,
        );

      const [
        runtime,
        mintV2Usecase,
      ] =
        await Promise.all([
          getBubblegumRuntime(),
          getMintV2Usecase(),
        ]);

      const result =
        await mintV2Usecase.execute({
          productId,
          tokenBlueprintId,
          brandId,
          leafOwnerAddress:
            toAddress,
          leafDelegateAddress:
            null,
          coreCollection: {
            name,
            metadataUri,
          },
          metadata: {
            name,
            symbol,
            uri: metadataUri,
            sellerFeeBasisPoints: 0,
            primarySaleHappened:
              false,
            isMutable: false,
            creators: [],
          },
          umi:
            runtime.umi,
          feePayer:
            runtime.feePayer,
          reserve:
            runtime.reserve,
        });

      res.status(200).json(
        result,
      );
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
    console.error(
      "[http]",
      error,
    );

    if (
      error instanceof
      SyntaxError
    ) {
      res.status(400).json({
        error:
          "invalid JSON body",
        message:
          error.message,
      });
      return;
    }

    if (
      error instanceof
      HttpRequestValidationError
    ) {
      res.status(400).json({
        error:
          "invalid request",
        field:
          error.field,
        message:
          error.message,
      });
      return;
    }

    if (
      error instanceof
      TransferOwnershipConflictError
    ) {
      res.status(409).json({
        error:
          "transfer ownership conflict",
        message:
          error.message,
        expectedOwner:
          error.expectedOwner,
        actualOwner:
          error.actualOwner,
      });
      return;
    }

    if (
      error instanceof
      TransferSignerMismatchError
    ) {
      res.status(409).json({
        error:
          "transfer signer mismatch",
        message:
          error.message,
        expectedAddress:
          error.expectedAddress,
        signerAddress:
          error.signerAddress,
      });
      return;
    }

    if (
      error instanceof
      TransferExecutionError
    ) {
      res.status(503).json({
        error:
          "transfer unavailable",
        message:
          error.message,
      });
      return;
    }

    if (
      error instanceof
      OwnedAssetsExecutionError
    ) {
      res.status(503).json({
        error:
          "owned assets unavailable",
        message:
          error.message,
      });
      return;
    }

    if (
      error instanceof
      MintV2UsecaseValidationError
    ) {
      res.status(400).json({
        error:
          "invalid mint request",
        field:
          error.field,
        message:
          error.message,
      });
      return;
    }

    if (
      isMintV2TransactionError(
        error,
      )
    ) {
      if (
        error.code ===
          "INVALID_INPUT" ||
        error.code ===
          "INVALID_PUBLIC_KEY" ||
        error.code ===
          "INVALID_SIGNATURE" ||
        error.code ===
          "INVALID_TRANSACTION_SIGNATURE"
      ) {
        res.status(400).json({
          error:
            "invalid mint transaction request",
          code:
            error.code,
          message:
            error.message,
        });
        return;
      }

      if (
        error.kind ===
        "FATAL"
      ) {
        res.status(422).json({
          error:
            "mint transaction failed fatally",
          code:
            error.code,
          message:
            error.message,
        });
        return;
      }

      res.status(503).json({
        error:
          "mint transaction failed retryably",
        code:
          error.code,
        message:
          error.message,
      });
      return;
    }

    if (
      error instanceof
      MintEstimateExecutionError
    ) {
      res.status(503).json({
        error:
          "mint funding estimate unavailable",
        message:
          error.message,
      });
      return;
    }

    if (
      error instanceof
      MintOperationPayloadConflictError
    ) {
      res.status(409).json({
        error:
          "idempotency conflict",
        productId:
          error.productId,
      });
      return;
    }

    if (
      error instanceof
        MintOperationStateConflictError ||
      error instanceof
        MintOperationSignedTransactionConflictError
    ) {
      res.status(409).json({
        error:
          "mint operation conflict",
        productId:
          error.productId,
      });
      return;
    }

    if (
      error instanceof
      MintV2UsecaseInvalidStateError
    ) {
      res.status(409).json({
        error:
          "invalid mint operation state",
        productId:
          error.productId,
        status:
          error.status,
      });
      return;
    }

    if (
      error instanceof
      MintV2UsecaseStoredFatalError
    ) {
      res.status(422).json({
        error:
          "mint operation failed fatally",
        productId:
          error.productId,
        errorCode:
          error.errorCode,
      });
      return;
    }

    if (
      error instanceof
      MintOperationNotFoundError
    ) {
      res.status(404).json({
        error:
          "mint operation not found",
        productId:
          error.productId,
      });
      return;
    }

    res.status(500).json({
      error:
        "internal server error",
      message:
        error instanceof Error
          ? error.message
          : String(error),
    });
  },
);