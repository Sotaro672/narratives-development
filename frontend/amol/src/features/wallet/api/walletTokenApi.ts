// frontend/amol/src/features/wallet/api/walletTokenApi.ts

import { readJsonResponse } from "../../../components/utils/apiResponse";
import {
  buildApiUrl,
  getApiBaseUrl,
} from "../../../lib/apiBaseUrl";
import { getFirebaseIdToken } from "../../../lib/authToken";

import {
  fetchMeWalletRaw,
  resolveWalletTokenRaw,
} from "../../shared/api/walletApiClient";

import type {
  TokenMetadataDTO,
  TokenResolveDTO,
  WalletDTO,
  WalletTokenItem,
  WalletTokenListResult,
} from "../types/tokenTypes";
import {
  extractWallet,
  toTokenMetadataDTO,
  toTokenResolveDTO,
} from "../utils/tokenGuards";

type WalletApiContext = {
  baseUrl: string;
  idToken: string;
};

function buildAuthHeaders(idToken: string): HeadersInit {
  return {
    Accept: "application/json",
    Authorization: `Bearer ${idToken}`,
  };
}

async function createWalletApiContext(): Promise<WalletApiContext> {
  const baseUrl = getApiBaseUrl();

  if (!baseUrl) {
    throw new Error(
      "VITE_API_BASE_URLが設定されていません。",
    );
  }

  const idToken = await getFirebaseIdToken();

  return {
    baseUrl,
    idToken,
  };
}

function createEmptyWalletTokenItem(
  mintAddress: string,
): WalletTokenItem {
  return {
    mintAddress,
    productId: "",
    brandId: "",
    brandName: "",
    productName: "",
    productBlueprintId: "",
    tokenBlueprintId: "",
    metadataUri: "",
    metadata: null,
  };
}

function createWalletTokenItem(
  mintAddress: string,
  resolved: TokenResolveDTO,
  metadata: TokenMetadataDTO | null,
): WalletTokenItem {
  const tokenBlueprintId =
    resolved.tokenBlueprintId ||
    metadata?.tokenBlueprintId ||
    "";

  return {
    mintAddress,
    productId: resolved.productId,
    brandId: resolved.brandId,
    brandName: resolved.brandName,
    productName: resolved.productName,
    productBlueprintId: resolved.productBlueprintId,
    tokenBlueprintId,
    metadataUri: resolved.metadataUri,
    metadata,
  };
}

async function fetchMeWalletWithContext(
  context: WalletApiContext,
): Promise<WalletDTO | null> {
  const result = await fetchMeWalletRaw(
    buildAuthHeaders(context.idToken),
  );

  if (!result.ok) {
    if (result.status === 404) {
      return null;
    }

    throw new Error(
      "ウォレット情報の取得が許可されていません。",
    );
  }

  return extractWallet(result.data);
}

async function resolveTokenByMintAddressWithContext(
  context: WalletApiContext,
  mintAddress: string,
): Promise<TokenResolveDTO | null> {
  const normalizedMintAddress = mintAddress.trim();

  if (!normalizedMintAddress) {
    return null;
  }

  const result = await resolveWalletTokenRaw({
    mintAddress: normalizedMintAddress,
    headers: buildAuthHeaders(context.idToken),
  });

  if (!result.ok) {
    if (result.status === 404) {
      return null;
    }

    throw new Error(
      "トークン情報の取得が許可されていません。",
    );
  }

  return toTokenResolveDTO(result.data);
}

async function fetchTokenMetadataWithContext(
  context: WalletApiContext,
  metadataUri: string,
): Promise<TokenMetadataDTO | null> {
  if (!metadataUri) {
    return null;
  }

  const url = new URL(
    buildApiUrl(
      context.baseUrl,
      "/mall/me/wallets/metadata/proxy",
    ),
  );

  url.searchParams.set("url", metadataUri);

  const response = await fetch(url.toString(), {
    method: "GET",
    headers: buildAuthHeaders(context.idToken),
  });

  if (response.status === 404) {
    return null;
  }

  const body = await readJsonResponse<unknown>(
    response,
    {
      requestErrorMessage:
        "トークンメタデータの取得に失敗しました。",
      nonJsonErrorMessage:
        "トークンメタデータAPIがJSON以外を返しました。",
      invalidJsonErrorMessage:
        "トークンメタデータAPIのJSON形式が不正です。",
    },
  );

  return toTokenMetadataDTO(body);
}

export async function fetchMeWallet(): Promise<WalletDTO | null> {
  const context = await createWalletApiContext();

  return fetchMeWalletWithContext(context);
}

export async function resolveTokenByMintAddress(
  mintAddress: string,
): Promise<TokenResolveDTO | null> {
  const context = await createWalletApiContext();

  return resolveTokenByMintAddressWithContext(
    context,
    mintAddress,
  );
}

export async function fetchTokenMetadata(
  metadataUri: string,
): Promise<TokenMetadataDTO | null> {
  const context = await createWalletApiContext();

  return fetchTokenMetadataWithContext(
    context,
    metadataUri,
  );
}

export async function fetchMeWalletTokens(): Promise<WalletTokenListResult> {
  const context = await createWalletApiContext();
  const wallet = await fetchMeWalletWithContext(context);

  if (!wallet) {
    return {
      wallet: null,
      tokens: [],
    };
  }

  const mints = wallet.tokens
    .map((token) => token.trim())
    .filter(Boolean);

  const tokens: WalletTokenItem[] = [];

  for (const mintAddress of mints) {
    try {
      const resolved =
        await resolveTokenByMintAddressWithContext(
          context,
          mintAddress,
        );

      if (!resolved) {
        tokens.push(
          createEmptyWalletTokenItem(mintAddress),
        );

        continue;
      }

      let metadata: TokenMetadataDTO | null = null;

      if (resolved.metadataUri) {
        try {
          metadata =
            await fetchTokenMetadataWithContext(
              context,
              resolved.metadataUri,
            );
        } catch {
          metadata = null;
        }
      }

      tokens.push(
        createWalletTokenItem(
          mintAddress,
          resolved,
          metadata,
        ),
      );
    } catch {
      tokens.push(
        createEmptyWalletTokenItem(mintAddress),
      );
    }
  }

  return {
    wallet,
    tokens,
  };
}