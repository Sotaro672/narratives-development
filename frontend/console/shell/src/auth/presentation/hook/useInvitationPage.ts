// frontend/console/shell/src/auth/presentation/hook/useInvitationPage.ts

import {
  useCallback,
  useEffect,
  useState,
  type FormEvent,
} from "react";
import {
  useNavigate,
} from "react-router-dom";

import {
  completeInvitation,
  fetchInvitationInfo,
} from "../../application/invitationService";

export function useInvitationPage() {
  const navigate = useNavigate();

  // -------------------------
  // 招待トークン
  // -------------------------

  const [
    token,
    setToken,
  ] = useState("");

  // -------------------------
  // 処理状態
  // -------------------------

  const [
    loadingInvitationInfo,
    setLoadingInvitationInfo,
  ] = useState(false);

  const [
    submitting,
    setSubmitting,
  ] = useState(false);

  const [
    error,
    setError,
  ] = useState<string | null>(null);

  // -------------------------
  // 入力値
  // -------------------------

  const [
    email,
    setEmail,
  ] = useState("");

  const [
    lastName,
    setLastName,
  ] = useState("");

  const [
    lastNameKana,
    setLastNameKana,
  ] = useState("");

  const [
    firstName,
    setFirstName,
  ] = useState("");

  const [
    firstNameKana,
    setFirstNameKana,
  ] = useState("");

  const [
    password,
    setPassword,
  ] = useState("");

  const [
    passwordConfirm,
    setPasswordConfirm,
  ] = useState("");

  // -------------------------
  // 招待情報
  // -------------------------

  const [
    companyName,
    setCompanyName,
  ] = useState("");

  const [
    assignedBrandNames,
    setAssignedBrandNames,
  ] = useState<string[]>([]);

  // ============================================================
  // 招待情報取得
  // ============================================================

  useEffect(() => {
    let disposed = false;

    if (!token) {
      setCompanyName("");
      setAssignedBrandNames([]);
      setLoadingInvitationInfo(false);
      setError(null);

      return () => {
        disposed = true;
      };
    }

    async function loadInvitationInfo() {
      setLoadingInvitationInfo(true);
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
            : "招待情報の取得に失敗しました。",
        );
      } finally {
        if (!disposed) {
          setLoadingInvitationInfo(false);
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

  const handleSubmit =
    useCallback(
      async (
        event: FormEvent<HTMLFormElement>,
      ) => {
        event.preventDefault();

        if (submitting) {
          return;
        }

        setError(null);

        const normalizedEmail =
          email.trim();

        if (!token) {
          setError(
            "招待トークンが無効です。招待リンクを再度ご確認ください。",
          );
          return;
        }

        if (!normalizedEmail) {
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

        setSubmitting(true);

        try {
          await completeInvitation({
            token,
            email: normalizedEmail,
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
              : "招待の完了処理に失敗しました。",
          );
        } finally {
          setSubmitting(false);
        }
      },
      [
        navigate,
        submitting,
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

  const loading =
    loadingInvitationInfo ||
    submitting;

  return {
    // 招待トークン
    setToken,

    // 処理状態
    loading,
    loadingInvitationInfo,
    submitting,
    error,

    // メールアドレス
    email,
    setEmail,

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

    // 招待情報
    companyName,
    assignedBrandNames,

    // 送信
    handleSubmit,
  };
}