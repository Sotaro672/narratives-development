// frontend/console/shell/src/auth/presentation/hook/useInvitationPage.ts
import { useCallback, useRef, useState } from "react";

export function useInvitationPage() {
  // ---- フォーム ref ----
  const formRef = useRef<HTMLFormElement>(null);

  // ---- 招待トークン ----
  const [token, setToken] = useState<string>("");

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
  const [assignedBrandIds, setAssignedBrandIds] = useState<string>("");
  const [permissions, setPermissions] = useState<string>("");

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

      // TODO:
      // ここに「招待トークンを用いた会員作成フロー」を実装する
      // 1) backend: /invitation/validate(token)
      // 2) auth.createUserWithEmailAndPassword
      // 3) sendEmailVerification
      // 4) backend: /invitation/complete(token, uid,...)
      console.log({
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
    setCompanyId,
    assignedBrandIds,
    setAssignedBrandIds,
    permissions,
    setPermissions,

    // Actions
    handleBack,
    handleCreate,
    handleSubmit,
  };
}
