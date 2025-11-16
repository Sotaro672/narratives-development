// frontend\console\shell\src\auth\presentation\hook\useCurrentMember.ts
/// <reference types="vite/client" />

import { useEffect, useMemo, useState } from "react";
import { useAuthContext } from "../../application/AuthContext";

// Application 層のサービス
import {
  getCompanyNameByIdCached,
  clearCompanyNameCache,
} from "../../application/companyService";

import { fetchCurrentMember } from "../../application/memberService";

// Domain 型
import type { MemberDTO } from "../../domain/entity/member";

/**
 * useAuth:
 * - AuthContext からログイン中の user を取得
 * - backend から currentMember を取得
 * - backend から companyName を取得
 */
export function useAuth() {
  const ctx = useAuthContext(); // { user, loading }

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
        if (!disposed) setCompanyName(name);
      } catch (e: any) {
        if (!disposed) {
          setCompanyName(null);
          setCompanyError(e?.message ?? "failed to fetch company name");
        }
      } finally {
        if (!disposed) setLoadingCompanyName(false);
      }
    }

    run();
    return () => {
      disposed = true;
    };
  }, [companyIdFromCtx, currentMember?.companyId]);

  // -------------------------------
  // Fetch currentMember
  // -------------------------------
  useEffect(() => {
    let disposed = false;

    async function loadMember() {
      if (!uid) {
        setCurrentMember(null);
        setMemberError(null);
        return;
      }

      setLoadingMember(true);
      setMemberError(null);

      try {
        const m = await fetchCurrentMember(uid);
        if (!disposed) setCurrentMember(m);
      } catch (e: any) {
        if (!disposed) {
          setCurrentMember(null);
          setMemberError(e?.message ?? "failed to fetch member");
        }
      } finally {
        if (!disposed) setLoadingMember(false);
      }
    }

    loadMember();
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
