// services/solana-bubblegum/src/infrastructure/solana/bubblegum-runtime.ts

import {
  signerIdentity,
  signerPayer,
  type KeypairSigner,
  type Umi,
} from "@metaplex-foundation/umi";

import {
  SecretManagerSignerLoader,
} from "../secret-manager/signer-loader.js";

import {
  createSolanaUmi,
} from "./umi-client.js";


export type BubblegumRuntime = {
  umi: Umi;

  feePayer:
    KeypairSigner;

  reserve:
    KeypairSigner;

  mintAuthority:
    KeypairSigner;
};


export type CreateBubblegumRuntimeInput = {
  rpcURL: string;
  googleCloudProject: string;
};


export async function createBubblegumRuntime(
  input: CreateBubblegumRuntimeInput,
): Promise<BubblegumRuntime> {
  const umi =
    createSolanaUmi(
      input.rpcURL,
    );


  const signerLoader =
    new SecretManagerSignerLoader(
      input.googleCloudProject,
    );


  const [
    feePayer,
    reserve,
    mintAuthority,
  ] =
    await Promise.all([
      signerLoader.loadSigner(
        umi,
        "feePayer",
      ),

      signerLoader.loadSigner(
        umi,
        "reserve",
      ),

      signerLoader.loadSigner(
        umi,
        "mintAuthority",
      ),
    ]);


  umi.use(
    signerIdentity(
      mintAuthority,
      false,
    ),
  );


  umi.use(
    signerPayer(
      feePayer,
    ),
  );


  return {
    umi,

    feePayer,

    reserve,

    mintAuthority,
  };
}