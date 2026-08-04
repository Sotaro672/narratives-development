// frontend/amol/src/features/shared/api/walletApiClient.ts

import {
  readJsonDataResponse,
} from "../../../lib/apiResponse";
import {
  buildApiUrl,
  getApiBaseUrl,
} from "../../../lib/apiBaseUrl";

type WalletApiUnavailableStatus = 403 | 404;

export type WalletApiTransportResult =
  | {
      ok: true;
      status: number;
      data: unknown;
    }
  | {
      ok: false;
      status: WalletApiUnavailableStatus;
      data: null;
    };

export type ResolveWalletTokenRawInput = {
  mintAddress: string;
  headers?: HeadersInit;
};

function getWalletApiBaseUrl(): string {
  const baseUrl = getApiBaseUrl();

  if (!baseUrl) {
    throw new Error(
      "VITE_API_BASE_URLが設定されていません。",
    );
  }

  return baseUrl;
}

function buildWalletApiHeaders(
  headers?: HeadersInit,
): Headers {
  const requestHeaders = new Headers(headers);

  if (!requestHeaders.has("Accept")) {
    requestHeaders.set(
      "Accept",
      "application/json",
    );
  }

  return requestHeaders;
}

function isUnavailableStatus(
  status: number,
): status is WalletApiUnavailableStatus {
  return status === 403 || status === 404;
}

async function readWalletApiResponse(
  response: Response,
): Promise<WalletApiTransportResult> {
  if (isUnavailableStatus(response.status)) {
    return {
      ok: false,
      status: response.status,
      data: null,
    };
  }

  const data =
    await readJsonDataResponse<unknown>(
      response,
      {
        requestErrorMessage:
          "ウォレットAPIの呼び出しに失敗しました。",
        nonJsonErrorMessage:
          "ウォレットAPIがJSON以外を返しました。",
        invalidJsonErrorMessage:
          "ウォレットAPIのJSON形式が不正です。",
      },
    );

  return {
    ok: true,
    status: response.status,
    data,
  };
}

export async function fetchMeWalletRaw(
  headers?: HeadersInit,
): Promise<WalletApiTransportResult> {
  const baseUrl = getWalletApiBaseUrl();

  const response = await fetch(
    buildApiUrl(
      baseUrl,
      "/mall/me/wallets",
    ),
    {
      method: "GET",
      headers:
        buildWalletApiHeaders(headers),
    },
  );

  return readWalletApiResponse(response);
}

export async function resolveWalletTokenRaw({
  mintAddress,
  headers,
}: ResolveWalletTokenRawInput): Promise<WalletApiTransportResult> {
  const normalizedMintAddress =
    mintAddress.trim();

  if (!normalizedMintAddress) {
    throw new Error(
      "mintAddressが指定されていません。",
    );
  }

  const baseUrl = getWalletApiBaseUrl();

  const url = new URL(
    buildApiUrl(
      baseUrl,
      "/mall/me/wallets/tokens/resolve",
    ),
  );

  url.searchParams.set(
    "mintAddress",
    normalizedMintAddress,
  );

  const response = await fetch(
    url.toString(),
    {
      method: "GET",
      headers:
        buildWalletApiHeaders(headers),
    },
  );

  return readWalletApiResponse(response);
}