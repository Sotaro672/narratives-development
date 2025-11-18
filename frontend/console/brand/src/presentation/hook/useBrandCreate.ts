// frontend/console/brand/src/presentation/hook/useBrandCreate.ts
import { useState, useCallback, useMemo, useEffect } from "react";
import { useNavigate } from "react-router-dom";

import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";
import type { BrandPatch } from "../../domain/entity/brand";

// Member 取得用
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";
import {
  fetchMemberListWithToken,
  formatLastFirst,
} from "../../../../member/src/infrastructure/query/memberQuery";
import type { Member } from "../../../../member/src/domain/entity/member";
import type { MemberFilter } from "../../../../member/src/domain/repository/memberRepository";
import type { Page } from "../../../../shell/src/shared/types/common/common";

export function useBrandCreate() {
  const navigate = useNavigate();

  // useAuth から currentMember を取得（companyName 系は使わない）
  const { currentMember } = useAuth();

  // Auth / currentMember から companyId を取得（入力させない）
  const companyId = useMemo(
    () => (currentMember?.companyId ?? "").trim(),
    [currentMember?.companyId],
  );

  // BrandPatch に対応するフォーム状態（必要なものだけ）
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [websiteUrl, setWebsiteUrl] = useState("");

  // ブランド責任者（managerId）… 選択された Member の id を保持
  const [managerId, setManagerId] = useState<string | null>(null);

  // メンバー選択用
  const [managerOptions, setManagerOptions] = useState<Member[]>([]);
  const [loadingManagers, setLoadingManagers] = useState(false);
  const [managerError, setManagerError] = useState<string | null>(null);

  // isActive は作成時は常に true
  const isActive = true;

  // ---------------------------
  // メンバー一覧取得（ブランド責任者候補）
  // ---------------------------
  useEffect(() => {
    let cancelled = false;

    async function loadManagers() {
      try {
        setLoadingManagers(true);
        setManagerError(null);

        const user = auth.currentUser;
        if (!user) {
          setManagerError("ログインユーザーが取得できませんでした。");
          return;
        }

        const token = await user.getIdToken();

        const page: Page = { limit: 50, offset: 0 };
        const filter: MemberFilter = {};

        const { items } = await fetchMemberListWithToken(token, page, filter);

        if (cancelled) return;
        setManagerOptions(items);

        // 初期値として最初のメンバーを選択（任意）
        if (!managerId && items.length > 0) {
          setManagerId(items[0].id);
        }
      } catch (e: any) {
        if (!cancelled) {
          setManagerError(
            e?.message ?? "ブランド責任者候補の取得に失敗しました。",
          );
        }
      } finally {
        if (!cancelled) {
          setLoadingManagers(false);
        }
      }
    }

    loadManagers();
    return () => {
      cancelled = true;
    };
  }, [managerId]);

  // ---------------------------
  // 戻る
  // ---------------------------
  const handleBack = useCallback(() => {
    navigate(-1);
  }, [navigate]);

  // ---------------------------
  // 保存処理（BrandPatch）
  //  - ブランド名 (name) を必須
  //  - ブランド責任者 (managerId) を必須
  // ---------------------------
  const handleSave = useCallback(() => {
    const trimmedName = name.trim();

    if (!trimmedName) {
      alert("ブランド名は必須です。");
      return;
    }

    if (!managerId) {
      alert("ブランド責任者を選択してください。");
      return;
    }

    const payload: BrandPatch = {
      companyId: companyId || null,
      name: trimmedName,
      description: description || null,
      websiteUrl: websiteUrl || null,
      isActive: true, // 作成時は常に true
      managerId,
      // walletAddress は自動設定されるためフロントからは送らない
    };

    console.log("保存 payload:", payload);
    alert("ブランド情報を保存しました（モック）");
  }, [companyId, name, description, websiteUrl, managerId]);

  return {
    // 会社情報
    companyId,

    // ブランド基本情報
    name,
    setName,

    description,
    setDescription,

    websiteUrl,
    setWebsiteUrl,

    // ブランド責任者
    managerId,
    setManagerId,
    managerOptions,
    loadingManagers,
    managerError,
    formatLastFirst,

    // ステータス（作成時は常に true）
    isActive,

    // 操作
    handleBack,
    handleSave,
  };
}
