// frontend/console/sales/src/presentation/hook/useSalesDetail.tsx
import { useCallback, useEffect, useMemo, useState } from "react";
import { useLocation, useNavigate, useParams } from "react-router-dom";

import type { TokenBlueprint } from "../../../tokenBlueprint/src/domain/entity/tokenBlueprint";
import { safeDateTimeLabelJa } from "../../../shell/src/shared/util/dateJa";

import { fetchTokenBlueprintDetail } from "../../../tokenBlueprint/src/application/tokenBlueprintDetailService";

export type SalesOwnerVM = {
  avatarId: string;
  avatarName: string;
  avatarIconUrl: string;
  mintAddress: string;
  productName: string;
  followerCount: number;
  followingCount: number;
  postCount: number;
};

type SalesEntity = {
  tokenBlueprintId: string;
};

type UseSalesDetailVM = {
  sales: SalesEntity | null;
  title: string;
  assigneeId: string;
  assigneeName: string;
  minted: boolean;

  createdByName: string;
  createdAt: string;
  updatedByName: string;
  updatedAt: string;

  owners: SalesOwnerVM[];
};

type UseSalesDetailHandlers = {
  onBack: () => void;
};

export type UseSalesDetailResult = {
  vm: UseSalesDetailVM;
  handlers: UseSalesDetailHandlers;
};

type SalesDetailLocationOwner = {
  avatarId?: string;
  avatarName?: string;
  avatarIcon?: string;
  followerCount?: number;
  followingCount?: number;
  postCount?: number;
};

type SalesDetailLocationProductBlueprint = {
  productBlueprintId?: string;
  productName?: string;
};

type SalesDetailLocationState = {
  mintAddresses?: string[];
  owners?: SalesDetailLocationOwner[];
  productBlueprints?: SalesDetailLocationProductBlueprint[];
};

function uniqueStrings(values: unknown): string[] {
  if (!Array.isArray(values)) return [];

  const seen = new Set<string>();
  const result: string[] = [];

  for (const v of values) {
    const s = String(v ?? "").trim();
    if (!s) continue;
    if (seen.has(s)) continue;
    seen.add(s);
    result.push(s);
  }

  return result;
}

function toSafeNumber(value: unknown): number {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }

  const n = Number(value);
  if (!Number.isFinite(n)) {
    return 0;
  }

  return n;
}

function getFirstProductName(productBlueprintsValue: unknown): string {
  if (!Array.isArray(productBlueprintsValue) || productBlueprintsValue.length === 0) {
    return "";
  }

  for (const item of productBlueprintsValue) {
    if (!item || typeof item !== "object") continue;

    const productName = String(
      (item as SalesDetailLocationProductBlueprint).productName ?? "",
    ).trim();

    if (productName) {
      return productName;
    }
  }

  return "";
}

function toOwnersFromState(
  ownersValue: unknown,
  mintAddressesValue: unknown,
  productBlueprintsValue: unknown,
): SalesOwnerVM[] {
  const mintAddresses = uniqueStrings(mintAddressesValue);
  const productName = getFirstProductName(productBlueprintsValue);

  if (!Array.isArray(ownersValue) || ownersValue.length === 0) {
    return mintAddresses.map((mintAddress) => ({
      avatarId: "",
      avatarName: "",
      avatarIconUrl: "",
      mintAddress,
      productName,
      followerCount: 0,
      followingCount: 0,
      postCount: 0,
    }));
  }

  return ownersValue.map((owner, index) => {
    const item =
      owner && typeof owner === "object"
        ? (owner as SalesDetailLocationOwner)
        : {};

    return {
      avatarId: String(item.avatarId ?? "").trim(),
      avatarName: String(item.avatarName ?? "").trim(),
      avatarIconUrl: String(item.avatarIcon ?? "").trim(),
      mintAddress: String(mintAddresses[index] ?? "").trim(),
      productName,
      followerCount: toSafeNumber(item.followerCount),
      followingCount: toSafeNumber(item.followingCount),
      postCount: toSafeNumber(item.postCount),
    };
  });
}

export function useSalesDetail(): UseSalesDetailResult {
  const navigate = useNavigate();
  const location = useLocation();
  const { tokenBlueprintId } = useParams<{ tokenBlueprintId: string }>();

  const [blueprint, setBlueprint] = useState<TokenBlueprint | null>(null);

  const locationState = (location.state ?? {}) as SalesDetailLocationState;
  const mintAddressesFromState = useMemo(
    () => uniqueStrings(locationState?.mintAddresses),
    [locationState],
  );
  const ownersFromState = useMemo(() => {
    return Array.isArray(locationState?.owners) ? locationState.owners : [];
  }, [locationState]);
  const productBlueprintsFromState = useMemo(() => {
    return Array.isArray(locationState?.productBlueprints)
      ? locationState.productBlueprints
      : [];
  }, [locationState]);

  useEffect(() => {
    const id = String(tokenBlueprintId ?? "").trim();
    if (!id) return;

    let cancelled = false;

    (async () => {
      try {
        const tb = await fetchTokenBlueprintDetail(id);
        if (cancelled) return;

        setBlueprint(tb);
      } catch {
        if (!cancelled) {
          navigate("/sales", { replace: true });
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [tokenBlueprintId, navigate]);

  const sales = useMemo<SalesEntity | null>(() => {
    const id = String((blueprint as any)?.id ?? tokenBlueprintId ?? "").trim();
    if (!id) return null;
    return { tokenBlueprintId: id };
  }, [blueprint, tokenBlueprintId]);

  const minted = useMemo(() => {
    return Boolean((blueprint as any)?.minted);
  }, [blueprint]);

  const createdByName = useMemo(() => {
    const name = String((blueprint as any)?.createdByName ?? "").trim();
    if (name) return name;
    return String((blueprint as any)?.createdBy ?? "").trim();
  }, [blueprint]);

  const updatedByName = useMemo(() => {
    const name = String((blueprint as any)?.updatedByName ?? "").trim();
    if (name) return name;
    return String((blueprint as any)?.updatedBy ?? "").trim();
  }, [blueprint]);

  const createdAt = useMemo(() => {
    return safeDateTimeLabelJa((blueprint as any)?.createdAt, "");
  }, [blueprint]);

  const updatedAt = useMemo(() => {
    return safeDateTimeLabelJa((blueprint as any)?.updatedAt, "");
  }, [blueprint]);

  const owners = useMemo(() => {
    return toOwnersFromState(
      ownersFromState,
      mintAddressesFromState,
      productBlueprintsFromState,
    );
  }, [ownersFromState, mintAddressesFromState, productBlueprintsFromState]);

  const handleBack = useCallback(() => {
    navigate("/sales", { replace: true });
  }, [navigate]);

  const vm: UseSalesDetailVM = {
    sales,
    title: "営業",
    assigneeId: String((blueprint as any)?.assigneeId ?? "").trim(),
    assigneeName:
      String((blueprint as any)?.assigneeName ?? "").trim() ||
      String((blueprint as any)?.assigneeId ?? "").trim(),
    minted,
    createdByName,
    createdAt,
    updatedByName,
    updatedAt,
    owners,
  };

  const handlers: UseSalesDetailHandlers = {
    onBack: handleBack,
  };

  return { vm, handlers };
}

export default useSalesDetail;