// frontend/console/shell/src/layout/Header/Header.tsx
import { useEffect, useRef, useState } from "react";
import { useLocation } from "react-router-dom";
import { UserRound, ChevronDown } from "lucide-react";
import "./Header.css";
import AdminPanel from "../../auth/presentation/components/AdminPanel";
import { useAuthActions } from "../../auth/application/useAuthActions";
import { useAuth } from "../../auth/presentation/hook/useCurrentMember";
import { getCompanyNameById } from "../../auth/application/companyService";

interface HeaderProps {
  username?: string;
  email?: string;
}

export default function Header({
  username = "ログインできていません",
  email = "ログインできていません",
}: HeaderProps) {
  const [openAdmin, setOpenAdmin] = useState(false);
  const location = useLocation();

  const panelContainerRef = useRef<HTMLDivElement | null>(null);
  const triggerRef = useRef<HTMLButtonElement | null>(null);

  const { signOut } = useAuthActions();
  const { user, currentMember } = useAuth();

  const [brandName, setBrandName] = useState<string>("Company Name");

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

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpenAdmin(false);
    };

    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, []);

  useEffect(() => {
    let alive = true;

    async function run() {
      const companyId = currentMember?.companyId ?? "";
      if (!companyId) {
        if (alive) setBrandName("Company Name");
        return;
      }

      const name = await getCompanyNameById(companyId);
      if (!alive) return;

      setBrandName(name && name.length > 0 ? name : "Company Name");
    }

    run();

    return () => {
      alive = false;
    };
  }, [currentMember?.companyId, location.key]);

  const handleLogout = async () => {
    try {
      await signOut();
    } finally {
      setOpenAdmin(false);
    }
  };

  const memberName =
    `${currentMember?.lastName ?? ""} ${currentMember?.firstName ?? ""}`;

  const fullName =
    memberName ||
    user?.email ||
    username ||
    "ログインできていません";

  const displayEmail =
    currentMember?.email ||
    user?.email ||
    email ||
    "ログインできていません";

  return (
    <header className="app-header">
      <div className="brand">
        <span className="brand-main">{brandName}</span>
        <span className="brand-sub">Console</span>
      </div>

      <div className="actions">
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
            onEditProfile={() => undefined}
            onChangeEmail={() => undefined}
            onChangePassword={() => undefined}
            onLogout={handleLogout}
          />
        </div>
      </div>
    </header>
  );
}