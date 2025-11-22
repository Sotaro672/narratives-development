// frontend/console/shell/src/auth/presentation/hook/useAdminPanel.ts
import { useEffect, useState, useCallback } from "react";
import { auth } from "../../infrastructure/config/firebaseClient";
import {
  fetchCurrentMember,
  updateCurrentMemberProfile,
} from "../../application/memberService";
import {
  changeEmail,
  // changePassword,  // ★ もう使わない
  sendPasswordResetForCurrentUser, // ★ 追加
} from "../../application/profileService";

// -------------------------
// かな関連ヘルパ
// -------------------------

// ひらがな・カタカナ・半角カナをひらがなに寄せる（削除はしない）
function toHiragana(input: string): string {
  if (!input) return "";

  let s = input;

  // 全角カタカナ → ひらがな
  s = s.replace(/[\u30A1-\u30F6]/g, (ch) =>
    String.fromCharCode(ch.charCodeAt(0) - 0x60),
  );

  // 半角カナ → 全角カナ → ひらがな（簡易変換）
  s = s.replace(/[\uff61-\uff9f]/g, (ch) => {
    const kataCode = ch.charCodeAt(0) - 0xff61 + 0x30a1;
    const hiraCode = kataCode - 0x60;
    return String.fromCharCode(hiraCode);
  });

  return s;
}

// 「ひらがな + スペースのみか」をチェック
function isHiraganaOnly(input: string): boolean {
  if (!input) return false;
  return /^[\u3041-\u3096\s]+$/.test(input);
}

export function useAdminPanel() {
  // dialog flags
  const [showProfileDialog, setShowProfileDialog] = useState(false);
  const [showEmailDialog, setShowEmailDialog] = useState(false);
  const [showPasswordDialog, setShowPasswordDialog] = useState(false);

  // currentMember の id
  const [memberId, setMemberId] = useState<string | null>(null);

  // profile fields
  const [lastName, setLastName] = useState("");
  const [firstName, setFirstName] = useState("");
  const [lastNameKana, setLastNameKana] = useState("");
  const [firstNameKana, setFirstNameKana] = useState("");

  // email fields
  const [newEmail, setNewEmail] = useState("");
  const [currentPasswordForEmail, setCurrentPasswordForEmail] = useState("");

  // ★ パスワード用の state は不要になったので削除
  // const [currentPassword, setCurrentPassword] = useState("");
  // const [newPassword, setNewPassword] = useState("");
  // const [confirmPassword, setConfirmPassword] = useState("");

  // ─────────────────────────────
  // currentMember を取得 & Auth email と差分があれば同期
  // ─────────────────────────────
  useEffect(() => {
    async function loadAndSyncCurrentMember() {
      const user = auth.currentUser;
      const uid = user?.uid;
      if (!uid) {
        console.warn("[useAdminPanel] no currentUser, skip loadCurrentMember");
        return;
      }

      try {
        const member = await fetchCurrentMember(uid);
        if (!member) {
          console.warn("[useAdminPanel] fetchCurrentMember returned null");
          return;
        }

        const memberIdResolved = member.id ?? uid;
        setMemberId(memberIdResolved);

        setFirstName(member.firstName ?? "");
        setLastName(member.lastName ?? "");
        setFirstNameKana(member.firstNameKana ?? "");
        setLastNameKana(member.lastNameKana ?? "");

        // Auth の email と members.email を比較して差分があれば同期
        const authEmail = user.email ?? null;
        const memberEmail = member.email ?? null;

        if (authEmail && authEmail !== memberEmail) {
          console.log(
            "[useAdminPanel] email mismatch detected. Syncing Firestore members.email",
            { authEmail, memberEmail },
          );

          await updateCurrentMemberProfile({
            id: memberIdResolved,
            firstName: member.firstName ?? "",
            lastName: member.lastName ?? "",
            firstNameKana: member.firstNameKana ?? "",
            lastNameKana: member.lastNameKana ?? "",
            email: authEmail,
          });
        }
      } catch (e) {
        console.error("[useAdminPanel] failed to load/sync currentMember:", e);
      }
    }

    void loadAndSyncCurrentMember();
  }, []);

  // ─────────────────────────────
  // プロフィール保存（名前系のみ更新）
  // ─────────────────────────────
  const saveProfile = useCallback(async () => {
    if (!memberId) {
      console.warn("[useAdminPanel] memberId is missing, skip saveProfile");
      return;
    }

    // かな入力をひらがなに正規化してからバリデーション
    const normalizedLastKana = toHiragana(lastNameKana.trim());
    const normalizedFirstKana = toHiragana(firstNameKana.trim());

    if (
      !isHiraganaOnly(normalizedLastKana) ||
      !isHiraganaOnly(normalizedFirstKana)
    ) {
      window.alert("姓・名のかなはひらがなのみで入力してください。");
      return;
    }

    try {
      await updateCurrentMemberProfile({
        id: memberId,
        firstName,
        lastName,
        firstNameKana: normalizedFirstKana,
        lastNameKana: normalizedLastKana,
      });

      // 正規化した値で state も更新しておくと UI と揃う
      setFirstNameKana(normalizedFirstKana);
      setLastNameKana(normalizedLastKana);

      setShowProfileDialog(false);
    } catch (e) {
      console.error("[useAdminPanel] failed to update profile:", e);
    }
  }, [memberId, firstName, lastName, firstNameKana, lastNameKana]);

  // ─────────────────────────────
  // メールアドレス変更（verifyBeforeUpdateEmail まで）
  // Firestore 同期は useEffect の auto sync に任せる
  // ─────────────────────────────
  const saveEmail = useCallback(async () => {
    if (!newEmail.trim()) {
      throw new Error("EMAIL_REQUIRED");
    }
    if (!currentPasswordForEmail) {
      throw new Error("PASSWORD_REQUIRED");
    }

    const normalizedEmail = newEmail.trim();
    await changeEmail(currentPasswordForEmail, normalizedEmail);

    setNewEmail("");
    setCurrentPasswordForEmail("");
  }, [newEmail, currentPasswordForEmail]);

  // ─────────────────────────────
  // ★ パスワード再設定メール送信
  //   - Firebase テンプレート「%APP_NAME% のパスワードを再設定してください」が送信される
  //   - 新しいパスワードはメール先の画面で入力
  // ─────────────────────────────
  const savePassword = useCallback(async () => {
    await sendPasswordResetForCurrentUser();
  }, []);

  return {
    // dialog flags
    showProfileDialog,
    setShowProfileDialog,
    showEmailDialog,
    setShowEmailDialog,
    showPasswordDialog,
    setShowPasswordDialog,

    // profile fields
    memberId,
    lastName,
    setLastName,
    lastNameKana,
    setLastNameKana,
    firstName,
    setFirstName,
    firstNameKana,
    setFirstNameKana,

    // email fields
    newEmail,
    setNewEmail,
    currentPasswordForEmail,
    setCurrentPasswordForEmail,

    // handlers
    saveProfile,
    saveEmail,
    savePassword, // ← これが「再設定メール送信」
  };
}
