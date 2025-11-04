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
  ChevronRight, // ← これだけ使う（回転で下向き表示）
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

// 開く可能性がある親キー
type OpenKey = "products" | "tokens" | "org" | "finance" | null;

export default function Sidebar({ isOpen }: SidebarProps) {
  const navigate = useNavigate();
  const location = useLocation();

  const openInquiriesCount = OPEN_INQUIRIES_DUMMY;

  const menuItems: MenuItem[] = useMemo(
    () => [
      { label: "問い合わせ", path: "/inquiry", icon: MessageSquare, badgeCount: openInquiriesCount > 0 ? openInquiriesCount : null },
      { label: "商品", path: "/product", icon: Box, hasSubmenu: true },
      { label: "トークン", path: "/token", icon: Coins, hasSubmenu: true },
      { label: "出品", path: "/list", icon: Store },
      { label: "注文", path: "/order", icon: ShoppingCart },
      { label: "広告", path: "/ad", icon: Target },
      { label: "組織", path: "/company", icon: Building2, hasSubmenu: true },
      { label: "財務", path: "/finance", icon: Wallet, hasSubmenu: true },
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
      { label: "運用", path: "/operation" },
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
      { label: "入出金履歴", path: "/transaction" },
      { label: "口座", path: "/account" },
    ],
    []
  );

  // ✅ 排他開閉用のキーを1つだけ保持
  const [openKey, setOpenKey] = useState<OpenKey>(null);

  // ルートに応じて自動的に該当グループを開く（他は閉じる）
  useEffect(() => {
    const p = location.pathname;
    if (
      p.startsWith("/product") ||
      p.startsWith("/productBlueprint") ||
      p.startsWith("/production") ||
      p.startsWith("/inventory")
    ) {
      setOpenKey("products");
    } else if (
      p.startsWith("/token") ||
      p.startsWith("/tokenBlueprint") ||
      p.startsWith("/mint") ||
      p.startsWith("/operation")
    ) {
      setOpenKey("tokens");
    } else if (
      p.startsWith("/company") ||
      p.startsWith("/member") ||
      p.startsWith("/brand") ||
      p.startsWith("/permission")
    ) {
      setOpenKey("org");
    } else if (
      p.startsWith("/finance") ||
      p.startsWith("/transaction") ||
      p.startsWith("/account")
    ) {
      setOpenKey("finance");
    } else {
      // それ以外（問い合わせ/出品/注文/広告など）はすべて閉じる
      setOpenKey(null);
    }
  }, [location.pathname]);

  // 親クリック時：同じキーなら閉じる、別キーならそれを開いて他は閉じる
  const toggleExclusive = (key: Exclude<OpenKey, null>) =>
    setOpenKey((curr) => (curr === key ? null : key));

  // 汎用：サブメニュー無しの行をクリックしたらすべて閉じる
  const navigateAndCloseAll = (path: string) => {
    setOpenKey(null);
    navigate(path);
  };

  if (!isOpen) return null;

  return (
    <aside className="sidebar">
      <nav className="sidebar-nav">
        {menuItems.map(({ label, path, icon: Icon, hasSubmenu, badgeCount }) => {
          const isActiveTop =
            location.pathname === path || location.pathname.startsWith(path + "/");

          // 各グループのオープン判定
          const isProductsOpen = openKey === "products";
          const isTokensOpen   = openKey === "tokens";
          const isOrgOpen      = openKey === "org";
          const isFinanceOpen  = openKey === "finance";

          // 商品
          if (label === "商品") {
            const isOpen = isProductsOpen;
            return (
              <div key={path} className={`group-block ${isOpen ? "group-open" : ""}`}>
                <button
                  type="button"
                  onClick={() => toggleExclusive("products")}
                  className={`sidebar-item parent ${isActiveTop ? "active" : ""}`}
                  aria-expanded={isOpen}
                  aria-controls="submenu-products"
                >
                  <Icon className="icon-left" aria-hidden />
                  <span className="label">{label}</span>
                  <span className="right">
                    {typeof badgeCount === "number" && badgeCount > 0 && (
                      <span className="badge">{badgeCount}</span>
                    )}
                    {/* ▶ を回転させて ▼ を表現 */}
                    <ChevronRight className="chevron" aria-hidden />
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

          // トークン
          if (label === "トークン") {
            const isOpen = isTokensOpen;
            return (
              <div key={path} className={`group-block ${isOpen ? "group-open" : ""}`}>
                <button
                  type="button"
                  onClick={() => toggleExclusive("tokens")}
                  className={`sidebar-item parent ${isActiveTop ? "active" : ""}`}
                  aria-expanded={isOpen}
                  aria-controls="submenu-tokens"
                >
                  <Icon className="icon-left" aria-hidden />
                  <span className="label">{label}</span>
                  <span className="right">
                    {typeof badgeCount === "number" && badgeCount > 0 && (
                      <span className="badge">{badgeCount}</span>
                    )}
                    <ChevronRight className="chevron" aria-hidden />
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

          // 組織
          if (label === "組織") {
            const isOpen = isOrgOpen;
            return (
              <div key={path} className={`group-block ${isOpen ? "group-open" : ""}`}>
                <button
                  type="button"
                  onClick={() => toggleExclusive("org")}
                  className={`sidebar-item parent ${isActiveTop ? "active" : ""}`}
                  aria-expanded={isOpen}
                  aria-controls="submenu-org"
                >
                  <Icon className="icon-left" aria-hidden />
                  <span className="label">{label}</span>
                  <span className="right">
                    {typeof badgeCount === "number" && badgeCount > 0 && (
                      <span className="badge">{badgeCount}</span>
                    )}
                    <ChevronRight className="chevron" aria-hidden />
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

          // 財務
          if (label === "財務") {
            const isOpen = isFinanceOpen;
            return (
              <div key={path} className={`group-block ${isOpen ? "group-open" : ""}`}>
                <button
                  type="button"
                  onClick={() => toggleExclusive("finance")}
                  className={`sidebar-item parent ${isActiveTop ? "active" : ""}`}
                  aria-expanded={isOpen}
                  aria-controls="submenu-finance"
                >
                  <Icon className="icon-left" aria-hidden />
                  <span className="label">{label}</span>
                  <span className="right">
                    {typeof badgeCount === "number" && badgeCount > 0 && (
                      <span className="badge">{badgeCount}</span>
                    )}
                    <ChevronRight className="chevron" aria-hidden />
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

          // サブメニューがない一般行
          return (
            <button
              key={path}
              onClick={() => navigateAndCloseAll(path)} // ← クリック時に全て閉じる
              className={`sidebar-item ${isActiveTop ? "active" : ""}`}
              aria-current={isActiveTop ? "page" : undefined}
            >
              <Icon className="icon-left" aria-hidden />
              <span className="label">{label}</span>
              <span className="right">
                {typeof badgeCount === "number" && badgeCount > 0 && (
                  <span className="badge">{badgeCount}</span>
                )}
                {hasSubmenu && <ChevronRight className="chevron" aria-hidden />}
              </span>
            </button>
          );
        })}
      </nav>

      <div className="sidebar-footer">
        <h2>Narratives</h2>
      </div>
    </aside>
  );
}
