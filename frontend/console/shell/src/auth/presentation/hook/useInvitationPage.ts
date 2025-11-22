// frontend/console/shell/src/auth/presentation/hook/useInvitationPage.ts
import { useCallback, useEffect, useRef, useState } from "react";
import {
  fetchInvitationInfo,
  completeInvitation,
  fetchCompanyNameById,
  fetchBrandNamesByIds,
} from "../../application/invitationService";

export function useInvitationPage() {
  // ---- フォーム ref ----
  const formRef = useRef<HTMLFormElement>(null);

  // ---- 招待トークン ----
  const [token, setToken] = useState<string>("");

  // ---- ローディング / エラー ----
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // ---- email（追加） ----
  const [email, setEmail] = useState<string>("");

  // ---- 氏名系 ----
  const [lastName, setLastName] = useState("");
  const [lastNameKana, setLastNameKana] = useState("");
  const [firstName, setFirstName] = useState("");
  const [firstNameKana, setFirstNameKana] = useState("");

  // ---- パスワード ----
  const [password, setPassword] = useState("");
  const [passwordConfirm, setPasswordConfirm] = useState("");

  // ---- 招待トークンから取得する割り当て情報（ID） ----
  const [companyId, setCompanyId] = useState<string>("");
  const [assignedBrandIds, setAssignedBrandIds] = useState<string[]>([]);
  const [permissions, setPermissions] = useState<string[]>([]);

  // ---- 表示用の名前 ----
  const [companyName, setCompanyName] = useState<string>("");
  const [assignedBrandNames, setAssignedBrandNames] = useState<string[]>([]);

  // ============================================================
  // 🔥 token が設定されたら backend から InvitationInfo を取得
  // ============================================================
  useEffect(() => {
    if (!token) return;

    const run = async () => {
      setLoading(true);
      setError(null);

      try {
        const data = await fetchInvitationInfo(token);

        // 📨 email
        if (data.email) setEmail(data.email);

        // ID はそのまま state に保持
        setCompanyId(data.companyId);
        const brands = data.assignedBrandIds || [];
        const perms = data.permissions || [];
        setAssignedBrandIds(brands);
        setPermissions(perms);

        // 会社名・ブランド名を並列取得
        try {
          const [companyNameResolved, brandNamesResolved] = await Promise.all([
            data.companyId
              ? fetchCompanyNameById(data.companyId)
              : Promise.resolve(""),
            fetchBrandNamesByIds(brands),
          ]);

          if (companyNameResolved) {
            setCompanyName(companyNameResolved);
          } else {
            setCompanyName("");
          }
          setAssignedBrandNames(brandNamesResolved);
        } catch (nameErr) {
          // eslint-disable-next-line no-console
          console.warn("[InvitationPage] failed to resolve names", nameErr);
          // 失敗した場合は名前は空・ID表示にフォールバックさせる
          setCompanyName("");
          setAssignedBrandNames([]);
        }

        // --- ログ ---
        // eslint-disable-next-line no-console
        console.log("[InvitationPage] Invitation info loaded:", {
          token,
          email: data.email,
          companyId: data.companyId,
          companyName,
          assignedBrandIds: data.assignedBrandIds,
          permissions: data.permissions,
        });
      } catch (e: any) {
        // eslint-disable-next-line no-console
        console.error("[InvitationPage] failed to load invitation info", e);
        setError(e?.message ?? "Unknown error");
      } finally {
        setLoading(false);
      }
    };

    run();
    // companyName は run 内で更新されるので依存から外しておく
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  // ---- Navigation ----
  const handleBack = useCallback(() => {
    history.back();
  }, []);

  const handleCreate = useCallback(() => {
    formRef.current?.requestSubmit();
  }, []);

  // ---- Submit ----
  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      setError(null);

      // --- ログに email を追記 ---
      // eslint-disable-next-line no-console
      console.log("[Invitation:create] payload:", {
        token,
        email,
        lastName,
        lastNameKana,
        firstName,
        firstNameKana,
        password,
        passwordConfirm,
        companyId,
        assignedBrandIds,
        permissions,
      });

      // バリデーション
      if (!token) {
        setError("招待トークンが無効です。招待リンクを再度ご確認ください。");
        return;
      }
      if (!password || !passwordConfirm) {
        setError("パスワードを入力してください。");
        return;
      }
      if (password !== passwordConfirm) {
        setError("パスワードが一致しません。");
        return;
      }

      setLoading(true);

      try {
        await completeInvitation({
          token,
          lastName,
          lastNameKana,
          firstName,
          firstNameKana,
          password,
          passwordConfirm,
          companyId,
          assignedBrandIds,
          permissions,
        });

        // eslint-disable-next-line no-console
        console.log("[Invitation:create] completed for:", email);
      } catch (e: any) {
        // eslint-disable-next-line no-console
        console.error("[InvitationPage] handleSubmit error", e);
        setError(e?.message ?? "Unexpected error");
      } finally {
        setLoading(false);
      }
    },
    [
      token,
      email,
      lastName,
      lastNameKana,
      firstName,
      firstNameKana,
      password,
      passwordConfirm,
      companyId,
      assignedBrandIds,
      permissions,
    ],
  );

  // ---- return ----
  return {
    formRef,

    // token
    token,
    setToken,

    // email（UI 側で表示も可能）
    email,

    // ローディング・エラー
    loading,
    error,

    // 氏名
    lastName,
    setLastName,
    lastNameKana,
    setLastNameKana,
    firstName,
    setFirstName,
    firstNameKana,
    setFirstNameKana,

    // パスワード
    password,
    setPassword,
    passwordConfirm,
    setPasswordConfirm,

    // 割り当て情報（ID）
    companyId,
    assignedBrandIds,
    permissions,

    // 表示用の名前
    companyName,
    assignedBrandNames,

    // Actions
    handleBack,
    handleCreate,
    handleSubmit,
  };
}
