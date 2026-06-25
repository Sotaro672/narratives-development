// frontend/shell/src/layout/Sidebar/Sidebar.tsx

import { useLocation, useNavigate } from "react-router-dom";
import { useCallback, useEffect, useMemo, useState } from "react";
import {
  MessageSquare,
  Box,
  Coins,
  Store,
  ShoppingCart,
  MessagesSquare,
  Building2,
  Wallet,
  ChevronRight,
} from "lucide-react";

import { buildConsoleUrl } from "../../shared/http/apiBase";
import { getAuthHeaders } from "../../shared/http/authHeaders";
import { fetchJSON } from "../../shared/http/fetchJSON";

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

type OpenKey = "products" | "tokens" | "reviews" | "org" | "finance" | null;

type InquiryUnreadCountResponse = {
  count?: number | null;
};

const CURRENT_COMPANY_ID_ROUTE_PLACEHOLDER = "current";
const INQUIRY_READ_STATE_CHANGED_EVENT = "inquiry:read-state-changed";

function toSafeCount(value: unknown): number | null {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return null;
  }

  const count = Math.trunc(value);

  return count > 0 ? count : null;
}

function isInquiryDetailPath(pathname: string): boolean {
  const normalized = String(pathname ?? "").trim();

  if (!normalized.startsWith("/inquiry/")) {
    return false;
  }

  const id = normalized.replace(/^\/inquiry\//, "").split("/")[0];

  return id !== "";
}

async function fetchInquiryUnreadCount(): Promise<number | null> {
  const headers = await getAuthHeaders();

  const url = buildConsoleUrl(
    `/inquiries/company/${encodeURIComponent(
      CURRENT_COMPANY_ID_ROUTE_PLACEHOLDER,
    )}/unread-count`,
  );

  const data = await fetchJSON<InquiryUnreadCountResponse>(url, {
    method: "GET",
    headers,
  });

  return toSafeCount(data.count);
}

export default function Sidebar({ isOpen }: SidebarProps) {
  const navigate = useNavigate();
  const location = useLocation();

  const [inquiryUnreadCount, setInquiryUnreadCount] = useState<number | null>(
    null,
  );

  const loadInquiryUnreadCount = useCallback(async () => {
    try {
      const count = await fetchInquiryUnreadCount();
      setInquiryUnreadCount(count);
    } catch {
      setInquiryUnreadCount(null);
    }
  }, []);

  useEffect(() => {
    let active = true;
    let timer: number | null = null;

    async function load() {
      try {
        const count = await fetchInquiryUnreadCount();
        if (!active) return;

        setInquiryUnreadCount(count);
      } catch {
        if (!active) return;

        setInquiryUnreadCount(null);
      }
    }

    void load();

    if (isInquiryDetailPath(location.pathname)) {
      timer = window.setTimeout(() => {
        if (!active) return;

        void load();
      }, 400);
    }

    return () => {
      active = false;

      if (timer !== null) {
        window.clearTimeout(timer);
      }
    };
  }, [location.pathname]);

  useEffect(() => {
    const refresh = () => {
      void loadInquiryUnreadCount();
    };

    window.addEventListener("focus", refresh);
    window.addEventListener(INQUIRY_READ_STATE_CHANGED_EVENT, refresh);

    return () => {
      window.removeEventListener("focus", refresh);
      window.removeEventListener(INQUIRY_READ_STATE_CHANGED_EVENT, refresh);
    };
  }, [loadInquiryUnreadCount]);

  const menuItems: MenuItem[] = useMemo(
    () => [
      {
        label: "問い合わせ",
        path: "/inquiry",
        icon: MessageSquare,
        badgeCount: inquiryUnreadCount,
      },
      { label: "商品", path: "/product", icon: Box, hasSubmenu: true },
      { label: "トークン", path: "/token", icon: Coins, hasSubmenu: true },
      { label: "出品", path: "/list", icon: Store },
      { label: "注文", path: "/order", icon: ShoppingCart },
      { label: "レビュー", path: "/review", icon: MessagesSquare, hasSubmenu: true },
      { label: "組織", path: "/company", icon: Building2, hasSubmenu: true },
      { label: "財務", path: "/finance", icon: Wallet, hasSubmenu: true },
    ],
    [inquiryUnreadCount],
  );

  const productSubItems: SubItem[] = useMemo(
    () => [
      { label: "設計", path: "/productBlueprint" },
      { label: "生産", path: "/production" },
      { label: "在庫", path: "/inventory" },
    ],
    [],
  );

  const tokenSubItems: SubItem[] = useMemo(
    () => [
      { label: "設計", path: "/tokenBlueprint" },
      { label: "ミント", path: "/mintRequest" },
      { label: "告知", path: "/sales" },
    ],
    [],
  );

  const reviewSubItems: SubItem[] = useMemo(
    () => [
      { label: "商品", path: "/productBlueprintReview" },
      { label: "トークン", path: "/tokenBlueprintReview" },
    ],
    [],
  );

  const orgSubItems: SubItem[] = useMemo(
    () => [
      { label: "メンバー", path: "/member" },
      { label: "ブランド", path: "/brand" },
      { label: "権限", path: "/permission" },
    ],
    [],
  );

  const financeSubItems: SubItem[] = useMemo(
    () => [
      { label: "入出金履歴", path: "/transaction" },
      { label: "口座", path: "/account" },
    ],
    [],
  );

  const [openKey, setOpenKey] = useState<OpenKey>(null);

  useEffect(() => {
    if (location.pathname === "/" || location.pathname === "") {
      navigate("/inquiry", { replace: true });
    }
  }, [location.pathname, navigate]);

  useEffect(() => {
    const p = location.pathname;

    if (
      p.startsWith("/review") ||
      p.startsWith("/productBlueprintReview") ||
      p.startsWith("/tokenBlueprintReview")
    ) {
      setOpenKey("reviews");
      return;
    }

    if (
      (p.startsWith("/product") ||
        p.startsWith("/productBlueprint") ||
        p.startsWith("/production") ||
        p.startsWith("/inventory")) &&
      !p.startsWith("/productBlueprintReview")
    ) {
      setOpenKey("products");
    } else if (
      (p.startsWith("/token") ||
        p.startsWith("/tokenBlueprint") ||
        p.startsWith("/mintRequest") ||
        p.startsWith("/sales")) &&
      !p.startsWith("/tokenBlueprintReview")
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
      setOpenKey(null);
    }
  }, [location.pathname]);

  const toggleExclusive = (key: Exclude<OpenKey, null>) =>
    setOpenKey((curr) => (curr === key ? null : key));

  const navigateAndCloseAll = (path: string) => {
    setOpenKey(null);
    navigate(path);
  };

  if (!isOpen) return null;

  return (
    <aside className="sidebar" style={{ height: "calc(100vh - 103px)" }}>
      <nav className="sidebar-nav">
        {menuItems.map(({ label, path, icon: Icon, hasSubmenu, badgeCount }) => {
          const isActiveTop =
            location.pathname === path || location.pathname.startsWith(path + "/");

          const isProductsOpen = openKey === "products";
          const isTokensOpen = openKey === "tokens";
          const isReviewsOpen = openKey === "reviews";
          const isOrgOpen = openKey === "org";
          const isFinanceOpen = openKey === "finance";

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

          if (label === "レビュー") {
            const isOpen = isReviewsOpen;
            return (
              <div key={path} className={`group-block ${isOpen ? "group-open" : ""}`}>
                <button
                  type="button"
                  onClick={() => toggleExclusive("reviews")}
                  className={`sidebar-item parent ${isActiveTop ? "active" : ""}`}
                  aria-expanded={isOpen}
                  aria-controls="submenu-reviews"
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
                  <div id="submenu-reviews" className="submenu-container">
                    {reviewSubItems.map((si) => {
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

          return (
            <button
              key={path}
              onClick={() => navigateAndCloseAll(path)}
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
    </aside>
  );
}