// frontend/shell/src/layout/Sidebar/Sidebar.tsx
import { useLocation, useNavigate } from "react-router-dom";

import { useEffect, useMemo, useState } from "react";
import {
  MessageSquare,
  Box,            // 商品
  Coins,          // トークン
  Store,          // 出品
  ShoppingCart,   // 注文
  Target,         // 広告
  Building2,      // 組織
  Wallet,         // 財務
  ChevronRight,
  ChevronDown,
} from "lucide-react";

// ★ 仮の未対応件数（repository を使わず固定値で表示）
const OPEN_INQUIRIES_DUMMY = 1;

import "./Sidebar.css";

interface SidebarProps {
  /** サイドバー開閉状態（モバイル対応用） */
  isOpen: boolean;
}

type MenuItem = {
  label: string;
  path: string;
  icon: React.ComponentType<React.SVGProps<SVGSVGElement>>;
  hasSubmenu?: boolean;
  badgeCount?: number | null;
};

type SubItem = { label: string; path: string };

export default function Sidebar({ isOpen }: SidebarProps) {
  const navigate = useNavigate();
  const location = useLocation();

  // ─ 未対応問い合わせ数（固定値）
  const openInquiriesCount = OPEN_INQUIRIES_DUMMY;

  // 第一階層
  const menuItems: MenuItem[] = useMemo(
    () => [
      {
        label: "問い合わせ",
        path: "/inquiries",
        icon: MessageSquare,
        badgeCount: openInquiriesCount > 0 ? openInquiriesCount : null,
      },
      { label: "商品", path: "/listings", icon: Box, hasSubmenu: true },
      { label: "トークン", path: "/mint", icon: Coins, hasSubmenu: true },
      { label: "出品", path: "/preview", icon: Store },
      { label: "注文", path: "/orders", icon: ShoppingCart },
      { label: "広告", path: "/ads", icon: Target },
      { label: "組織", path: "/org", icon: Building2, hasSubmenu: true },
      { label: "財務", path: "/accounts", icon: Wallet, hasSubmenu: true },
    ],
    [openInquiriesCount]
  );

  // サブメニュー：商品
  const productSubItems: SubItem[] = useMemo(
    () => [
      { label: "設計", path: "/design" },
      { label: "生産", path: "/production" },
      { label: "在庫", path: "/operations" }, // 将来 /inventory に差し替え可
    ],
    []
  );

  // サブメニュー：トークン
  const tokenSubItems: SubItem[] = useMemo(
    () => [
      { label: "設計", path: "/design" },
      { label: "ミント", path: "/mint" },
      { label: "運用", path: "/operations" },
    ],
    []
  );

  // サブメニュー：組織
  const orgSubItems: SubItem[] = useMemo(
    () => [
      { label: "メンバー", path: "/org/members" },
      { label: "ブランド", path: "/org/brands" },
      { label: "権限", path: "/org/roles" },
    ],
    []
  );

  // サブメニュー：財務
  const financeSubItems: SubItem[] = useMemo(
    () => [
      { label: "入出金履歴", path: "/transactions" },
      { label: "口座", path: "/accounts" },
    ],
    []
  );

  // 展開状態（ルートに応じて自動開閉）
  const [openMap, setOpenMap] = useState<Record<string, boolean>>({
    products: false,
    tokens: false,
    org: false,
    finance: true, // 初期は財務を展開
  });

  useEffect(() => {
    const p = location.pathname;

    const next: Record<string, boolean> = {
      products:
        p.startsWith("/listings") ||
        p === "/design" ||
        p.startsWith("/design/") ||
        p === "/production" ||
        p.startsWith("/production/") ||
        p === "/operations" ||
        p.startsWith("/operations/"),
      tokens:
        p.startsWith("/mint") ||
        p === "/tokens" ||
        p === "/design" ||
        p === "/operations",
      org: p.startsWith("/org"),
      finance: p.startsWith("/transactions") || p.startsWith("/accounts"),
    };

    if (Object.values(next).some(Boolean)) setOpenMap(next);
  }, [location.pathname]);

  const toggleOpen = (key: keyof typeof openMap) =>
    setOpenMap((s) => ({ ...s, [key]: !s[key] }));

  if (!isOpen) return null;

  return (
    <aside className="sidebar" style={{ marginTop: 0, paddingTop: 0 }}>
      <nav className="sidebar-nav">
        {menuItems.map(({ label, path, icon: Icon, hasSubmenu, badgeCount }) => {
          const isActiveTop =
            location.pathname === path || location.pathname.startsWith(path + "/");

          // ─ 商品
          if (label === "商品") {
            const isOpen = !!openMap.products;
            return (
              <div key={path} className={`group-block ${isOpen ? "group-open" : ""}`}>
                <button
                  type="button"
                  onClick={() => toggleOpen("products")}
                  className={`sidebar-item parent ${isActiveTop ? "active" : ""}`}
                  aria-expanded={isOpen}
                  aria-controls="submenu-products"
                >
                  <Icon className="icon-left" aria-hidden />
                  <span className="label">{label}</span>
                  <span className="right">
                    {typeof badgeCount === "number" && badgeCount > 0 && (
                      <span className="badge" aria-label={`${badgeCount}件の未読`}>
                        {badgeCount}
                      </span>
                    )}
                    {hasSubmenu &&
                      (isOpen ? (
                        <ChevronDown className="chevron" aria-hidden />
                      ) : (
                        <ChevronRight className="chevron" aria-hidden />
                      ))}
                  </span>
                </button>

                {isOpen && (
                  <div id="submenu-products" className="submenu-container">
                    {productSubItems.map((si) => {
                      const activeSub =
                        location.pathname === si.path ||
                        location.pathname.startsWith(si.path + "/");
                      return (
                        <button
                          key={si.path}
                          onClick={() => navigate(si.path)}
                          className={`submenu-item ${activeSub ? "active" : ""}`}
                        >
                          <span className="submenu-label">{si.label}</span>
                        </button>
                      );
                    })}
                  </div>
                )}
              </div>
            );
          }

          // ─ トークン
          if (label === "トークン") {
            const isOpen = !!openMap.tokens;
            return (
              <div key={path} className={`group-block ${isOpen ? "group-open" : ""}`}>
                <button
                  type="button"
                  onClick={() => toggleOpen("tokens")}
                  className={`sidebar-item parent ${isActiveTop ? "active" : ""}`}
                  aria-expanded={isOpen}
                  aria-controls="submenu-tokens"
                >
                  <Icon className="icon-left" aria-hidden />
                  <span className="label">{label}</span>
                  <span className="right">
                    {typeof badgeCount === "number" && badgeCount > 0 && (
                      <span className="badge" aria-label={`${badgeCount}件の未読`}>
                        {badgeCount}
                      </span>
                    )}
                    {hasSubmenu &&
                      (isOpen ? (
                        <ChevronDown className="chevron" aria-hidden />
                      ) : (
                        <ChevronRight className="chevron" aria-hidden />
                      ))}
                  </span>
                </button>

                {isOpen && (
                  <div id="submenu-tokens" className="submenu-container">
                    {tokenSubItems.map((si) => {
                      const activeSub =
                        location.pathname === si.path ||
                        location.pathname.startsWith(si.path + "/");
                      return (
                        <button
                          key={si.path}
                          onClick={() => navigate(si.path)}
                          className={`submenu-item ${activeSub ? "active" : ""}`}
                        >
                          <span className="submenu-label">{si.label}</span>
                        </button>
                      );
                    })}
                  </div>
                )}
              </div>
            );
          }

          // ─ 組織
          if (label === "組織") {
            const isOpen = !!openMap.org;
            return (
              <div key={path} className={`group-block ${isOpen ? "group-open" : ""}`}>
                <button
                  type="button"
                  onClick={() => toggleOpen("org")}
                  className={`sidebar-item parent ${isActiveTop ? "active" : ""}`}
                  aria-expanded={isOpen}
                  aria-controls="submenu-org"
                >
                  <Icon className="icon-left" aria-hidden />
                  <span className="label">{label}</span>
                  <span className="right">
                    {typeof badgeCount === "number" && badgeCount > 0 && (
                      <span className="badge" aria-label={`${badgeCount}件の未読`}>
                        {badgeCount}
                      </span>
                    )}
                    {hasSubmenu &&
                      (isOpen ? (
                        <ChevronDown className="chevron" aria-hidden />
                      ) : (
                        <ChevronRight className="chevron" aria-hidden />
                      ))}
                  </span>
                </button>

                {isOpen && (
                  <div id="submenu-org" className="submenu-container">
                    {orgSubItems.map((si) => {
                      const activeSub =
                        location.pathname === si.path ||
                        location.pathname.startsWith(si.path + "/");
                      return (
                        <button
                          key={si.path}
                          onClick={() => navigate(si.path)}
                          className={`submenu-item ${activeSub ? "active" : ""}`}
                        >
                          <span className="submenu-label">{si.label}</span>
                        </button>
                      );
                    })}
                  </div>
                )}
              </div>
            );
          }

          // ─ 財務
          if (label === "財務") {
            const isOpen = !!openMap.finance;
            return (
              <div key={path} className={`group-block ${isOpen ? "group-open" : ""}`}>
                <button
                  type="button"
                  onClick={() => toggleOpen("finance")}
                  className={`sidebar-item parent ${isActiveTop ? "active" : ""}`}
                  aria-expanded={isOpen}
                  aria-controls="submenu-finance"
                >
                  <Icon className="icon-left" aria-hidden />
                  <span className="label">{label}</span>
                  <span className="right">
                    {typeof badgeCount === "number" && badgeCount > 0 && (
                      <span className="badge" aria-label={`${badgeCount}件の未読`}>
                        {badgeCount}
                      </span>
                    )}
                    {hasSubmenu &&
                      (isOpen ? (
                        <ChevronDown className="chevron" aria-hidden />
                      ) : (
                        <ChevronRight className="chevron" aria-hidden />
                      ))}
                  </span>
                </button>

                {isOpen && (
                  <div id="submenu-finance" className="submenu-container">
                    {financeSubItems.map((si) => {
                      const activeSub =
                        location.pathname === si.path ||
                        location.pathname.startsWith(si.path + "/");
                      return (
                        <button
                          key={si.path}
                          onClick={() => navigate(si.path)}
                          className={`submenu-item ${activeSub ? "active" : ""}`}
                        >
                          <span className="submenu-label">{si.label}</span>
                        </button>
                      );
                    })}
                  </div>
                )}
              </div>
            );
          }

          // ─ 通常行
          return (
            <button
              key={path}
              onClick={() => navigate(path)}
              className={`sidebar-item ${isActiveTop ? "active" : ""}`}
              aria-current={isActiveTop ? "page" : undefined}
            >
              <Icon className="icon-left" aria-hidden />
              <span className="label">{label}</span>
              <span className="right">
                {typeof badgeCount === "number" && badgeCount > 0 && (
                  <span className="badge" aria-label={`${badgeCount}件の未読`}>
                    {badgeCount}
                  </span>
                )}
                {hasSubmenu && <ChevronRight className="chevron" aria-hidden />}
              </span>
            </button>
          );
        })}
      </nav>

      <div className="p-6 border-t border-border">
        <div className="flex items-center gap-3">
          <div>
            <h1 className="font-medium">Narratives</h1>
          </div>
        </div>
      </div>
    </aside>
  );
}
