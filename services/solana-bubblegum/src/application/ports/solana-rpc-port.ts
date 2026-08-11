//services\solana-bubblegum\src\application\ports\solana-rpc-port.ts
export interface SolanaRpcPort {
  getBalanceLamports(
    address: string,
  ): Promise<number>;

  requestAirdrop(
    address: string,
    lamports: number,
  ): Promise<string>;

  waitForConfirmation(
    signature: string,
  ): Promise<void>;
}