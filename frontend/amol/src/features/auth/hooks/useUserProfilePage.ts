//frontend\amol\src\features\auth\hooks\useUserProfilePage.ts
import { useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { signOut as firebaseSignOut } from "firebase/auth";

import { auth } from "../../../lib/firebase";
import { saveUserProfile } from "../api/userApi";

export function useUserProfilePage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  const from = searchParams.get("from");
  const intent = searchParams.get("intent");

  const [lastName, setLastName] = useState("");
  const [lastNameKana, setLastNameKana] = useState("");
  const [firstName, setFirstName] = useState("");
  const [firstNameKana, setFirstNameKana] = useState("");

  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  const backendUrl =
    import.meta.env.VITE_API_BASE_URL || "";

  const loggedIn = auth.currentUser !== null;

  const canSave = useMemo(() => {
    return (
      loggedIn &&
      !saving &&
      lastName.trim().length > 0 &&
      lastNameKana.trim().length > 0 &&
      firstName.trim().length > 0 &&
      firstNameKana.trim().length > 0
    );
  }, [firstName, firstNameKana, lastName, lastNameKana, loggedIn, saving]);

  const signInPath = useMemo(() => {
    const params = new URLSearchParams();

    if (from) params.set("from", from);
    if (intent) params.set("intent", intent);

    const query = params.toString();

    return query ? `/signin?${query}` : "/signin";
  }, [from, intent]);

  function clearMessages() {
    if (error) setError("");
    if (message) setMessage("");
  }

  async function goSignIn() {
    navigate(signInPath);
  }

  async function signOut() {
    setError("");
    setMessage("");

    try {
      await firebaseSignOut(auth);
      navigate("/", { replace: true });
    } catch (error) {
      if (error instanceof Error) {
        setError(error.message);
      } else {
        setError("サインアウトに失敗しました。");
      }
    }
  }

  async function save() {
    if (saving) return;

    setError("");
    setMessage("");

    const currentUser = auth.currentUser;

    if (!currentUser) {
      setError("サインインが必要です。");
      return;
    }

    if (!backendUrl) {
      setError("API base が未設定です。");
      return;
    }

    setSaving(true);

    try {
      const result = await saveUserProfile({
        currentUser,
        backendUrl,
        body: {
          lastName: lastName.trim(),
          lastNameKana: lastNameKana.trim(),
          firstName: firstName.trim(),
          firstNameKana: firstNameKana.trim(),
        },
      });

      if (!result.ok) {
        setError(result.error);
        return;
      }

      setMessage(result.message);

      navigate("/settings/shipping-address");
    } catch (error) {
      if (error instanceof Error) {
        setError(error.message);
      } else {
        setError("保存に失敗しました。");
      }
    } finally {
      setSaving(false);
    }
  }

  return {
    lastName,
    setLastName,
    lastNameKana,
    setLastNameKana,
    firstName,
    setFirstName,
    firstNameKana,
    setFirstNameKana,
    saving,
    error,
    message,
    loggedIn,
    canSave,
    clearMessages,
    goSignIn,
    signOut,
    save,
  };
}