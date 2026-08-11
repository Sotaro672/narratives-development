// services/solana-bubblegum/src/infrastructure/solana/solana-rpc-client.ts

import type {
  SolanaRpcPort,
} from "../../application/ports/solana-rpc-port.js";


type JsonRpcResponse<T> = {
  jsonrpc: "2.0";
  id: number;
  result?: T;
  error?: {
    code: number;
    message: string;
  };
};


export class SolanaRpcRateLimitError
  extends Error {
  constructor(
    public readonly retryAfterSeconds?: number,
  ) {
    const retryAfterMessage =
      retryAfterSeconds === undefined
        ? ""
        : ` retryAfterSeconds=${retryAfterSeconds}`;


    super(
      `solana_rpc: HTTP 429${retryAfterMessage}`,
    );


    this.name =
      "SolanaRpcRateLimitError";
  }
}


export class SolanaRpcHttpError
  extends Error {
  constructor(
    public readonly status: number,
    public readonly statusText: string,
  ) {
    super(
      `solana_rpc: HTTP ${status}${statusText ? ` ${statusText}` : ""}`,
    );


    this.name =
      "SolanaRpcHttpError";
  }
}


export class SolanaRpcJsonRpcError
  extends Error {
  constructor(
    public readonly method: string,
    public readonly rpcCode: number,
    public readonly rpcMessage: string,
  ) {
    super(
      `solana_rpc: ${method} failed code=${rpcCode} message=${rpcMessage}`,
    );


    this.name =
      "SolanaRpcJsonRpcError";
  }
}


function parseRetryAfterSeconds(
  retryAfter: string | null,
): number | undefined {
  if (!retryAfter) {
    return undefined;
  }


  const seconds =
    Number(retryAfter);


  if (
    Number.isFinite(seconds) &&
    seconds >= 0
  ) {
    return Math.ceil(seconds);
  }


  const retryAt =
    Date.parse(retryAfter);


  if (
    Number.isNaN(retryAt)
  ) {
    return undefined;
  }


  return Math.max(
    0,
    Math.ceil(
      (retryAt - Date.now()) /
        1000,
    ),
  );
}


export class SolanaRpcClient
  implements SolanaRpcPort {
  constructor(
    private readonly rpcURL: string,
  ) {}


  private async rpc<T>(
    method: string,
    params: unknown[],
  ): Promise<T> {
    const response =
      await fetch(
        this.rpcURL,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            jsonrpc: "2.0",
            id: 1,
            method,
            params,
          }),
        },
      );


    if (
      response.status === 429
    ) {
      const retryAfterSeconds =
        parseRetryAfterSeconds(
          response.headers.get(
            "retry-after",
          ),
        );


      throw new SolanaRpcRateLimitError(
        retryAfterSeconds,
      );
    }


    if (!response.ok) {
      throw new SolanaRpcHttpError(
        response.status,
        response.statusText,
      );
    }


    const payload =
      await response.json() as JsonRpcResponse<T>;


    if (payload.error) {
      throw new SolanaRpcJsonRpcError(
        method,
        payload.error.code,
        payload.error.message,
      );
    }


    if (
      payload.result === undefined
    ) {
      throw new Error(
        `solana_rpc: ${method} returned no result`,
      );
    }


    return payload.result;
  }


  async getBalanceLamports(
    address: string,
  ): Promise<number> {
    const result =
      await this.rpc<{
        context: {
          slot: number;
        };
        value: number;
      }>(
        "getBalance",
        [
          address,
          {
            commitment: "confirmed",
          },
        ],
      );


    return result.value;
  }


  async requestAirdrop(
    address: string,
    lamports: number,
  ): Promise<string> {
    return this.rpc<string>(
      "requestAirdrop",
      [
        address,
        lamports,
        {
          commitment: "confirmed",
        },
      ],
    );
  }


  async waitForConfirmation(
    signature: string,
  ): Promise<void> {
    const deadline =
      Date.now() + 60_000;


    while (
      Date.now() < deadline
    ) {
      const result =
        await this.rpc<{
          context: {
            slot: number;
          };
          value: Array<{
            slot: number;
            confirmations: number | null;
            err: unknown;
            confirmationStatus:
              | "processed"
              | "confirmed"
              | "finalized"
              | null;
          } | null>;
        }>(
          "getSignatureStatuses",
          [
            [signature],
            {
              searchTransactionHistory: true,
            },
          ],
        );


      const status =
        result.value[0];


      if (status?.err) {
        throw new Error(
          `solana_rpc: transaction failed signature=${signature}`,
        );
      }


      if (
        status?.confirmationStatus ===
          "confirmed" ||
        status?.confirmationStatus ===
          "finalized"
      ) {
        return;
      }


      await new Promise<void>(
        (resolve) =>
          setTimeout(
            resolve,
            2000,
          ),
      );
    }


    throw new Error(
      `solana_rpc: confirmation timeout signature=${signature}`,
    );
  }
}