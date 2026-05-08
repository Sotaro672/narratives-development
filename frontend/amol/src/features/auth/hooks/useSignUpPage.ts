// frontend/src/features/auth/hooks/useSignUpPage.ts

import { useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

import { auth } from "../../../lib/firebase";
import { createAccountAndSendVerification } from "../services/createAccountService";
import {
  isEmailValid,
  isPasswordMatch,
  isPasswordValid,
} from "../utils/authValidation";

export function useSignUpPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  const from = searchParams.get("from");
  const intent = searchParams.get("intent");

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [passwordConfirmation, setPasswordConfirmation] = useState("");
  const [agree, setAgree] = useState(false);

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const topMessage = useMemo(() => {
    if (intent === "purchase") {
      return "購入するにはアカウント作成が必要です。";
    }

    return "メールアドレスとパスワードを入力して新規登録してください。";
  }, [intent]);

  const loginBackTo = useMemo(() => {
    const params = new URLSearchParams();

    if (from) params.set("from", from);
    if (intent) params.set("intent", intent);

    const query = params.toString();

    return query ? `/signin?${query}` : "/signin";
  }, [from, intent]);

  const canSubmit = useMemo(() => {
    return (
      !loading &&
      agree &&
      isEmailValid(email) &&
      isPasswordValid(password) &&
      isPasswordMatch(password, passwordConfirmation)
    );
  }, [agree, email, loading, password, passwordConfirmation]);

  function clearError() {
    if (error) {
      setError("");
    }
  }

  async function handleSignUp() {
    if (loading) return;

    setError("");
    setLoading(true);

    try {
      const result = await createAccountAndSendVerification({
        auth,
        emailRaw: email,
        password,
        passwordConfirmation,
        agree,
      });

      if (!result.ok) {
        setError(result.error);
        return;
      }

      const params = new URLSearchParams();

      if (result.email) params.set("email", result.email);
      if (from) params.set("from", from);
      if (intent) params.set("intent", intent);

      const query = params.toString();

      navigate(query ? `/verification-sent?${query}` : "/verification-sent");
    } catch (e) {
      if (e instanceof Error) {
        setError(e.message);
      } else {
        setError("新規登録に失敗しました。");
      }
    } finally {
      setLoading(false);
    }
  }

  return {
    email,
    setEmail,
    password,
    setPassword,
    passwordConfirmation,
    setPasswordConfirmation,
    agree,
    setAgree,
    loading,
    error,
    topMessage,
    loginBackTo,
    canSubmit,
    clearError,
    handleSignUp,
  };
}