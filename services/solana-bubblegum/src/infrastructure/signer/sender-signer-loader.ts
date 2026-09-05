// services/solana-bubblegum/src/infrastructure/signer/sender-signer-loader.ts

import { Buffer } from "node:buffer";
import { SecretManagerServiceClient } from "@google-cloud/secret-manager";
import {
  createSignerFromKeypair,
  type KeypairSigner,
  type Umi,
} from "@metaplex-foundation/umi";

import { env } from "../../config/env.js";
import {
  HttpRequestValidationError,
  TransferSignerMismatchError,
} from "../../http/errors.js";

const secretManagerClient = new SecretManagerServiceClient();

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
        `detail=${error instanceof Error ? error.message : String(error)}`,
      ].join(" "),
    );
  }

  if (!Array.isArray(parsed) || parsed.length !== 64) {
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

export async function loadSenderSigner(
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
        `detail=${error instanceof Error ? error.message : String(error)}`,
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

  const raw = Buffer.from(data).toString("utf8");
  const secretKey = parseSecretKey(secretID, raw);
  const keypair = umi.eddsa.createKeypairFromSecretKey(secretKey);
  const signer = createSignerFromKeypair(umi, keypair);
  const signerAddress = String(signer.publicKey);

  if (signerAddress !== input.fromWalletAddress) {
    throw new TransferSignerMismatchError(
      input.fromWalletAddress,
      signerAddress,
    );
  }

  return signer;
}