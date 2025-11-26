// frontend/console/brand/src/presentation/hook/useBrandCreate.ts
import { useState, useCallback, useMemo, useEffect } from "react";
import { useNavigate } from "react-router-dom";

import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";
import type { Member } from "../../../../member/src/domain/entity/member";
import type { MemberFilter } from "../../../../member/src/domain/repository/memberRepository";
import type { Brand } from "../../domain/entity/brand";
import { brandRepositoryHTTP } from "../../infrastructure/http/brandRepositoryHTTP";

// メンバー取得用 HTTP リポジトリ
import { MemberRepositoryHTTP } from "../../../../member/src/infrastructure/http/memberRepositoryHTTP";

const memberRepo = new MemberRepositoryHTTP();

// 姓名フォーマット（brand 作成画面専用）
function formatLastFirst(
  lastName?: string | null,
  firstName?: string | null,
) {
  const ln = String(lastName ?? "").trim();
  const fn = String(firstName ?? "").trim();
  if (ln && fn) return `${ln} ${fn}`;
  if (ln) return ln;
  if (fn) return fn;
  return "";
}

export function useBrandCreate() {
  const navigate = useNavigate();
  const { currentMember } = useAuth();

  const companyId = useMemo(
    () => (currentMember?.companyId ?? "").trim(),
    [currentMember?.companyId],
  );

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [websiteUrl, setWebsiteUrl] = useState("");

  const [managerId, setManagerId] = useState<string | null>(null);

  const [nameError, setNameError] = useState<string | null>(null);
  const [managerIdError, setManagerIdError] = useState<string | null>(null);

  const [managerOptions, setManagerOptions] = useState<Member[]>([]);
  const [loadingManagers, setLoadingManagers] = useState(false);
  const [managerError, setManagerError] = useState<string | null>(null);

  const isActive = true;

  useEffect(() => {
    let cancelled = false;

    async function loadManagers() {
      try {
        setLoadingManagers(true);
        setManagerError(null);

        // ページネーションはこの画面では使わないので 1 ページ固定
        const filter: MemberFilter = {};
        const { items } = await memberRepo.list(
          { number: 1, perPage: 200, totalPages: 1 },
          filter,
        );

        if (cancelled) return;
        setManagerOptions(items);
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
        if (!cancelled) setLoadingManagers(false);
      }
    }

    loadManagers();
    return () => {
      cancelled = true;
    };
  }, [managerId]);

  const handleBack = useCallback(() => {
    navigate(-1);
  }, [navigate]);

  // 実際に /brands へ POST
  const handleSave = useCallback(async () => {
    const trimmedName = name.trim();
    const trimmedManagerId = (managerId ?? "").trim();

    let hasError = false;
    if (!trimmedName) {
      setNameError("ブランド名は必須です。");
      hasError = true;
    } else {
      setNameError(null);
    }

    if (!trimmedManagerId) {
      setManagerIdError("ブランド責任者は必須です。");
      hasError = true;
    } else {
      setManagerIdError(null);
    }

    if (hasError) {
      alert("ブランド名とブランド責任者を入力してください。");
      return;
    }

    if (!companyId) {
      alert("companyId が取得できません。");
      return;
    }

    // backend の Create は Brand 全体を受ける想定
    const payload: Omit<Brand, "createdAt" | "updatedAt"> = {
      id: "", // サーバ採番
      companyId,
      name: trimmedName,
      description: description || "",
      websiteUrl: websiteUrl || "",
      isActive: true,
      managerId: trimmedManagerId,
      walletAddress: "pending", // サーバで正式値に更新される前提
      createdBy: (currentMember?.id ?? null) as any, // TS 型の都合：サーバ側で上書き可
      updatedBy: null as any,
      deletedAt: null as any,
      deletedBy: null as any,
      // createdAt/updatedAt は除外（サーバで付与/無視）
    } as any;

    try {
      console.log("[brand] create payload", payload);
      const created = await brandRepositoryHTTP.create(payload);
      console.log("[brand] created", created);
      alert("ブランドを登録しました。");
      navigate("/brand"); // 一覧などへ遷移
    } catch (e: any) {
      console.error("[brand] create error:", e);
      alert(`ブランド登録に失敗しました: ${e?.message ?? e}`);
    }
  }, [
    companyId,
    name,
    description,
    websiteUrl,
    managerId,
    currentMember?.id,
    navigate,
  ]);

  return {
    companyId,

    name,
    setName,
    nameError,
    description,
    setDescription,
    websiteUrl,
    setWebsiteUrl,

    managerId,
    setManagerId,
    managerIdError,
    managerOptions,
    loadingManagers,
    managerError,
    formatLastFirst,

    isActive,
    handleBack,
    handleSave,
  };
}
