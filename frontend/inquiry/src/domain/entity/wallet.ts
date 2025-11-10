// frontend/inquiry/src/domain/entity/wallet.ts
// ------------------------------------------------------
// Domain entity for Wallet
// Mirrors backend/internal/domain/wallet/entity.go
// and web shared wallet type semantics.
// ------------------------------------------------------

/**
 * WalletStatus
 * 'active' | 'inactive'
 */
export type WalletStatus = "active" | "inactive";

export const WALLET_STATUS_ACTIVE: WalletStatus = "active";
export const WALLET_STATUS_INACTIVE: WalletStatus = "inactive";

/**
 * Wallet domain model
 *
 * NOTE:
 * - This is the domain representation: dates are `Date`.
 * - The shared type (frontend/shell/src/shared/types/wallet.ts) will use `Date | string`.
 */
export interface Wallet {
  walletAddress: string;
  tokens: string[];
  lastUpdatedAt: Date;
  status: WalletStatus;
  createdAt: Date;
  updatedAt: Date;
}

/**
 * Base58-ish address format for Solana-like addresses.
 * (Same as backend base58Re: ^[1-9A-HJ-NP-Za-km-z]{32,44}$)
 */
const BASE58_REGEX = /^[1-9A-HJ-NP-Za-km-z]{32,44}$/;

/**
 * Validate wallet status.
 */
export function isValidWalletStatus(status: WalletStatus): boolean {
  return status === "active" || status === "inactive";
}

/**
 * Validate wallet address format.
 */
export function isValidWalletAddress(addr: string): boolean {
  return BASE58_REGEX.test(addr.trim());
}

/**
 * Validate mint address format.
 * (Same rule as wallet address)
 */
export function isValidMintAddress(mint: string): boolean {
  return BASE58_REGEX.test(mint.trim());
}

/**
 * Create basic Wallet.
 * Backend `New` equivalent:
 * - Status = 'active'
 * - createdAt / updatedAt / lastUpdatedAt = updatedAt(引数)
 */
export function createWallet(
  walletAddress: string,
  tokens: string[],
  updatedAt: Date
): Wallet {
  const utc = toUTC(updatedAt);

  const wallet: Wallet = {
    walletAddress: walletAddress.trim(),
    tokens: [],
    lastUpdatedAt: utc,
    status: WALLET_STATUS_ACTIVE,
    createdAt: utc,
    updatedAt: utc,
  };

  setTokensOrThrow(wallet, tokens);
  validateWalletOrThrow(wallet);

  return wallet;
}

/**
 * Full constructor.
 * Backend `NewFull` equivalent.
 */
export function createWalletFull(params: {
  walletAddress: string;
  tokens: string[];
  lastUpdatedAt: Date;
  createdAt: Date;
  updatedAt: Date;
  status: WalletStatus;
}): Wallet {
  const wallet: Wallet = {
    walletAddress: params.walletAddress.trim(),
    tokens: [],
    lastUpdatedAt: toUTC(params.lastUpdatedAt),
    status: params.status,
    createdAt: toUTC(params.createdAt),
    updatedAt: toUTC(params.updatedAt),
  };

  setTokensOrThrow(wallet, params.tokens);
  validateWalletOrThrow(wallet);

  return wallet;
}

/**
 * NewNow equivalent:
 * - createdAt/updatedAt/lastUpdatedAt = now
 */
export function createWalletNow(
  walletAddress: string,
  tokens: string[],
  status: WalletStatus
): Wallet {
  const now = new Date();
  return createWalletFull({
    walletAddress,
    tokens,
    lastUpdatedAt: now,
    createdAt: now,
    updatedAt: now,
    status,
  });
}

/**
 * NewFromStringTime equivalent:
 * - lastUpdatedAt from string
 * - createdAt/updatedAt = lastUpdatedAt
 * - status = 'active'
 */
export function createWalletFromLastUpdatedString(
  walletAddress: string,
  tokens: string[],
  lastUpdatedAt: string
): Wallet {
  const lut = parseTimeOrThrow(lastUpdatedAt, "wallet: invalid lastUpdatedAt");

  return createWallet(walletAddress, tokens, lut);
}

/**
 * NewFromStringTimes equivalent:
 * - all timestamps as string, explicit status.
 */
export function createWalletFromStrings(params: {
  walletAddress: string;
  tokens: string[];
  lastUpdatedAt: string;
  createdAt: string;
  updatedAt: string;
  status: WalletStatus | string;
}): Wallet {
  const lut = parseTimeOrThrow(
    params.lastUpdatedAt,
    "wallet: invalid lastUpdatedAt"
  );
  const ct = parseTimeOrThrow(
    params.createdAt,
    "wallet: invalid createdAt"
  );
  const ut = parseTimeOrThrow(
    params.updatedAt,
    "wallet: invalid updatedAt"
  );

  const status = params.status as WalletStatus;
  if (!isValidWalletStatus(status)) {
    throw new Error("wallet: invalid status");
  }

  return createWalletFull({
    walletAddress: params.walletAddress,
    tokens: params.tokens,
    lastUpdatedAt: lut,
    createdAt: ct,
    updatedAt: ut,
    status,
  });
}

