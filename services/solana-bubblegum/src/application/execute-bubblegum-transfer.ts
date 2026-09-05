// services/solana-bubblegum/src/application/execute-bubblegum-transfer.ts

import { transferV2 } from "@metaplex-foundation/mpl-bubblegum";
import { publicKey, type PublicKey } from "@metaplex-foundation/umi";
import { base58 } from "@metaplex-foundation/umi/serializers";

import { getBubblegumRuntime } from "../bootstrap/container.js";
import { TransferOwnershipConflictError } from "../http/errors.js";
import { parseSolanaPublicKey } from "../http/request-validation.js";
import { fetchTransferAssetWithProof } from "../infrastructure/das/das-client.js";
import type { DasTransferAsset } from "../infrastructure/das/das-types.js";
import { loadSenderSigner } from "../infrastructure/signer/sender-signer-loader.js";

export type TransferExecutionInput = {
  productId: string;
  assetId: string;
  fromAvatarId: string;
  fromBrandId: string;
  toAvatarId: string;
  fromWalletAddress: string;
  toWalletAddress: string;
};

export type TransferExecutionResult = {
  signature: string;
  assetId: string;
};

function parseCoreCollectionPublicKey(value: string): PublicKey {
  try {
    return publicKey(value);
  } catch (error) {
    throw new Error([
      "DAS getAsset.grouping.collection must be a valid Solana public key",
      `value=${value}`,
      `detail=${error instanceof Error ? error.message : String(error)}`,
    ].join(" "));
  }
}

function resolveCoreCollection(asset: DasTransferAsset): PublicKey | null {
  const collectionGroup = asset.grouping.find((group) => group.groupKey === "collection");
  if (!collectionGroup) {
    return null;
  }
  return parseCoreCollectionPublicKey(collectionGroup.groupValue);
}

export async function executeBubblegumTransfer(
  input: TransferExecutionInput,
): Promise<TransferExecutionResult> {
  const runtime = await getBubblegumRuntime();

  parseSolanaPublicKey("assetId", input.assetId);
  const fromWalletPublicKey = parseSolanaPublicKey("fromWalletAddress", input.fromWalletAddress);
  const toWalletPublicKey = parseSolanaPublicKey("toWalletAddress", input.toWalletAddress);
  const canonicalFromWalletAddress = String(fromWalletPublicKey);

  const senderSigner = await loadSenderSigner(runtime.umi, {
    fromAvatarId: input.fromAvatarId,
    fromBrandId: input.fromBrandId,
    fromWalletAddress: canonicalFromWalletAddress,
  });

  const assetWithProof = await fetchTransferAssetWithProof(runtime.umi, input.assetId);
  const currentOwner = String(assetWithProof.leafOwner);

  if (currentOwner !== canonicalFromWalletAddress) {
    throw new TransferOwnershipConflictError(canonicalFromWalletAddress, currentOwner);
  }

  const coreCollection = resolveCoreCollection(assetWithProof.asset);

  const transactionResult = await transferV2(runtime.umi, {
    payer: runtime.feePayer,
    authority: senderSigner,
    leafOwner: assetWithProof.leafOwner,
    leafDelegate: assetWithProof.leafDelegate,
    newLeafOwner: toWalletPublicKey,
    merkleTree: assetWithProof.merkleTree,
    root: assetWithProof.root,
    dataHash: assetWithProof.dataHash,
    creatorHash: assetWithProof.creatorHash,
    ...(assetWithProof.assetDataHash === undefined
      ? {}
      : { assetDataHash: assetWithProof.assetDataHash }),
    ...(assetWithProof.flags === undefined
      ? {}
      : { flags: assetWithProof.flags }),
    nonce: assetWithProof.nonce,
    index: assetWithProof.index,
    proof: assetWithProof.proof,
    ...(coreCollection === null ? {} : { coreCollection }),
  }).sendAndConfirm(runtime.umi, {
    confirm: {
      commitment: "finalized",
    },
  });

  const signature = base58.deserialize(transactionResult.signature)[0];

  if (!signature) {
    throw new Error("transfer: transaction signature is empty");
  }

  return {
    signature,
    assetId: input.assetId,
  };
}