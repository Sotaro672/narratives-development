// frontend/console/shell/src/auth/presentation/hook/useInvitationPage.ts

import {
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";
import { useNavigate } from "react-router-dom";

import {
  completeInvitation,
  fetchInvitationInfo,
} from "../../application/invitationService";

export function useInvitationPage() {
  const navigate = useNavigate();

  const formRef =
    useRef<HTMLFormElement>(null);

  const [token, setToken] =
    useState("");

  const [, setLoading] =
    useState(false);

  const [, setError] =
    useState<string | null>(null);

  const [email, setEmail] =
    useState("");

  const [lastName, setLastName] =
    useState("");

  const [
    lastNameKana,
    setLastNameKana,
  ] = useState("");

  const [firstName, setFirstName] =
    useState("");

  const [
    firstNameKana,
    setFirstNameKana,
  ] = useState("");

  const [password, setPassword] =
    useState("");

  const [
    passwordConfirm,
    setPasswordConfirm,
  ] = useState("");

  const [
    companyName,
    setCompanyName,
  ] = useState("");

  const [
    assignedBrandNames,
    setAssignedBrandNames,
  ] = useState<string[]>([]);

  // ============================================================
  // tokenが設定されたらBackendから公開可能な招待情報を取得
  // ============================================================
  useEffect(() => {
    if (!token) {
      setCompanyName("");
      setAssignedBrandNames([]);
      return;
    }

    let disposed = false;

    async function loadInvitationInfo() {
      setLoading(true);
      setError(null);

      try {
        const data =
          await fetchInvitationInfo(
            token,
          );

        if (disposed) {
          return;
        }

        setCompanyName(
          data.companyName ?? "",
        );

        setAssignedBrandNames(
          data.brandNames ?? [],
        );
      } catch (error: unknown) {
        if (disposed) {
          return;
        }

        setCompanyName("");
        setAssignedBrandNames([]);

        setError(
          error instanceof Error
            ? error.message
            : "Unknown error",
        );
      } finally {
        if (!disposed) {
          setLoading(false);
        }
      }
    }

    void loadInvitationInfo();

    return () => {
      disposed = true;
    };
  }, [token]);

  // ============================================================
  // 招待完了
  // ============================================================
  const handleSubmit = useCallback(
    async (
      event: React.FormEvent,
    ) => {
      event.preventDefault();
      setError(null);

      if (!token) {
        setError(
          "招待トークンが無効です。招待リンクを再度ご確認ください。",
        );
        return;
      }

      if (!email) {
        setError(
          "メールアドレスを入力してください。",
        );
        return;
      }

      if (
        !password ||
        !passwordConfirm
      ) {
        setError(
          "パスワードを入力してください。",
        );
        return;
      }

      if (
        password !==
        passwordConfirm
      ) {
        setError(
          "パスワードが一致しません。",
        );
        return;
      }

      setLoading(true);

      try {
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

        navigate("/", {
          replace: true,
        });
      } catch (error: unknown) {
        setError(
          error instanceof Error
            ? error.message
            : "Unexpected error",
        );
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

    setToken,

    email,
    setEmail,

    lastName,
    setLastName,
    lastNameKana,
    setLastNameKana,
    firstName,
    setFirstName,
    firstNameKana,
    setFirstNameKana,

    password,
    setPassword,
    passwordConfirm,
    setPasswordConfirm,

    companyName,
    assignedBrandNames,

    handleSubmit,
  };
}