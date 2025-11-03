// frontend/shell/src/layout/Sidebar/Sidebar.tsx
import { useLocation, useNavigate } from "react-router-dom";
import { useEffect, useMemo, useState } from "react";
import {
  MessageSquare,
  Box,
  Coins,
  Store,
  ShoppingCart,
  Target,
  Building2,
  Wallet,
  ChevronRight,
  ChevronDown,
} from "lucide-react";

const OPEN_INQUIRIES_DUMMY = 1;

import "./Sidebar.css";

interface SidebarProps {
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

  const openInquiriesCount = OPEN_INQUIRIES_DUMMY;

  // 第一階層（parent の path はアクティブ判定用に適当なルートを与える）
  const menuItems: MenuItem[] = useMemo(
    () => [
      { label: "問い合わせ", path: "/inquiry", icon: MessageSquare, badgeCount: openInquiriesCount > 0 ? openInquiriesCount : null },
      { label: "商品",       path: "/product",   icon: Box,       hasSubmenu: true },
      { label: "トークン",   path: "/token",     icon: Coins,     hasSubmenu: true },
      { label: "出品",       path: "/list",      icon: Store },
      { label: "注文",       path: "/order",    icon: ShoppingCart },
      { label: "広告",       path: "/ads",       icon: Target },
      { label: "組織",       path: "/company",   icon: Building2, hasSubmenu: true },
      { label: "財務",       path: "/finance",   icon: Wallet,    hasSubmenu: true },
    ],
    [openInquiriesCount]
  );

  const productSubItems: SubItem[] = useMemo(
    () => [
      { label: "設計", path: "/productBlueprint" },
      { label: "生産", path: "/production" },
      { label: "在庫", path: "/inventory" },
    ],
    []
  );

  const tokenSubItems: SubItem[] = useMemo(
    () => [
      { label: "設計", path: "/tokenBlueprint" },
      { label: "ミント", path: "/mint" },
      { label: "運用", path: "/operations" },
    ],
    []
  );

  const orgSubItems: SubItem[] = useMemo(
    () => [
      { label: "メンバー", path: "/member" },
      { label: "ブランド", path: "/brand" },
      { label: "権限", path: "/permission" },
    ],
    []
  );

  const financeSubItems: SubItem[] = useMemo(
    () => [
      { label: "入出金履歴", path: "/transactions" },
      { label: "口座",       path: "/accounts" },
    ],
    []
  );

  const [openMap, setOpenMap] = useState<Record<string, boolean>>({
    products: false,
    tokens: false,
    org: false,
    finance: true,
  });

  useEffect(() => {
    const p = location.pathname;
    const next: Record<string, boolean> = {
      products:
        p.startsWith("/product") ||
        p.startsWith("/productBlueprint") ||
        p.startsWith("/production") ||
        p.startsWith("/inventory"),
      tokens:
        p.startsWith("/token") ||
        p.startsWith("/tokenBlueprint") ||
        p.startsWith("/mint") ||
        p.startsWith("/operations"),
      org:
        p.startsWith("/company") ||
        p.startsWith("/member") ||
        p.startsWith("/brand") ||
        p.startsWith("/permission"),
      finance:
        p.startsWith("/finance") ||
        p.startsWith("/transactions") ||
        p.startsWith("/accounts"),
    };
    if (Object.values(next).some(Boolean)) setOpenMap(next);
  }, [location.pathname]);

  const toggleOpen = (key: keyof typeof openMap) =>
    setOpenMap((s) => ({ ...s, [key]: !s[key] }));

  if (!isOpen) return null;

