// frontend/amol/src/features/wallet/api/walletTokenApi.ts

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
  unwrapData,
} from "../utils/tokenGuards";

type WalletTokenApiInput = {
  backendUrl: string;
  idToken: string;
};

type FetchWalletByAvatarIdInput = WalletTokenApiInput & {
  avatarId: string;
};

type ResolveTokenInput = WalletTokenApiInput & {
  mintAddress: string;
};

type FetchTokenMetadataInput = WalletTokenApiInput & {
  metadataUri: string;
};

function normalizeBackendUrl(backendUrl: string): string {
  return backendUrl.replace(/\/+$/, "");
}

function buildAuthHeaders(idToken: string): HeadersInit {
  return {
    Accept: "application/json",
    Authorization: `Bearer ${idToken}`,
  };
}

function createEmptyWalletTokenItem(mintAddress: string): WalletTokenItem {
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
  metadata: TokenMetadataDTO | null
): WalletTokenItem {
  const tokenBlueprintId =
    resolved.tokenBlueprintId || metadata?.tokenBlueprintId || "";

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

async function readJsonObject(response: Response): Promise<unknown> {
  const contentType = response.headers.get("content-type") || "";

  if (!contentType.includes("application/json")) {
    const body = await response.text().catch(() => "");
    throw new Error(
      body
        ? `APIがJSON以外を返しました: ${body}`
        : "APIがJSON以外を返しました。"
    );
  }

  return response.json();
}

export async function fetchMeWallet({
  backendUrl,
  idToken,
}: WalletTokenApiInput): Promise<WalletDTO | null> {
  const baseUrl = normalizeBackendUrl(backendUrl);

  const response = await fetch(`${baseUrl}/mall/me/wallets`, {
    method: "GET",
    headers: buildAuthHeaders(idToken),
  });

  if (response.status === 404) {
    return null;
  }

  if (!response.ok) {
    return null;
  }

  const body = await readJsonObject(response);
  const decoded = unwrapData(body);

  return extractWallet(decoded);
}

export async function syncMeWallet({
  backendUrl,
  idToken,
}: WalletTokenApiInput): Promise<void> {
  const baseUrl = normalizeBackendUrl(backendUrl);

  const response = await fetch(`${baseUrl}/mall/me/wallets/sync`, {
    method: "POST",
    headers: {
      ...buildAuthHeaders(idToken),
      "Content-Type": "application/json; charset=utf-8",
    },
    body: JSON.stringify({}),
  });

  if (!response.ok) {
    const body = await response.text().catch(() => "");
    throw new Error(`wallet sync failed: ${response.status} ${body}`);
  }
}

export async function syncAndFetchMeWallet({
  backendUrl,
  idToken,
}: WalletTokenApiInput): Promise<WalletDTO | null> {
  try {
    await syncMeWallet({ backendUrl, idToken });
  } catch {
    // sync 失敗時も fetch は試す
  }

  return fetchMeWallet({ backendUrl, idToken });
}

export async function fetchWalletByAvatarId({
  backendUrl,
  idToken,
  avatarId,
}: FetchWalletByAvatarIdInput): Promise<WalletDTO | null> {
  const normalizedAvatarId = avatarId.trim();

  if (!normalizedAvatarId) {
    return null;
  }

  const baseUrl = normalizeBackendUrl(backendUrl);
  const url = new URL(`${baseUrl}/mall/wallets`);
  url.searchParams.set("avatarId", normalizedAvatarId);

  const response = await fetch(url.toString(), {
    method: "GET",
    headers: buildAuthHeaders(idToken),
  });

  if (response.status === 404) {
    return null;
  }

  if (!response.ok) {
    const body = await response.text().catch(() => "");
    throw new Error(`wallet fetch failed: ${response.status} ${body}`);
  }

  const body = await readJsonObject(response);
  const decoded = unwrapData(body);

  return extractWallet(decoded);
}

export async function resolveTokenByMintAddress({
  backendUrl,
  idToken,
  mintAddress,
}: ResolveTokenInput): Promise<TokenResolveDTO | null> {
  const normalizedMintAddress = mintAddress.trim();

  if (!normalizedMintAddress) {
    return null;
  }

  const baseUrl = normalizeBackendUrl(backendUrl);
  const url = new URL(`${baseUrl}/mall/me/wallets/tokens/resolve`);
  url.searchParams.set("mintAddress", normalizedMintAddress);

  const response = await fetch(url.toString(), {
    method: "GET",
    headers: buildAuthHeaders(idToken),
  });

  if (response.status === 404) {
    return null;
  }

  if (!response.ok) {
    const body = await response.text().catch(() => "");
    throw new Error(`resolve failed: ${response.status} ${body}`);
  }

  const body = await readJsonObject(response);
  const decoded = unwrapData(body);

  return toTokenResolveDTO(decoded);
}

export async function fetchTokenMetadata({
  backendUrl,
  idToken,
  metadataUri,
}: FetchTokenMetadataInput): Promise<TokenMetadataDTO | null> {
  const normalizedMetadataUri = metadataUri.trim();

  if (!normalizedMetadataUri) {
    return null;
  }

  const baseUrl = normalizeBackendUrl(backendUrl);
  const url = new URL(`${baseUrl}/mall/me/wallets/metadata/proxy`);
  url.searchParams.set("url", normalizedMetadataUri);

  const response = await fetch(url.toString(), {
    method: "GET",
    headers: buildAuthHeaders(idToken),
  });

  if (response.status === 404) {
    return null;
  }

  if (!response.ok) {
    const body = await response.text().catch(() => "");
    throw new Error(`metadata fetch failed: ${response.status} ${body}`);
  }

  const body = await readJsonObject(response);

  return toTokenMetadataDTO(body);
}

export async function fetchOwnedProductBlueprintIdsByAvatarId({
  backendUrl,
  idToken,
  avatarId,
}: FetchWalletByAvatarIdInput): Promise<Set<string>> {
  const wallet = await fetchWalletByAvatarId({
    backendUrl,
    idToken,
    avatarId,
  });

  if (!wallet) {
    return new Set<string>();
  }

  const mints = wallet.tokens.map((token) => token.trim()).filter(Boolean);

  if (mints.length === 0) {
    return new Set<string>();
  }

  const out = new Set<string>();

  for (const mintAddress of mints) {
    try {
      const resolved = await resolveTokenByMintAddress({
        backendUrl,
        idToken,
        mintAddress,
      });

      const productBlueprintId = resolved?.productBlueprintId.trim() || "";

      if (productBlueprintId) {
        out.add(productBlueprintId);
      }
    } catch {
      // 個別 token resolve 失敗は握りつぶす
    }
  }

  return out;
}

export async function fetchMeWalletTokens({
  backendUrl,
  idToken,
}: WalletTokenApiInput): Promise<WalletTokenListResult> {
  const wallet = await fetchMeWallet({
    backendUrl,
    idToken,
  });

  if (!wallet) {
    return {
      wallet: null,
      tokens: [],
    };
  }

  const mints = wallet.tokens.map((token) => token.trim()).filter(Boolean);

  const tokens: WalletTokenItem[] = [];

  for (const mintAddress of mints) {
    try {
      const resolved = await resolveTokenByMintAddress({
        backendUrl,
        idToken,
        mintAddress,
      });

      if (!resolved) {
        tokens.push(createEmptyWalletTokenItem(mintAddress));
        continue;
      }

      let metadata: TokenMetadataDTO | null = null;

      if (resolved.metadataUri) {
        try {
          metadata = await fetchTokenMetadata({
            backendUrl,
            idToken,
            metadataUri: resolved.metadataUri,
          });
        } catch {
          metadata = null;
        }
      }

      tokens.push(createWalletTokenItem(mintAddress, resolved, metadata));
    } catch {
      tokens.push(createEmptyWalletTokenItem(mintAddress));
    }
  }

  return {
    wallet,
    tokens,
  };
}