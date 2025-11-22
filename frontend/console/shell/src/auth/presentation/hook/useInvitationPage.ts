// frontend/console/shell/src/auth/presentation/hook/useInvitationPage.ts
import { useCallback, useEffect, useRef, useState } from "react";
import {
  fetchInvitationInfo,
  completeInvitation,
} from "../../application/invitationService";

// ★ サインイン用
import { signInWithEmailAndPassword } from "firebase/auth";
import { auth } from "../../infrastructure/config/firebaseClient";

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

        // --- ログ ---
        // eslint-disable-next-line no-console
        console.log("[InvitationPage] Invitation info loaded:", {
          token,
          email: data.email,
          companyId: data.companyId,
          assignedBrandIds: data.assignedBrandIds,
          permissions: data.permissions,
        });

        // 会社名・ブランド名の解決は、必要であれば別途
        // useInvitationPage から API を呼ぶ形で後から追加できます
      } catch (e: any) {
        // eslint-disable-next-line no-console
        console.error("[InvitationPage] failed to load invitation info", e);
        setError(e?.message ?? "Unknown error");
      } finally {
        setLoading(false);
      }
    };

    run();
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
        // 1) 招待完了 (backend + Firebase createUser + verify mail)
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

        // 2) ★ Firebase Authentication へサインイン
        //    （作成した email / password をそのまま使用）
        await signInWithEmailAndPassword(auth, email, password);

        // eslint-disable-next-line no-console
        console.log("[Invitation:create] completed & signed in for:", email);
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

    // 表示用の名前（今後使う場合に備えてそのまま残す）
    companyName,
    assignedBrandNames,

    // Actions
    handleBack,
    handleCreate,
    handleSubmit,
  };
}
