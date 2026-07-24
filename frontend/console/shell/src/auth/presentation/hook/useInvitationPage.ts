// frontend/console/shell/src/auth/presentation/hook/useInvitationPage.ts
import { useCallback, useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";

import {
  completeInvitation,
  fetchInvitationInfo,
} from "../../application/invitationService";

export function useInvitationPage() {
  const navigate = useNavigate();

  // ---- フォーム ref ----
  const formRef = useRef<HTMLFormElement>(null);

  // ---- 招待トークン ----
  const [token, setToken] = useState<string>("");

  // ---- ローディング / エラー ----
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // ---- email ----
  const [email, setEmail] = useState<string>("");

  // ---- 氏名系 ----
  const [lastName, setLastName] = useState("");
  const [lastNameKana, setLastNameKana] = useState("");
  const [firstName, setFirstName] = useState("");
  const [firstNameKana, setFirstNameKana] = useState("");

  // ---- パスワード ----
  const [password, setPassword] = useState("");
  const [passwordConfirm, setPasswordConfirm] = useState("");

  // ---- 招待トークンから取得する割り当て情報（ID）----
  const [companyId, setCompanyId] = useState<string>("");
  const [assignedBrandIds, setAssignedBrandIds] = useState<string[]>([]);
  const [permissions, setPermissions] = useState<string[]>([]);

  // ---- 表示用の名前 ----
  const [companyName, setCompanyName] = useState<string>("");
  const [assignedBrandNames, setAssignedBrandNames] = useState<string[]>([]);

  // ============================================================
  // tokenが設定されたらBackendからInvitationInfoを取得
  // ============================================================
  useEffect(() => {
    if (!token) return;

    const run = async () => {
      setLoading(true);
      setError(null);

      try {
        const data = await fetchInvitationInfo(token);

        if (data.email) {
          setEmail(data.email);
        }

        const brands = data.assignedBrandIds || [];
        const perms = data.permissions || [];

        setCompanyId(data.companyId);
        setAssignedBrandIds(brands);
        setPermissions(perms);

        setCompanyName(data.companyName ?? data.companyId ?? "");
        setAssignedBrandNames(data.brandNames ?? brands);

        // eslint-disable-next-line no-console
        console.log("[InvitationPage] Invitation info loaded:", {
          token,
          email: data.email,
          companyId: data.companyId,
          companyName: data.companyName,
          assignedBrandIds: data.assignedBrandIds,
          assignedBrandNames: data.brandNames,
          permissions: data.permissions,
        });
      } catch (e: any) {
        // eslint-disable-next-line no-console
        console.error(
          "[InvitationPage] failed to load invitation info",
          e,
        );

        setError(e?.message ?? "Unknown error");
      } finally {
        setLoading(false);
      }
    };

    run();
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
        companyName,
        assignedBrandIds,
        assignedBrandNames,
        permissions,
      });

      if (!token) {
        setError(
          "招待トークンが無効です。招待リンクを再度ご確認ください。",
        );
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

      if (!email) {
        setError("招待情報にメールアドレスがありません。");
        return;
      }

      setLoading(true);

      try {
        // Firebaseユーザー作成、Auth state更新、Backend招待完了を実行する。
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

        navigate("/", { replace: true });
      } catch (e: any) {
        // eslint-disable-next-line no-console
        console.error("[InvitationPage] handleSubmit error", e);

        setError(e?.message ?? "Unexpected error");
      } finally {
        setLoading(false);
      }
    },
    [
      navigate,
      token,
      email,
      lastName,
      lastNameKana,
      firstName,
      firstNameKana,
      password,
      passwordConfirm,
      companyId,
      companyName,
      assignedBrandIds,
      assignedBrandNames,
      permissions,
    ],
  );

  return {
    formRef,

    // token
    token,
    setToken,

    // email
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