// frontend/console/shell/src/layout/Header/Header.tsx
import { useEffect, useRef, useState } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { Bell, MessageSquare, UserRound, ChevronDown } from "lucide-react";
import "./Header.css";
import AdminPanel from "../../auth/presentation/components/AdminPanel";
import { useAuthActions } from "../../auth/application/useAuthActions";
import { useAuth } from "../../auth/presentation/hook/useCurrentMember";
import { getCompanyNameById } from "../../auth/application/companyService";

interface HeaderProps {
  username?: string;
  email?: string;
  announcementsCount?: number;
  messagesCount?: number;
}

export default function Header({
  username = "管理者",
  email = "admin@narratives.com",
  announcementsCount = 3,
  messagesCount = 2,
}: HeaderProps) {
  const [openAdmin, setOpenAdmin] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();

  const panelContainerRef = useRef<HTMLDivElement | null>(null);
  const triggerRef = useRef<HTMLButtonElement | null>(null);

  // Auth
  const { signOut } = useAuthActions();
  // companyName は信用せず、currentMember をソースにする
  const { user, currentMember } = useAuth();

  // Header 表示用の companyName（毎回 fresh に更新）
  const [brandName, setBrandName] = useState<string>("Company Name");

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
  // ルートアクセス（location.key） or companyId 変化の度に、必ず fresh 取得
  // ─────────────────────────────────────────────
  useEffect(() => {
    let alive = true;

    async function run() {
      const companyId = (currentMember?.companyId ?? "").trim();
      if (!companyId) {
        if (alive) setBrandName("Company Name");
        return;
      }

      const name = await getCompanyNameById(companyId); // fresh（キャッシュ無し）
      if (!alive) return;

      setBrandName(name && name.trim().length > 0 ? name : "Company Name");
    }

    run();

    return () => {
      alive = false;
    };
  }, [currentMember?.companyId, location.key]);

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

  // ヘッダー右上のユーザー名表示
  const fullName =
    (currentMember?.fullName ?? "").trim() ||
    `${currentMember?.lastName ?? ""} ${currentMember?.firstName ?? ""}`.trim() ||
    user?.email ||
    username ||
    "ゲスト";

  // メールアドレス表示
  const displayEmail =
    (currentMember?.email ?? "").trim() ||
    (user?.email ?? "").trim() ||
    email;

  return (
    <header className="app-header">
      {/* Left: Brand */}
      <div className="brand">
        <span className="brand-main">{brandName}</span>
        <span className="brand-sub">Console</span>
      </div>

      {/* Right: Actions */}
      <div className="actions">
        <button
          className="icon-btn"
          aria-label="通知"
          onClick={handleNotificationClick}
        >
          <span className="icon-wrap">
            <Bell className="icon" aria-hidden />
            {announcementsCount > 0 && (
              <span className="badge" aria-label={`${announcementsCount}件の通知`}>
                {announcementsCount}
              </span>
            )}
          </span>
        </button>

        <button
          className="icon-btn"
          aria-label="メッセージ"
          onClick={handleMessageClick}
        >
          <span className="icon-wrap">
            <MessageSquare className="icon" aria-hidden />
            {messagesCount > 0 && (
              <span className="badge" aria-label={`${messagesCount}件の新着メッセージ`}>
                {messagesCount}
              </span>
            )}
          </span>
        </button>

        <div className="relative" ref={panelContainerRef}>
          <button
            ref={triggerRef}
            className="icon-btn user-trigger"
            aria-haspopup="menu"
            aria-expanded={openAdmin}
            aria-controls="admin-dropdown"
            onClick={() => setOpenAdmin((v) => !v)}
          >
            <UserRound className="icon" aria-hidden />
            <ChevronDown
              className={`caret ${openAdmin ? "open" : ""}`}
              aria-hidden
            />
          </button>

          <AdminPanel
            open={openAdmin}
            fullName={fullName}
            email={displayEmail}
            onEditProfile={() => console.log("プロフィール変更")}
            onChangeEmail={() => console.log("メールアドレス変更")}
            onChangePassword={() => console.log("パスワード変更")}
            onLogout={handleLogout}
          />
        </div>
      </div>
    </header>
  );
}