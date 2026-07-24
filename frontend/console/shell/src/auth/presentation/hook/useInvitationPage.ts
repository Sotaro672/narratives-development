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

  // ---- 表示用の所属情報 ----
  const [companyName, setCompanyName] = useState<string>("");
  const [assignedBrandNames, setAssignedBrandNames] = useState<string[]>([]);

  // ============================================================
  // tokenが設定されたらBackendから公開可能な招待情報を取得
  // ============================================================
  useEffect(() => {
    if (!token) {
      setCompanyName("");
      setAssignedBrandNames([]);
      return;
    }

    const run = async () => {
      setLoading(true);
      setError(null);

      try {
        const data = await fetchInvitationInfo(token);

        setCompanyName(data.companyName ?? "");
        setAssignedBrandNames(data.brandNames ?? []);
      } catch (e: any) {
        setCompanyName("");
        setAssignedBrandNames([]);
        setError(e?.message ?? "Unknown error");
      } finally {
        setLoading(false);
      }
    };

    void run();
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

      if (!token) {
        setError(
          "招待トークンが無効です。招待リンクを再度ご確認ください。",
        );
        return;
      }

      if (!email) {
        setError("メールアドレスを入力してください。");
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
        // Firebaseユーザー作成、Auth state更新、Backend招待完了を実行する。
        await completeInvitation({
          token,
          email,
          lastName,
          lastNameKana,
          firstName,
          firstNameKana,
          password,
          passwordConfirm,
        });

        navigate("/", { replace: true });
      } catch (e: any) {
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
    ],
  );

  return {
    formRef,

    // token
    token,
    setToken,

    // email
    email,
    setEmail,

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

    // 表示用の所属情報
    companyName,
    assignedBrandNames,

    // Actions
    handleBack,
    handleCreate,
    handleSubmit,
  };
}