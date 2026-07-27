// frontend/console/shell/src/auth/presentation/hook/useCurrentMember.ts

/// <reference types="vite/client" />

import { useEffect, useMemo, useState } from "react";
import { useAuthContext } from "../../application/AuthContext";

// Application層のサービス
import {
  getCompanyNameByIdCached,
  clearCompanyNameCache,
} from "../../application/companyService";

import {
  getCurrentMember,
  type CurrentMemberResponse,
} from "../../application/authService";

// 共通Member型
import type { MemberDTO } from "../../../shared/types/member";

type UnknownRecord = Record<string, unknown>;

function toStringValue(value: unknown): string {
  return typeof value === "string"
    ? value.trim()
    : "";
}

function toNullableString(
  value: unknown,
): string | null {
  if (typeof value !== "string") {
    return null;
  }

  const normalized = value.trim();

  return normalized || null;
}

function toStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) {
    return [];
  }

  return value
    .filter(
      (item): item is string =>
        typeof item === "string",
    )
    .map((item) => item.trim())
    .filter((item) => item.length > 0);
}

function toAssignedBrands(
  value: unknown,
): string[] | null {
  const assignedBrands = toStringArray(value);

  return assignedBrands.length > 0
    ? assignedBrands
    : null;
}

function mapCurrentMemberResponse(
  member: CurrentMemberResponse | null,
): MemberDTO | null {
  if (!member) {
    return null;
  }

  const raw = member as unknown as UnknownRecord;

  const id = toStringValue(raw.id);
  const companyId = toStringValue(raw.companyId);

  if (!id || !companyId) {
    return null;
  }

  const firstName = toStringValue(raw.firstName);
  const lastName = toStringValue(raw.lastName);

  const displayNameFromResponse = toStringValue(
    raw.displayName,
  );

  const displayNameFromNameParts = [
    lastName,
    firstName,
  ]
    .filter((value) => value.length > 0)
    .join(" ");

  return {
    id,
    uid: toStringValue(raw.uid),

    firstName,
    lastName,
    firstNameKana: toStringValue(
      raw.firstNameKana,
    ),
    lastNameKana: toStringValue(
      raw.lastNameKana,
    ),

    email: toStringValue(raw.email),

    permissions: toStringArray(
      raw.permissions,
    ),

    assignedBrands: toAssignedBrands(
      raw.assignedBrands,
    ),

    companyId,
    status: toStringValue(raw.status),

    createdAt: toStringValue(raw.createdAt),
    updatedAt: toNullableString(
      raw.updatedAt,
    ),
    updatedBy: toNullableString(
      raw.updatedBy,
    ),

    displayName:
      displayNameFromResponse ||
      displayNameFromNameParts,
  };
}

/**
 * useAuth:
 * - AuthContextからログイン中のuserを取得
 * - BackendからcurrentMemberを取得
 * - BackendからcompanyNameを取得
 */
export function useAuth() {
  const ctx = useAuthContext();

  const uid = ctx.user?.uid ?? "";
  const companyIdFromCtx =
    ctx.user?.companyId?.trim() ?? "";

  const [companyName, setCompanyName] =
    useState<string | null>(null);

  const [
    loadingCompanyName,
    setLoadingCompanyName,
  ] = useState(false);

  const [companyError, setCompanyError] =
    useState<string | null>(null);

  const [currentMember, setCurrentMember] =
    useState<MemberDTO | null>(null);

  const [loadingMember, setLoadingMember] =
    useState(false);

  const [memberError, setMemberError] =
    useState<string | null>(null);

  // -------------------------------
  // Fetch companyName
  // - currentMember.companyIdを最優先
  // - なければFirebase AuthのcompanyIdを使用
  // -------------------------------
  useEffect(() => {
    let disposed = false;

    async function run() {
      const effectiveCompanyId =
        currentMember?.companyId.trim() ||
        companyIdFromCtx;

      if (!effectiveCompanyId) {
        setCompanyName(null);
        setCompanyError(null);
        setLoadingCompanyName(false);
        return;
      }

      setLoadingCompanyName(true);
      setCompanyError(null);

      try {
        const name =
          await getCompanyNameByIdCached(
            effectiveCompanyId,
          );

        if (!disposed) {
          setCompanyName(name);
        }
      } catch (error: unknown) {
        if (!disposed) {
          setCompanyName(null);
          setCompanyError(
            error instanceof Error
              ? error.message
              : "failed to fetch company name",
          );
        }
      } finally {
        if (!disposed) {
          setLoadingCompanyName(false);
        }
      }
    }

    void run();

    return () => {
      disposed = true;
    };
  }, [
    companyIdFromCtx,
    currentMember?.companyId,
  ]);

  // -------------------------------
  // Fetch currentMember
  // - Bootstrap完了前の取得失敗を考慮して再試行
  // -------------------------------
  useEffect(() => {
    let disposed = false;

    async function loadMember() {
      if (!uid) {
        setCurrentMember(null);
        setMemberError(null);
        setLoadingMember(false);
        return;
      }

      setCurrentMember(null);
      setLoadingMember(true);
      setMemberError(null);

      try {
        const response = await getCurrentMember({
          retries: 8,
          retryDelayMs: 250,
        });

        if (disposed) {
          return;
        }

        const member =
          mapCurrentMemberResponse(response);

        if (!member) {
          setCurrentMember(null);
          setMemberError(
            "ログインユーザーの会社情報を確認できませんでした。",
          );
          return;
        }

        setCurrentMember(member);
        setMemberError(null);
      } catch (error: unknown) {
        if (!disposed) {
          setCurrentMember(null);
          setMemberError(
            error instanceof Error
              ? error.message
              : "failed to fetch member",
          );
        }
      } finally {
        if (!disposed) {
          setLoadingMember(false);
        }
      }
    }

    void loadMember();

    return () => {
      disposed = true;
    };
  }, [uid]);

  // -------------------------------
  // Public API
  // -------------------------------
  return useMemo(
    () => ({
      ...ctx,

      // company
      companyName,
      loadingCompanyName,
      companyError,

      // currentMember
      currentMember,
      loadingMember,
      memberError,

      // service helpers
      clearCompanyNameCache,
    }),
    [
      ctx,
      companyName,
      loadingCompanyName,
      companyError,
      currentMember,
      loadingMember,
      memberError,
    ],
  );
}