// frontend/console/shell/src/auth/presentation/hook/useCurrentMember.ts

/// <reference types="vite/client" />

import { useEffect, useMemo, useState } from "react";
import { useAuthContext } from "../../application/AuthContext";

// Application 層のサービス
import {
  getCompanyNameByIdCached,
  clearCompanyNameCache,
} from "../../application/companyService";

import {
  getCurrentMember,
  type CurrentMemberResponse,
} from "../../application/authService";

// Domain 型
import type { MemberDTO } from "../../domain/entity/member";

function mapCurrentMemberResponse(
  member: CurrentMemberResponse | null,
): MemberDTO | null {
  if (!member) {
    return null;
  }

  const id = String(member.id ?? "").trim();
  const companyId = String(member.companyId ?? "").trim();

  if (!id || !companyId) {
    return null;
  }

  const uid = String(member.uid ?? "").trim();

  return {
    id,
    uid: uid || null,
    firstName: member.firstName ?? null,
    lastName: member.lastName ?? null,
    firstNameKana: member.firstNameKana ?? null,
    lastNameKana: member.lastNameKana ?? null,
    email: member.email ?? null,
    companyId,
    displayName: member.displayName ?? null,
  };
}

/**
 * useAuth:
 * - AuthContext からログイン中の user を取得
 * - backend から currentMember を取得
 * - backend から companyName を取得
 */
export function useAuth() {
  const ctx = useAuthContext();

  const uid = ctx.user?.uid ?? "";
  const companyIdFromCtx = ctx.user?.companyId?.trim() ?? "";

  const [companyName, setCompanyName] = useState<string | null>(null);
  const [loadingCompanyName, setLoadingCompanyName] = useState(false);
  const [companyError, setCompanyError] = useState<string | null>(null);

  const [currentMember, setCurrentMember] = useState<MemberDTO | null>(null);
  const [loadingMember, setLoadingMember] = useState(false);
  const [memberError, setMemberError] = useState<string | null>(null);

  // -------------------------------
  // Fetch companyName
  //   - currentMember.companyId を最優先
  //   - 無ければ Firebase Auth の companyId を使用
  // -------------------------------
  useEffect(() => {
    let disposed = false;

    async function run() {
      const effectiveCompanyId =
        (currentMember?.companyId ?? "").trim() || companyIdFromCtx;

      if (!effectiveCompanyId) {
        setCompanyName(null);
        setCompanyError(null);
        setLoadingCompanyName(false);
        return;
      }

      setLoadingCompanyName(true);
      setCompanyError(null);

      try {
        const name = await getCompanyNameByIdCached(effectiveCompanyId);

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
  }, [companyIdFromCtx, currentMember?.companyId]);

  // -------------------------------
  // Fetch currentMember
  //   - bootstrap 完了前の取得失敗を考慮して再試行
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

        const member = mapCurrentMemberResponse(response);

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