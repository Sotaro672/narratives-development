// services/solana-bubblegum/src/infrastructure/das/das-client.ts

import { isValidLeafSchemaV2Flags } from "@metaplex-foundation/mpl-bubblegum";
import { fetchMerkleTree } from "@metaplex-foundation/spl-account-compression";
import {
  publicKey,
  publicKeyBytes,
  type PublicKey,
  type Umi,
} from "@metaplex-foundation/umi";

import { env } from "../../config/env.js";

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

type DasTransferProof = {
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

function isRecord(
  value: unknown,
): value is Record<string, unknown> {
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
    throw new Error(`DAS ${field} must be a non-empty string`);
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
    throw new Error(`DAS ${field} must be a non-negative safe integer`);
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
        `detail=${error instanceof Error ? error.message : String(error)}`,
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
    throw new Error(`DAS ${method} returned HTTP ${response.status}`);
  }

  let payload: unknown;

  try {
    payload = await response.json();
  } catch (error) {
    throw new Error(
      `DAS ${method} returned invalid JSON: ${
        error instanceof Error ? error.message : String(error)
      }`,
    );
  }

  if (!isRecord(payload)) {
    throw new Error(`DAS ${method} returned invalid response`);
  }

  const rpcResponse = payload as DasJsonRpcResponse;

  if (rpcResponse.error !== undefined) {
    const message = readDasErrorMessage(rpcResponse.error);

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
    throw new Error(`DAS ${method} result is missing`);
  }

  return rpcResponse.result;
}

function parseTransferDasAsset(
  value: unknown,
  expectedAssetID: string,
): DasTransferAsset {
  const asset = requiredRecord(
    "getAsset.result",
    value,
  );

  const assetID = requiredDasString(
    "getAsset.id",
    asset.id,
  );

  if (assetID !== expectedAssetID) {
    throw new Error(
      `DAS getAsset returned unexpected asset id expected=${expectedAssetID} actual=${assetID}`,
    );
  }

  const compression = requiredRecord(
    "getAsset.compression",
    asset.compression,
  );

  if (compression.compressed !== true) {
    throw new Error(
      `DAS asset is not compressed assetId=${assetID}`,
    );
  }

  const ownership = requiredRecord(
    "getAsset.ownership",
    asset.ownership,
  );

  const owner = requiredDasString(
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

  const rawFlags = compression.flags;

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
  const result = requiredRecord(
    "getAssetProof.result",
    value,
  );

  if (!Array.isArray(result.proof)) {
    throw new Error(
      "DAS getAssetProof.proof must be an array",
    );
  }

  const proof = result.proof.map(
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

export async function fetchOwnedBubblegumAssetIDs(
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
          error instanceof Error ? error.message : String(error)
        }`,
      );
    }

    if (!isRecord(payload)) {
      throw new Error(
        "DAS getAssetsByOwner returned invalid response",
      );
    }

    const rpcResponse = payload as DasJsonRpcResponse;

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
      const asset = readDasAsset(value);

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

export async function fetchTransferAssetWithProof(
  umi: Umi,
  assetID: string,
): Promise<BubblegumTransferAssetWithProof> {
  const [
    assetValue,
    proofValue,
  ] = await Promise.all([
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