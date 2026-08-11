// services/solana-bubblegum/src/infrastructure/secret-manager/signer-loader.ts

import {
  SecretManagerServiceClient,
} from "@google-cloud/secret-manager";

import {
  createSignerFromKeypair,
  type KeypairSigner,
  type Umi,
} from "@metaplex-foundation/umi";


export type LogicalSigner =
  | "feePayer"
  | "reserve"
  | "mintAuthority";


const secretIDByLogicalSigner: Record<
  LogicalSigner,
  string
> = {
  feePayer:
    "bubblegum-fee-payer-keypair",

  reserve:
    "bubblegum-reserve-keypair",

  mintAuthority:
    "bubblegum-mint-authority-keypair",
};


function parseSecretKey(
  secretID: string,
  raw: string,
): Uint8Array {
  let parsed: unknown;


  try {
    parsed =
      JSON.parse(raw);
  } catch {
    throw new Error(
      `secret_manager: invalid JSON secret=${secretID}`,
    );
  }


  if (
    !Array.isArray(parsed) ||
    parsed.length !== 64
  ) {
    throw new Error(
      `secret_manager: invalid Solana keypair secret=${secretID}`,
    );
  }


  const bytes: number[] =
    [];


  for (
    const value of parsed
  ) {
    if (
      typeof value !== "number" ||
      !Number.isInteger(value) ||
      value < 0 ||
      value > 255
    ) {
      throw new Error(
        `secret_manager: invalid Solana keypair byte secret=${secretID}`,
      );
    }


    bytes.push(
      value,
    );
  }


  return Uint8Array.from(
    bytes,
  );
}


export class SecretManagerSignerLoader {
  private readonly client:
    SecretManagerServiceClient;


  constructor(
    private readonly projectID: string,
    client?: SecretManagerServiceClient,
  ) {
    this.client =
      client ??
      new SecretManagerServiceClient();
  }


  private async loadSecretKey(
    logicalSigner: LogicalSigner,
  ): Promise<Uint8Array> {
    const secretID =
      secretIDByLogicalSigner[
        logicalSigner
      ];


    const name =
      `projects/${this.projectID}/secrets/${secretID}/versions/latest`;


    const [version] =
      await this.client.accessSecretVersion({
        name,
      });


    const data =
      version.payload?.data;


    if (!data) {
      throw new Error(
        `secret_manager: payload is empty secret=${secretID}`,
      );
    }


    const raw =
      Buffer.from(
        data,
      ).toString(
        "utf8",
      );


    return parseSecretKey(
      secretID,
      raw,
    );
  }


  async loadSigner(
    umi: Umi,
    logicalSigner: LogicalSigner,
  ): Promise<KeypairSigner> {
    const secretKey =
      await this.loadSecretKey(
        logicalSigner,
      );


    const keypair =
      umi.eddsa.createKeypairFromSecretKey(
        secretKey,
      );


    return createSignerFromKeypair(
      umi,
      keypair,
    );
  }
}