/* =====================================================
 * Behavior helpers (immutable-ish utility style)
 * ===================================================== */

/**
 * AddToken:
 * - adds mint if not present
 * - updates updatedAt / lastUpdatedAt
 */
export function addToken(
  wallet: Wallet,
  mint: string,
  now: Date = new Date()
): Wallet {
  if (!isValidMintAddress(mint)) {
    throw new Error("wallet: invalid mintAddress");
  }

  if (wallet.tokens.includes(mint)) {
    return wallet;
  }

  const next = { ...wallet, tokens: [...wallet.tokens, mint] };
  touch(next, now);
  validateWalletOrThrow(next);
  return next;
}

/**
 * RemoveToken:
 * - removes mint if present
 * - updates updatedAt / lastUpdatedAt only if changed
 */
export function removeToken(
  wallet: Wallet,
  mint: string,
  now: Date = new Date()
): Wallet {
  if (!mint) return wallet;

  const filtered = wallet.tokens.filter((t) => t !== mint);
  if (filtered.length === wallet.tokens.length) {
    return wallet;
  }

  const next = { ...wallet, tokens: filtered };
  touch(next, now);
  validateWalletOrThrow(next);
  return next;
}

/**
 * ReplaceTokens:
 * - dedup + validate all
 * - updates updatedAt / lastUpdatedAt
 */
export function replaceTokens(
  wallet: Wallet,
  tokens: string[],
  now: Date = new Date()
): Wallet {
  const next: Wallet = { ...wallet, tokens: [] };
  setTokensOrThrow(next, tokens);
  touch(next, now);
  validateWalletOrThrow(next);
  return next;
}

/**
 * HasToken:
 * - check if mint is included
 */
export function hasToken(wallet: Wallet, mint: string): boolean {
  return wallet.tokens.includes(mint);
}

/**
 * SetStatus:
 * - validates status and updates updatedAt
 */
export function setWalletStatus(
  wallet: Wallet,
  status: WalletStatus,
  now: Date = new Date()
): Wallet {
  if (!isValidWalletStatus(status)) {
    throw new Error("wallet: invalid status");
  }
  const next: Wallet = {
    ...wallet,
    status,
    updatedAt: toUTC(now),
  };
  validateWalletOrThrow(next);
  return next;
}

/* =====================================================
 * Validation & internal helpers
 * ===================================================== */

export function validateWalletOrThrow(wallet: Wallet): void {
  if (!isValidWalletAddress(wallet.walletAddress)) {
    throw new Error("wallet: invalid walletAddress");
  }

  if (!wallet.createdAt || isNaN(wallet.createdAt.getTime())) {
    throw new Error("wallet: invalid createdAt");
  }
  if (!wallet.updatedAt || isNaN(wallet.updatedAt.getTime())) {
    throw new Error("wallet: invalid updatedAt");
  }
  if (!wallet.lastUpdatedAt || isNaN(wallet.lastUpdatedAt.getTime())) {
    throw new Error("wallet: invalid lastUpdatedAt");
  }
  if (wallet.updatedAt < wallet.createdAt) {
    throw new Error("wallet: invalid updatedAt (before createdAt)");
  }
  if (wallet.lastUpdatedAt < wallet.createdAt) {
    throw new Error("wallet: invalid lastUpdatedAt (before createdAt)");
  }
  if (!isValidWalletStatus(wallet.status)) {
    throw new Error("wallet: invalid status");
  }

  for (const t of wallet.tokens) {
    if (!isValidMintAddress(t)) {
      throw new Error("wallet: invalid mintAddress in tokens");
    }
  }
}

function setTokensOrThrow(wallet: Wallet, tokens: string[]): void {
  const deduped = dedupeTokens(tokens);
  for (const t of deduped) {
    if (!isValidMintAddress(t)) {
      throw new Error("wallet: invalid mintAddress");
    }
  }
  wallet.tokens = deduped;
}

function dedupeTokens(tokens: string[]): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const tRaw of tokens) {
    const t = tRaw.trim();
    if (!t) continue;
    if (seen.has(t)) continue;
    seen.add(t);
    out.push(t);
  }
  return out;
}

function touch(wallet: Wallet, now: Date): void {
  const utc = toUTC(now || new Date());
  wallet.lastUpdatedAt = utc;
  wallet.updatedAt = utc;
}

function toUTC(d: Date): Date {
  return new Date(d.toISOString());
}

function parseTimeOrThrow(s: string, msg: string): Date {
  const trimmed = s.trim();
  if (!trimmed) {
    throw new Error(msg);
  }
  const d = new Date(trimmed);
  if (Number.isNaN(d.getTime())) {
    throw new Error(`${msg}: cannot parse "${s}"`);
  }
  return toUTC(d);
}
