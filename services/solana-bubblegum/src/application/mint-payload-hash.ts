// services/solana-bubblegum/src/application/mint-payload-hash.ts

import {
  createHash,
} from "node:crypto";


export type MintPayloadHashValue =
  | null
  | boolean
  | number
  | string
  | MintPayloadHashValue[]
  | {
      [key: string]:
        MintPayloadHashValue;
    };


export type MintPayloadHashInput = {
  [key: string]:
    MintPayloadHashValue;
};


function canonicalizeNumber(
  value: number,
): string {
  if (
    !Number.isFinite(
      value,
    )
  ) {
    throw new Error(
      "mint_payload_hash: non-finite number is not supported",
    );
  }

  return JSON.stringify(
    value,
  );
}


function canonicalizeString(
  value: string,
): string {
  return JSON.stringify(
    value,
  );
}


function canonicalizeArray(
  value: MintPayloadHashValue[],
): string {
  return [
    "[",
    value
      .map(
        (item) =>
          canonicalizeValue(
            item,
          ),
      )
      .join(
        ",",
      ),
    "]",
  ].join(
    "",
  );
}


function canonicalizeObject(
  value: {
    [key: string]:
      MintPayloadHashValue;
  },
): string {
  const keys =
    Object.keys(
      value,
    )
      .sort();

  const entries =
    keys.map(
      (key) => {
        const serializedKey =
          canonicalizeString(
            key,
          );

        const serializedValue =
          canonicalizeValue(
            value[key],
          );

        return [
          serializedKey,
          ":",
          serializedValue,
        ].join(
          "",
        );
      },
    );

  return [
    "{",
    entries.join(
      ",",
    ),
    "}",
  ].join(
    "",
  );
}


function canonicalizeValue(
  value: MintPayloadHashValue,
): string {
  if (
    value ===
    null
  ) {
    return "null";
  }

  if (
    typeof value ===
    "string"
  ) {
    return canonicalizeString(
      value,
    );
  }

  if (
    typeof value ===
    "number"
  ) {
    return canonicalizeNumber(
      value,
    );
  }

  if (
    typeof value ===
    "boolean"
  ) {
    return value
      ? "true"
      : "false";
  }

  if (
    Array.isArray(
      value,
    )
  ) {
    return canonicalizeArray(
      value,
    );
  }

  return canonicalizeObject(
    value,
  );
}


export function canonicalizeMintPayload(
  input: MintPayloadHashInput,
): string {
  if (
    input ===
      null ||
    Array.isArray(
      input,
    ) ||
    typeof input !==
      "object"
  ) {
    throw new Error(
      "mint_payload_hash: input must be an object",
    );
  }

  return canonicalizeObject(
    input,
  );
}


export function createMintPayloadHash(
  input: MintPayloadHashInput,
): string {
  const canonicalPayload =
    canonicalizeMintPayload(
      input,
    );

  return createHash(
    "sha256",
  )
    .update(
      canonicalPayload,
      "utf8",
    )
    .digest(
      "hex",
    );
}