  return (
    <aside className="sidebar">
      <nav className="sidebar-nav">
        {menuItems.map(({ label, path, icon: Icon, hasSubmenu, badgeCount }) => {
          const isActiveTop = location.pathname === path || location.pathname.startsWith(path + "/");

          if (label === "商品") {
            const isOpen = !!openMap.products;
            return (
              <div key={path} className={`group-block ${isOpen ? "group-open" : ""}`}>
                <button type="button" onClick={() => toggleOpen("products")} className={`sidebar-item parent ${isActiveTop ? "active" : ""}`} aria-expanded={isOpen} aria-controls="submenu-products">
                  <Icon className="icon-left" aria-hidden />
                  <span className="label">{label}</span>
                  <span className="right">
                    {typeof badgeCount === "number" && badgeCount > 0 && <span className="badge">{badgeCount}</span>}
                    {hasSubmenu && (isOpen ? <ChevronDown className="chevron" /> : <ChevronRight className="chevron" />)}
                  </span>
                </button>
                {isOpen && (
                  <div id="submenu-products" className="submenu-container">
                    {productSubItems.map((si) => {
                      const activeSub = location.pathname === si.path || location.pathname.startsWith(si.path + "/");
                      return (
                        <button key={si.path} onClick={() => navigate(si.path)} className={`submenu-item ${activeSub ? "active" : ""}`}>
                          <span className="submenu-label">{si.label}</span>
                        </button>
                      );
                    })}
                  </div>
                )}
              </div>
            );
          }

          if (label === "トークン") {
            const isOpen = !!openMap.tokens;
            return (
              <div key={path} className={`group-block ${isOpen ? "group-open" : ""}`}>
                <button type="button" onClick={() => toggleOpen("tokens")} className={`sidebar-item parent ${isActiveTop ? "active" : ""}`} aria-expanded={isOpen} aria-controls="submenu-tokens">
                  <Icon className="icon-left" aria-hidden />
                  <span className="label">{label}</span>
                  <span className="right">
                    {typeof badgeCount === "number" && badgeCount > 0 && <span className="badge">{badgeCount}</span>}
                    {hasSubmenu && (isOpen ? <ChevronDown className="chevron" /> : <ChevronRight className="chevron" />)}
                  </span>
                </button>
                {isOpen && (
                  <div id="submenu-tokens" className="submenu-container">
                    {tokenSubItems.map((si) => {
                      const activeSub = location.pathname === si.path || location.pathname.startsWith(si.path + "/");
                      return (
                        <button key={si.path} onClick={() => navigate(si.path)} className={`submenu-item ${activeSub ? "active" : ""}`}>
                          <span className="submenu-label">{si.label}</span>
                        </button>
                      );
                    })}
                  </div>
                )}
              </div>
            );
          }

          if (label === "組織") {
            const isOpen = !!openMap.org;
            return (
              <div key={path} className={`group-block ${isOpen ? "group-open" : ""}`}>
                <button type="button" onClick={() => toggleOpen("org")} className={`sidebar-item parent ${isActiveTop ? "active" : ""}`} aria-expanded={isOpen} aria-controls="submenu-org">
                  <Icon className="icon-left" aria-hidden />
                  <span className="label">{label}</span>
                  <span className="right">
                    {typeof badgeCount === "number" && badgeCount > 0 && <span className="badge">{badgeCount}</span>}
                    {hasSubmenu && (isOpen ? <ChevronDown className="chevron" /> : <ChevronRight className="chevron" />)}
                  </span>
                </button>
                {isOpen && (
                  <div id="submenu-org" className="submenu-container">
                    {orgSubItems.map((si) => {
                      const activeSub = location.pathname === si.path || location.pathname.startsWith(si.path + "/");
                      return (
                        <button key={si.path} onClick={() => navigate(si.path)} className={`submenu-item ${activeSub ? "active" : ""}`}>
                          <span className="submenu-label">{si.label}</span>
                        </button>
                      );
                    })}
                  </div>
                )}
              </div>
            );
          }

          if (label === "財務") {
            const isOpen = !!openMap.finance;
            return (
              <div key={path} className={`group-block ${isOpen ? "group-open" : ""}`}>
                <button type="button" onClick={() => toggleOpen("finance")} className={`sidebar-item parent ${isActiveTop ? "active" : ""}`} aria-expanded={isOpen} aria-controls="submenu-finance">
                  <Icon className="icon-left" aria-hidden />
                  <span className="label">{label}</span>
                  <span className="right">
                    {typeof badgeCount === "number" && badgeCount > 0 && <span className="badge">{badgeCount}</span>}
                    {hasSubmenu && (isOpen ? <ChevronDown className="chevron" /> : <ChevronRight className="chevron" />)}
                  </span>
                </button>
                {isOpen && (
                  <div id="submenu-finance" className="submenu-container">
                    {financeSubItems.map((si) => {
                      const activeSub = location.pathname === si.path || location.pathname.startsWith(si.path + "/");
                      return (
                        <button key={si.path} onClick={() => navigate(si.path)} className={`submenu-item ${activeSub ? "active" : ""}`}>
                          <span className="submenu-label">{si.label}</span>
                        </button>
                      );
                    })}
                  </div>
                )}
              </div>
            );
          }

          return (
            <button key={path} onClick={() => navigate(path)} className={`sidebar-item ${isActiveTop ? "active" : ""}`} aria-current={isActiveTop ? "page" : undefined}>
              <Icon className="icon-left" aria-hidden />
              <span className="label">{label}</span>
              <span className="right">
                {typeof badgeCount === "number" && badgeCount > 0 && <span className="badge">{badgeCount}</span>}
                {hasSubmenu && <ChevronRight className="chevron" aria-hidden />}
              </span>
            </button>
          );
        })}
      </nav>

      {/* フッター（CSS .sidebar-footer と一致） */}
      <div className="sidebar-footer">
        <div className="flex items-center gap-3">
          <h1 className="font-medium">Narratives</h1>
        </div>
      </div>
    </aside>
  );
}
