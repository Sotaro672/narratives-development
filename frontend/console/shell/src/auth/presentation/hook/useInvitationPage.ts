// frontend/console/shell/src/auth/presentation/hook/useInvitationPage.ts
import { useCallback, useEffect, useRef, useState } from "react";

// 🔙 他のサービスと同様に BACKEND の BASE URL を決める
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/g,
    "",
  ) ?? "";

const FALLBACK_BASE = "https://narratives-backend-871263659099.asia-northeast1.run.app";
const API_BASE = ENV_BASE || FALLBACK_BASE;

export function useInvitationPage() {
  // ---- フォーム ref ----
  const formRef = useRef<HTMLFormElement>(null);

  // ---- 招待トークン ----
  const [token, setToken] = useState<string>("");

  // ---- ローディング / エラー ----
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // ---- 氏名系 ----
  const [lastName, setLastName] = useState("");
  const [lastNameKana, setLastNameKana] = useState("");
  const [firstName, setFirstName] = useState("");
  const [firstNameKana, setFirstNameKana] = useState("");

  // ---- パスワード ----
  const [password, setPassword] = useState("");
  const [passwordConfirm, setPasswordConfirm] = useState("");

  // ---- 招待トークンから取得する割り当て情報 ----
  const [companyId, setCompanyId] = useState<string>("");
  const [assignedBrandIds, setAssignedBrandIds] = useState<string[]>([]);
  const [permissions, setPermissions] = useState<string[]>([]);

  // ============================================================
  // 🔥 token が設定されたら backend から InvitationInfo を取得
  // ============================================================
  useEffect(() => {
    if (!token) return;

    const fetchInvitationInfo = async () => {
      setLoading(true);
      setError(null);

      try {
        // ✅ ここを相対パスではなく BACKEND 直指定に変更
        const url = `${API_BASE}/api/invitation?token=${encodeURIComponent(token)}`;

        // eslint-disable-next-line no-console
        console.log("[InvitationPage] Fetching invitation info:", url);

        const res = await fetch(url, {
          method: "GET",
          headers: {
            "Content-Type": "application/json",
          },
        });

        const text = await res.text();
        // eslint-disable-next-line no-console
        console.log("[InvitationPage] raw response:", text);

        if (!res.ok) {
          throw new Error(`Failed to load invitation info (status ${res.status})`);
        }

        const data = JSON.parse(text) as {
          memberId: string;
          companyId: string;
          assignedBrandIds: string[];
          permissions: string[];
        };

        // ---- API の値を state に反映 ----
        setCompanyId(data.companyId);
        setAssignedBrandIds(data.assignedBrandIds || []);
        setPermissions(data.permissions || []);
      } catch (e: any) {
        // eslint-disable-next-line no-console
        console.error("[InvitationPage] failed to load invitation info", e);
        setError(e.message ?? "Unknown error");
      } finally {
        setLoading(false);
      }
    };

    fetchInvitationInfo();
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
    (e: React.FormEvent) => {
      e.preventDefault();

      // eslint-disable-next-line no-console
      console.log("[Invitation:create] payload:", {
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

      // ここに以下の処理を実装する：
      // 1) backend: /invitation/validate(token)
      // 2) auth.createUserWithEmailAndPassword
      // 3) sendEmailVerification
      // 4) backend: /invitation/complete(token, uid,...)
    },
    [
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
    ],
  );

  // ---- return ----
  return {
    formRef,

    // token
    token,
    setToken,

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

    // 割り当て情報
    companyId,
    assignedBrandIds,
    permissions,

    // Actions
    handleBack,
    handleCreate,
    handleSubmit,
  };
}
