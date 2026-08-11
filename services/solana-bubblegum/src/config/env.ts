//services\solana-bubblegum\src\config\env.ts
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
  solanaCluster:
    requiredEnv(
      "SOLANA_CLUSTER",
    ),

  devnetAirdropRpcURL:
    requiredEnv(
      "SOLANA_DEVNET_AIRDROP_RPC_URL",
    ),

  reservePublicKey:
    requiredEnv(
      "BUBBLEGUM_RESERVE_PUBLIC_KEY",
    ),

  devnetReserveTargetSOL:
    numberEnv(
      "BUBBLEGUM_DEVNET_RESERVE_TARGET_SOL",
      10,
    ),

  devnetAirdropSOL:
    numberEnv(
      "BUBBLEGUM_DEVNET_AIRDROP_SOL",
      5,
    ),
};