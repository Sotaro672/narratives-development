// frontend/amol/src/features/wallet/api/walletTokenApi.ts

import { readJsonResponse } from "../../../lib/apiResponse";
import { buildApiUrl, getApiBaseUrl } from "../../../lib/apiBaseUrl";
import { getFirebaseIdToken } from "../../../lib/authToken";

import { fetchMeWalletRaw, resolveWalletTokenRaw } from "../../shared/api";

import type {
  TokenMetadataDTO,
  TokenResolveDTO,
  WalletDTO,
  WalletTokenItem,
  WalletTokenListResult,
} from "../../shared/types/tokenTypes";

import { toTokenMetadataDTO } from "../utils/tokenGuards";

type WalletApiContext = {
  baseUrl: string;
  idToken: string;
};

type MeWalletsResponse = {
  wallets: WalletDTO[];
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
    throw new Error("VITE_API_BASE_URLが設定されていません。");
  }

  const idToken = await getFirebaseIdToken();

  return {
    baseUrl,
    idToken,
  };
}

function createWalletTokenItem(
  resolved: TokenResolveDTO,
  metadata: TokenMetadataDTO | null,
): WalletTokenItem {
  return {
    assetId: resolved.assetId,
    productId: resolved.productId,
    brandId: resolved.brandId,
    brandName: resolved.brandName,
    productName: resolved.productName,
    productBlueprintId: resolved.productBlueprintId,
    tokenBlueprintId: metadata?.tokenBlueprintId ?? "",
    metadataUri: resolved.metadataUri,
    metadata,
  };
}

async function fetchMeWalletWithContext(
  context: WalletApiContext,
): Promise<WalletDTO | null> {
  const result = await fetchMeWalletRaw(buildAuthHeaders(context.idToken));

  if (!result.ok) {
    if (result.status === 404) {
      return null;
    }

    throw new Error("ウォレット情報の取得が許可されていません。");
  }

  const response = result.data as MeWalletsResponse;

  return response.wallets[0] ?? null;
}

async function resolveTokenByAssetIdWithContext(
  context: WalletApiContext,
  assetId: string,
): Promise<TokenResolveDTO | null> {
  if (!assetId) {
    return null;
  }

  const result = await resolveWalletTokenRaw({
    assetId,
    headers: buildAuthHeaders(context.idToken),
  });

  if (!result.ok) {
    if (result.status === 404) {
      return null;
    }

    throw new Error("トークン情報の取得が許可されていません。");
  }

  return result.data as TokenResolveDTO;
}

async function fetchTokenMetadataWithContext(
  context: WalletApiContext,
  metadataUri: string,
): Promise<TokenMetadataDTO | null> {
  if (!metadataUri) {
    return null;
  }

  const url = new URL(
    buildApiUrl(context.baseUrl, "/mall/me/wallets/metadata/proxy"),
  );

  url.searchParams.set("url", metadataUri);

  const response = await fetch(url.toString(), {
    method: "GET",
    headers: buildAuthHeaders(context.idToken),
  });

  if (response.status === 404) {
    return null;
  }

  const body = await readJsonResponse<unknown>(response, {
    requestErrorMessage: "トークンメタデータの取得に失敗しました。",
    nonJsonErrorMessage: "トークンメタデータAPIがJSON以外を返しました。",
    invalidJsonErrorMessage: "トークンメタデータAPIのJSON形式が不正です。",
  });

  return toTokenMetadataDTO(body);
}

export async function fetchMeWallet(): Promise<WalletDTO | null> {
  const context = await createWalletApiContext();

  return fetchMeWalletWithContext(context);
}

export async function resolveTokenByAssetId(
  assetId: string,
): Promise<TokenResolveDTO | null> {
  const context = await createWalletApiContext();

  return resolveTokenByAssetIdWithContext(context, assetId);
}

export async function fetchTokenMetadata(
  metadataUri: string,
): Promise<TokenMetadataDTO | null> {
  const context = await createWalletApiContext();

  return fetchTokenMetadataWithContext(context, metadataUri);
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

  const tokens: WalletTokenItem[] = [];

  for (const assetId of wallet.assetIds) {
    const resolved = await resolveTokenByAssetIdWithContext(context, assetId);

    if (!resolved) {
      continue;
    }

    const metadata = resolved.metadataUri
      ? await fetchTokenMetadataWithContext(context, resolved.metadataUri)
      : null;

    tokens.push(createWalletTokenItem(resolved, metadata));
  }

  return {
    wallet,
    tokens,
  };
}