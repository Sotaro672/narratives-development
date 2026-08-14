// frontend/console/shell/src/auth/presentation/hook/useAdminPanel.ts

import { useCallback, useEffect, useState } from "react";

import { useAuthContext } from "../../application/AuthContext";
import { updateCurrentMemberProfile } from "../../application/memberService";
import {
  changeEmail,
  sendPasswordResetForCurrentUser,
} from "../../application/profileService";

function isHiraganaOnly(input: string): boolean {
  if (!input) {
    return false;
  }

  return /^[\u3041-\u3096\s]+$/.test(input);
}

export function useAdminPanel() {
  const { currentMember } = useAuthContext();

  // -------------------------
  // ダイアログ表示状態
  // -------------------------

  const [showProfileDialog, setShowProfileDialog] = useState(false);
  const [showEmailDialog, setShowEmailDialog] = useState(false);
  const [showPasswordDialog, setShowPasswordDialog] = useState(false);

  // -------------------------
  // プロフィール入力値
  // -------------------------

  const [lastName, setLastName] = useState("");
  const [firstName, setFirstName] = useState("");
  const [lastNameKana, setLastNameKana] = useState("");
  const [firstNameKana, setFirstNameKana] = useState("");

  // -------------------------
  // メールアドレス変更入力値
  // -------------------------

  const [newEmail, setNewEmail] = useState("");
  const [currentPasswordForEmail, setCurrentPasswordForEmail] = useState("");

  // -------------------------
  // Backend GET /members/me の値を
  // プロフィール入力値へ反映
  // -------------------------

  useEffect(() => {
    if (!currentMember) {
      setLastName("");
      setFirstName("");
      setLastNameKana("");
      setFirstNameKana("");
      return;
    }

    setLastName(currentMember.lastName);
    setFirstName(currentMember.firstName);
    setLastNameKana(currentMember.lastNameKana);
    setFirstNameKana(currentMember.firstNameKana);
  }, [currentMember]);

  // -------------------------
  // プロフィール保存
  // -------------------------

  const saveProfile = useCallback(async () => {
    if (!currentMember?.id) {
      throw new Error("MEMBER_NOT_FOUND");
    }

    const normalizedLastNameKana = lastNameKana.trim();
    const normalizedFirstNameKana = firstNameKana.trim();

    if (
      !isHiraganaOnly(normalizedLastNameKana) ||
      !isHiraganaOnly(normalizedFirstNameKana)
    ) {
      throw new Error("KANA_INVALID");
    }

    const updatedMember = await updateCurrentMemberProfile({
      id: currentMember.id,
      firstName,
      lastName,
      firstNameKana: normalizedFirstNameKana,
      lastNameKana: normalizedLastNameKana,
    });

    if (!updatedMember) {
      throw new Error("PROFILE_UPDATE_FAILED");
    }

    setLastName(updatedMember.lastName);
    setFirstName(updatedMember.firstName);
    setLastNameKana(updatedMember.lastNameKana);
    setFirstNameKana(updatedMember.firstNameKana);
    setShowProfileDialog(false);
  }, [
    currentMember?.id,
    firstName,
    lastName,
    firstNameKana,
    lastNameKana,
  ]);

  // -------------------------
  // メールアドレス変更
  // -------------------------

  const saveEmail = useCallback(async () => {
    const normalizedEmail = newEmail.trim();

    if (!normalizedEmail) {
      throw new Error("EMAIL_REQUIRED");
    }

    if (!currentPasswordForEmail) {
      throw new Error("PASSWORD_REQUIRED");
    }

    await changeEmail(currentPasswordForEmail, normalizedEmail);

    setNewEmail("");
    setCurrentPasswordForEmail("");
    setShowEmailDialog(false);
  }, [newEmail, currentPasswordForEmail]);

  // -------------------------
  // パスワード再設定メール送信
  // -------------------------

  const savePassword = useCallback(async () => {
    await sendPasswordResetForCurrentUser();
    setShowPasswordDialog(false);
  }, []);

  return {
    // ダイアログ
    showProfileDialog,
    setShowProfileDialog,
    showEmailDialog,
    setShowEmailDialog,
    showPasswordDialog,
    setShowPasswordDialog,

    // プロフィール
    lastName,
    setLastName,
    lastNameKana,
    setLastNameKana,
    firstName,
    setFirstName,
    firstNameKana,
    setFirstNameKana,

    // メールアドレス
    newEmail,
    setNewEmail,
    currentPasswordForEmail,
    setCurrentPasswordForEmail,

    // 保存処理
    saveProfile,
    saveEmail,
    savePassword,
  };
}