// services/solana-bubblegum/src/config/env.ts

function requiredEnv(
  key: string,
): string {
  const value =
    process.env[key];

  if (!value) {
    throw new Error(
      `config: ${key} is required`,
    );
  }

  return value;
}

function numberEnv(
  key: string,
  fallback: number,
): number {
  const raw =
    process.env[key];

  if (!raw) {
    return fallback;
  }

  const value =
    Number(raw);

  if (
    !Number.isFinite(value) ||
    value < 0
  ) {
    throw new Error(
      `config: ${key} is invalid`,
    );
  }

  return value;
}

export const env = {
  googleCloudProject:
    requiredEnv(
      "GOOGLE_CLOUD_PROJECT",
    ),

  solanaCluster:
    requiredEnv(
      "SOLANA_CLUSTER",
    ),

  solanaRpcURL:
    requiredEnv(
      "SOLANA_RPC_URL",
    ),

  feePayerTargetSOL:
    numberEnv(
      "BUBBLEGUM_FEE_PAYER_TARGET_SOL",
      0.5,
    ),

  reserveMinimumSOL:
    numberEnv(
      "BUBBLEGUM_RESERVE_MINIMUM_SOL",
      1,
    ),
};