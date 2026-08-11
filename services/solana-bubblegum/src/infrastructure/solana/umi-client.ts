// services/solana-bubblegum/src/infrastructure/solana/umi-client.ts

import {
  mplBubblegum,
} from "@metaplex-foundation/mpl-bubblegum";

import {
  mplCore,
} from "@metaplex-foundation/mpl-core";

import type {
  Umi,
} from "@metaplex-foundation/umi";

import {
  createUmi,
} from "@metaplex-foundation/umi-bundle-defaults";


export function createSolanaUmi(
  rpcURL: string,
): Umi {
  if (!rpcURL) {
    throw new Error(
      "umi: rpcURL is required",
    );
  }


  return createUmi(
    rpcURL,
  )
    .use(
      mplBubblegum(),
    )
    .use(
      mplCore(),
    );
}