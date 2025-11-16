// frontend/console/shell/src/auth/presentation/hook/useHeader.ts
import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";

import { useAuthActions } from "../../application/useAuthActions";
import { useAuth } from "./useCurrentMember";

type UseHeaderParams = {
  username?: string;
  email?: string;
  announcementsCount?: number;
  messagesCount?: number;
};

export function useHeader({
  username = "管理者",
  email = "admin@narratives.com",
  announcementsCount = 3,
  messagesCount = 2,
}: UseHeaderParams) {
  const [openAdmin, setOpenAdmin] = useState(false);
  const navigate = useNavigate();

  const panelContainerRef = useRef<HTMLDivElement | null>(null);
  const triggerRef = useRef<HTMLButtonElement | null>(null);

  // Auth
  const { signOut } = useAuthActions();
  // useAuth から companyName / currentMember / user を受け取る
  const { user, companyName, currentMember } = useAuth();

  // ─────────────────────────────────────────────
  // 外側クリックで閉じる
  // ─────────────────────────────────────────────
  useEffect(() => {
    const onDocClick = (e: MouseEvent) => {
      const t = e.target as Node;
      if (!panelContainerRef.current) return;
      if (panelContainerRef.current.contains(t)) return;
      setOpenAdmin(false);
    };
    document.addEventListener("mousedown", onDocClick);
    return () => document.removeEventListener("mousedown", onDocClick);
  }, []);

  // ─────────────────────────────────────────────
  // Esc キーで閉じる
  // ─────────────────────────────────────────────
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpenAdmin(false);
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, []);

  // ─────────────────────────────────────────────
  // ボタン押下時の遷移処理
  // ─────────────────────────────────────────────
  const handleNotificationClick = () => {
    navigate("/announcement");
  };

  const handleMessageClick = () => {
    navigate("/message");
  };

  // ─────────────────────────────────────────────
  // ログアウト処理
  // ─────────────────────────────────────────────
  const handleLogout = async () => {
    try {
      await signOut();
      setOpenAdmin(false);
    } catch (e) {
      console.error("logout failed", e);
    }
  };

  const handleToggleAdmin = () => {
    setOpenAdmin((v) => !v);
  };

  // 表示名（companyName が取れていればそれ、なければフォールバック）
  const brandMain =
    companyName && companyName.trim().length > 0 ? companyName : "Company Name";

  // ヘッダー右上のユーザー名表示:
  // 1. currentMember.fullName（backend からの表示名）
  // 2. currentMember.lastName + firstName（fullName が無い場合）
  // 3. user.email
  // 4. props.username
  // 5. "ゲスト"
  const fullName =
    (currentMember?.fullName ?? "").trim() ||
    `${currentMember?.lastName ?? ""} ${currentMember?.firstName ?? ""}`.trim() ||
    user?.email ||
    username ||
    "ゲスト";

  // メールアドレス表示:
  // currentMember.email → user.email → props.email
  const displayEmail =
    (currentMember?.email ?? "").trim() ||
    (user?.email ?? "").trim() ||
    email;

  return {
    // state
    openAdmin,
    panelContainerRef,
    triggerRef,

    // 表示用値
    brandMain,
    fullName,
    displayEmail,
    announcementsCount,
    messagesCount,

    // handlers
    handleNotificationClick,
    handleMessageClick,
    handleToggleAdmin,
    handleLogout,
  };
}
