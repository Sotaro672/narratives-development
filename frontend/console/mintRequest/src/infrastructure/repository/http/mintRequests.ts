// frontend/console/mintRequest/src/infrastructure/repository/http/mintRequests.ts

import { API_BASE } from "../../http/consoleApiBase";
import { getIdTokenOrThrow } from "../../http/firebaseAuth";
import { buildHeaders } from "../../http/httpClient";
import {
  logHttpError,
  logHttpRequest,
  logHttpResponse,
  safeTokenHint,
} from "../../http/httpLogger";

import type {
  InspectionBatchDTO,
  MintDTO,
  MintListRowDTO,
} from "../../api/mintRequestApi";

import type {
  MintRequestRowRaw,
  MintRequestsPayloadRaw,
} from "../../dto/mintRequestRaw.dto";

import { normalizeMintDTO } from "../../normalizers/mint";
import { normalizeMintListRow } from "../../normalizers/mintListRow";
import {
  extractRowKeyAsProductionId,
  normalizeMintRequestsRows,
} from "../../normalizers/mintRequestsRows";

// ===============================
// internal: /mint/requests
// ===============================

async function fetchMintRequestsRowsRaw(
  ids: string[],
  view: "management" | "dto" | "list" | null,
): Promise<MintRequestRowRaw[]> {
  const idToken = await getIdTokenOrThrow();

  const base = `${API_BASE}/mint/requests?productionIds=${encodeURIComponent(
    ids.join(","),
  )}`;
  const url = view ? `${base}&view=${encodeURIComponent(view)}` : base;

  logHttpRequest("fetchMintRequestsRowsRaw", {
    method: "GET",
    url,
    headers: {
      Authorization: `Bearer ${safeTokenHint(idToken)}`,
      "Content-Type": "application/json",
    },
    productionIds: ids,
    view,
  });

  const res = await fetch(url, { method: "GET", headers: buildHeaders(idToken) });

  logHttpResponse("fetchMintRequestsRowsRaw", {
    method: "GET",
    url,
    status: res.status,
    statusText: res.statusText,
  });

  if (res.status === 404) return [];

  if (!res.ok) {
    const body = await res.text().catch(() => "");
    logHttpError("fetchMintRequestsRowsRaw", {
      method: "GET",
      url,
      status: res.status,
      statusText: res.statusText,
      bodyPreview: body ? body.slice(0, 800) : "",
    });
    throw new Error(
      `Failed to fetch mint requests: ${res.status} ${res.statusText}${
        body ? ` body=${body.slice(0, 400)}` : ""
      }`,
    );
  }

  const json = (await res.json()) as MintRequestsPayloadRaw | null | undefined;
  return normalizeMintRequestsRows(json);
}

// ===============================
// public exports
// ===============================

export async function fetchMintByInspectionIdHTTP(
  inspectionId: string,
): Promise<MintDTO | null> {
  const iid = String(inspectionId ?? "").trim();
  if (!iid) throw new Error("inspectionId が空です");

  try {
    const rows = await fetchMintRequestsRowsRaw([iid], "management");

    const row =
      (rows ?? []).find((r) => extractRowKeyAsProductionId(r) === iid) ??
      rows?.[0] ??
      null;

    if (!row) return null;

    const mintRaw = (row as any)?.mint ?? (row as any)?.Mint ?? null;
    if (mintRaw) return normalizeMintDTO(mintRaw);

    return normalizeMintDTO(row);
  } catch (_e: any) {
    return null;
  }
}

export async function fetchMintListRowsByInspectionIdsHTTP(
  inspectionIds: string[],
): Promise<Record<string, MintListRowDTO>> {
  const ids = (inspectionIds ?? [])
    .map((s) => String(s ?? "").trim())
    .filter((s) => !!s);

  if (ids.length === 0) return {};

  try {
    const rows = await fetchMintRequestsRowsRaw(ids, "management");

    const out: Record<string, MintListRowDTO> = {};
    for (const r of rows ?? []) {
      const key = extractRowKeyAsProductionId(r);
      if (!key) continue;

      const base =
        (r as any)?.mint ?? (r as any)?.Mint ?? null
          ? (r as any)?.mint ?? (r as any)?.Mint
          : (r as any);

      const merged = {
        ...(base ?? {}),
        inspectionId: key,
        productionId: key,
        tokenName: (r as any)?.tokenName ?? (base as any)?.tokenName ?? null,
        createdByName:
          (r as any)?.createdByName ?? (base as any)?.createdByName ?? null,
        mintedAt: (r as any)?.mintedAt ?? (base as any)?.mintedAt ?? null,
        minted:
          typeof (r as any)?.minted === "boolean"
            ? (r as any).minted
            : (base as any)?.minted,
      };

      out[key] = normalizeMintListRow(merged);
    }

    return out;
  } catch (_e: any) {
    return {};
  }
}

export async function fetchMintsByInspectionIdsHTTP(
  inspectionIds: string[],
): Promise<Record<string, MintDTO>> {
  const ids = (inspectionIds ?? [])
    .map((s) => String(s ?? "").trim())
    .filter((s) => !!s);

  if (ids.length === 0) return {};

  try {
    const rows = await fetchMintRequestsRowsRaw(ids, "management");

    const out: Record<string, MintDTO> = {};
    for (const r of rows ?? []) {
      const key = extractRowKeyAsProductionId(r);
      if (!key) continue;

      const mintRaw = (r as any)?.mint ?? (r as any)?.Mint ?? null;
      if (mintRaw) {
        out[key] = normalizeMintDTO(mintRaw);
        continue;
      }
      out[key] = normalizeMintDTO(r);
    }

    return out;
  } catch (_e: any) {
    return {};
  }
}

export async function listMintsByInspectionIDsHTTP(
  inspectionIds: string[],
): Promise<Record<string, MintListRowDTO>> {
  return await fetchMintListRowsByInspectionIdsHTTP(inspectionIds);
}

// ===============================
// POST: mint request
// ===============================

export async function postMintRequestHTTP(
  productionId: string,
  tokenBlueprintId: string,
  scheduledBurnDate?: string,
): Promise<InspectionBatchDTO | null> {
  const trimmed = String(productionId ?? "").trim();
  if (!trimmed) throw new Error("productionId が空です");

  const idToken = await getIdTokenOrThrow();

  const url = `${API_BASE}/mint/inspections/${encodeURIComponent(trimmed)}/request`;

  const payload: { tokenBlueprintId: string; scheduledBurnDate?: string } = {
    tokenBlueprintId: String(tokenBlueprintId ?? "").trim(),
  };

  if (scheduledBurnDate && String(scheduledBurnDate).trim()) {
    payload.scheduledBurnDate = String(scheduledBurnDate).trim();
  }

  logHttpRequest("postMintRequestHTTP", {
    method: "POST",
    url,
    headers: {
      Authorization: `Bearer ${safeTokenHint(idToken)}`,
      "Content-Type": "application/json",
    },
    productionId: trimmed,
    payload,
  });

  const res = await fetch(url, {
    method: "POST",
    headers: buildHeaders(idToken),
    body: JSON.stringify(payload),
  });

  logHttpResponse("postMintRequestHTTP", {
    method: "POST",
    url,
    status: res.status,
    statusText: res.statusText,
  });

  if (res.status === 404) return null;

  if (!res.ok) {
    const body = await res.text().catch(() => "");
    logHttpError("postMintRequestHTTP", {
      method: "POST",
      url,
      status: res.status,
      statusText: res.statusText,
      payload,
      bodyPreview: body ? body.slice(0, 1200) : "",
    });
    throw new Error(
      `Failed to post mint request: ${res.status} ${res.statusText}${
        body ? ` body=${body.slice(0, 400)}` : ""
      }`,
    );
  }

  const text = await res.text().catch(() => "");
  if (!text.trim()) return null;

  try {
    const json = JSON.parse(text) as InspectionBatchDTO | null | undefined;
    return json ?? null;
  } catch (e: any) {
    logHttpError("postMintRequestHTTP(parse)", {
      url,
      error: String(e?.message ?? e),
      bodyPreview: text.slice(0, 800),
    });
    return null;
  }
}
