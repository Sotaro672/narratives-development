// frontend/console/shell/src/layout/Sidebar/Sidebar.tsx

import { useCallback, useEffect, useMemo, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import {
  MessageSquare,
  Box,
  Coins,
  Store,
  ShoppingCart,
  Truck,
  MessagesSquare,
  Building2,
  Wallet,
  ChevronRight,
} from "lucide-react";

import { countUnreadInquiriesHTTP } from "../../features/inquiry/infrastructure/inquiryRepositoryHTTP";
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

type SubItem = {
  label: string;
  path: string;
};

type OpenKey =
  | "products"
  | "tokens"
  | "shipping"
  | "reviews"
  | "org"
  | "finance"
  | null;

const CURRENT_COMPANY_ID_ROUTE_PLACEHOLDER = "current";
const INQUIRY_READ_STATE_CHANGED_EVENT = "inquiry:read-state-changed";

function toSafeCount(value: unknown): number | null {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return null;
  }

  const count = Math.trunc(value);
  return count > 0 ? count : null;
}

export default function Sidebar({ isOpen }: SidebarProps) {
  const navigate = useNavigate();
  const location = useLocation();

  const [inquiryUnreadCount, setInquiryUnreadCount] = useState<number | null>(null);
  const [openKey, setOpenKey] = useState<OpenKey>(null);

  const loadInquiryUnreadCount = useCallback(async () => {
    try {
      const result = await countUnreadInquiriesHTTP({
        companyId: CURRENT_COMPANY_ID_ROUTE_PLACEHOLDER,
      });

      setInquiryUnreadCount(toSafeCount(result.count));
    } catch {
      setInquiryUnreadCount(null);
    }
  }, []);

  useEffect(() => {
    void loadInquiryUnreadCount();
  }, [loadInquiryUnreadCount]);

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
      {
        label: "商品",
        path: "/product",
        icon: Box,
        hasSubmenu: true,
      },
      {
        label: "トークン",
        path: "/token",
        icon: Coins,
        hasSubmenu: true,
      },
      {
        label: "出品",
        path: "/list",
        icon: Store,
      },
      {
        label: "注文",
        path: "/order",
        icon: ShoppingCart,
      },
      {
        label: "配送",
        path: "/stockLocation",
        icon: Truck,
        hasSubmenu: true,
      },
      {
        label: "レビュー",
        path: "/review",
        icon: MessagesSquare,
        hasSubmenu: true,
      },
      {
        label: "組織",
        path: "/company",
        icon: Building2,
        hasSubmenu: true,
      },
      {
        label: "財務",
        path: "/finance",
        icon: Wallet,
        hasSubmenu: true,
      },
    ],
    [inquiryUnreadCount],
  );

  const productSubItems: SubItem[] = useMemo(
    () => [
      {
        label: "設計",
        path: "/productBlueprint",
      },
      {
        label: "生産",
        path: "/production",
      },
      {
        label: "在庫",
        path: "/inventory",
      },
    ],
    [],
  );

  const tokenSubItems: SubItem[] = useMemo(
    () => [
      {
        label: "設計",
        path: "/tokenBlueprint",
      },
      {
        label: "ミント",
        path: "/mint",
      },
      {
        label: "告知",
        path: "/sales",
      },
    ],
    [],
  );

  const shippingSubItems: SubItem[] = useMemo(
    () => [
      {
        label: "保管場所",
        path: "/stockLocation",
      },
      {
        label: "料金設計",
        path: "/transportationFee",
      },
    ],
    [],
  );

  const reviewSubItems: SubItem[] = useMemo(
    () => [
      {
        label: "商品",
        path: "/productBlueprintReview",
      },
      {
        label: "トークン",
        path: "/tokenBlueprintReview",
      },
    ],
    [],
  );

  const orgSubItems: SubItem[] = useMemo(
    () => [
      {
        label: "メンバー",
        path: "/member",
      },
      {
        label: "ブランド",
        path: "/brand",
      },
      {
        label: "権限",
        path: "/permission",
      },
    ],
    [],
  );

  const financeSubItems: SubItem[] = useMemo(
    () => [
      {
        label: "入出金履歴",
        path: "/transaction",
      },
      {
        label: "口座",
        path: "/account",
      },
    ],
    [],
  );

  useEffect(() => {
    if (location.pathname === "/" || location.pathname === "") {
      navigate("/inquiry", {
        replace: true,
      });
    }
  }, [location.pathname, navigate]);

  useEffect(() => {
    const path = location.pathname;

    if (
      path.startsWith("/review") ||
      path.startsWith("/productBlueprintReview") ||
      path.startsWith("/tokenBlueprintReview")
    ) {
      setOpenKey("reviews");
      return;
    }

    if (
      (path.startsWith("/product") ||
        path.startsWith("/productBlueprint") ||
        path.startsWith("/production") ||
        path.startsWith("/inventory")) &&
      !path.startsWith("/productBlueprintReview")
    ) {
      setOpenKey("products");
      return;
    }

    if (
      (path.startsWith("/token") ||
        path.startsWith("/tokenBlueprint") ||
        path.startsWith("/mint") ||
        path.startsWith("/sales")) &&
      !path.startsWith("/tokenBlueprintReview")
    ) {
      setOpenKey("tokens");
      return;
    }

    if (
      path.startsWith("/stockLocation") ||
      path.startsWith("/transportationFee")
    ) {
      setOpenKey("shipping");
      return;
    }

    if (
      path.startsWith("/company") ||
      path.startsWith("/member") ||
      path.startsWith("/brand") ||
      path.startsWith("/permission")
    ) {
      setOpenKey("org");
      return;
    }

    if (
      path.startsWith("/finance") ||
      path.startsWith("/transaction") ||
      path.startsWith("/account")
    ) {
      setOpenKey("finance");
      return;
    }

    setOpenKey(null);
  }, [location.pathname]);

  const toggleExclusive = (key: Exclude<OpenKey, null>) => {
    setOpenKey((current) => {
      return current === key ? null : key;
    });
  };

  const navigateAndCloseAll = (path: string) => {
    setOpenKey(null);
    navigate(path);
  };

  if (!isOpen) {
    return null;
  }

  return (
    <aside
      className="sidebar"
      style={{
        height: "calc(100vh - 103px)",
      }}
    >
      <nav className="sidebar-nav">
        {menuItems.map(({ label, path, icon: Icon, hasSubmenu, badgeCount }) => {
          const isActiveTop =
            location.pathname === path ||
            location.pathname.startsWith(`${path}/`);

          const isProductsOpen = openKey === "products";
          const isTokensOpen = openKey === "tokens";
          const isShippingOpen = openKey === "shipping";
          const isReviewsOpen = openKey === "reviews";
          const isOrgOpen = openKey === "org";
          const isFinanceOpen = openKey === "finance";

          if (label === "商品") {
            const isGroupOpen = isProductsOpen;

            return (
              <div
                key={path}
                className={`group-block ${isGroupOpen ? "group-open" : ""}`}
              >
                <button
                  type="button"
                  onClick={() => toggleExclusive("products")}
                  className={`sidebar-item parent ${isActiveTop ? "active" : ""}`}
                  aria-expanded={isGroupOpen}
                  aria-controls="submenu-products"
                >
                  <Icon className="icon-left" aria-hidden />

                  <span className="label">{label}</span>

                  <span className="right">
                    {typeof badgeCount === "number" && badgeCount > 0 ? (
                      <span className="badge">{badgeCount}</span>
                    ) : null}

                    <ChevronRight className="chevron" aria-hidden />
                  </span>
                </button>

                {isGroupOpen ? (
                  <div id="submenu-products" className="submenu-container">
                    {productSubItems.map((subItem) => {
                      const activeSub =
                        location.pathname === subItem.path ||
                        location.pathname.startsWith(`${subItem.path}/`);

                      return (
                        <button
                          key={subItem.path}
                          type="button"
                          onClick={() => navigate(subItem.path)}
                          className={`submenu-item ${activeSub ? "active" : ""}`}
                        >
                          <span className="submenu-label">{subItem.label}</span>
                        </button>
                      );
                    })}
                  </div>
                ) : null}
              </div>
            );
          }

          if (label === "トークン") {
            const isGroupOpen = isTokensOpen;

            return (
              <div
                key={path}
                className={`group-block ${isGroupOpen ? "group-open" : ""}`}
              >
                <button
                  type="button"
                  onClick={() => toggleExclusive("tokens")}
                  className={`sidebar-item parent ${isActiveTop ? "active" : ""}`}
                  aria-expanded={isGroupOpen}
                  aria-controls="submenu-tokens"
                >
                  <Icon className="icon-left" aria-hidden />

                  <span className="label">{label}</span>

                  <span className="right">
                    {typeof badgeCount === "number" && badgeCount > 0 ? (
                      <span className="badge">{badgeCount}</span>
                    ) : null}

                    <ChevronRight className="chevron" aria-hidden />
                  </span>
                </button>

                {isGroupOpen ? (
                  <div id="submenu-tokens" className="submenu-container">
                    {tokenSubItems.map((subItem) => {
                      const activeSub =
                        location.pathname === subItem.path ||
                        location.pathname.startsWith(`${subItem.path}/`);

                      return (
                        <button
                          key={subItem.path}
                          type="button"
                          onClick={() => navigate(subItem.path)}
                          className={`submenu-item ${activeSub ? "active" : ""}`}
                        >
                          <span className="submenu-label">{subItem.label}</span>
                        </button>
                      );
                    })}
                  </div>
                ) : null}
              </div>
            );
          }

          if (label === "配送") {
            const isGroupOpen = isShippingOpen;

            return (
              <div
                key={path}
                className={`group-block ${isGroupOpen ? "group-open" : ""}`}
              >
                <button
                  type="button"
                  onClick={() => toggleExclusive("shipping")}
                  className={`sidebar-item parent ${isActiveTop ? "active" : ""}`}
                  aria-expanded={isGroupOpen}
                  aria-controls="submenu-shipping"
                >
                  <Icon className="icon-left" aria-hidden />

                  <span className="label">{label}</span>

                  <span className="right">
                    {typeof badgeCount === "number" && badgeCount > 0 ? (
                      <span className="badge">{badgeCount}</span>
                    ) : null}

                    <ChevronRight className="chevron" aria-hidden />
                  </span>
                </button>

                {isGroupOpen ? (
                  <div id="submenu-shipping" className="submenu-container">
                    {shippingSubItems.map((subItem) => {
                      const activeSub =
                        location.pathname === subItem.path ||
                        location.pathname.startsWith(`${subItem.path}/`);

                      return (
                        <button
                          key={subItem.path}
                          type="button"
                          onClick={() => navigate(subItem.path)}
                          className={`submenu-item ${activeSub ? "active" : ""}`}
                        >
                          <span className="submenu-label">{subItem.label}</span>
                        </button>
                      );
                    })}
                  </div>
                ) : null}
              </div>
            );
          }

          if (label === "レビュー") {
            const isGroupOpen = isReviewsOpen;

            return (
              <div
                key={path}
                className={`group-block ${isGroupOpen ? "group-open" : ""}`}
              >
                <button
                  type="button"
                  onClick={() => toggleExclusive("reviews")}
                  className={`sidebar-item parent ${isActiveTop ? "active" : ""}`}
                  aria-expanded={isGroupOpen}
                  aria-controls="submenu-reviews"
                >
                  <Icon className="icon-left" aria-hidden />

                  <span className="label">{label}</span>

                  <span className="right">
                    {typeof badgeCount === "number" && badgeCount > 0 ? (
                      <span className="badge">{badgeCount}</span>
                    ) : null}

                    <ChevronRight className="chevron" aria-hidden />
                  </span>
                </button>

                {isGroupOpen ? (
                  <div id="submenu-reviews" className="submenu-container">
                    {reviewSubItems.map((subItem) => {
                      const activeSub =
                        location.pathname === subItem.path ||
                        location.pathname.startsWith(`${subItem.path}/`);

                      return (
                        <button
                          key={subItem.path}
                          type="button"
                          onClick={() => navigate(subItem.path)}
                          className={`submenu-item ${activeSub ? "active" : ""}`}
                        >
                          <span className="submenu-label">{subItem.label}</span>
                        </button>
                      );
                    })}
                  </div>
                ) : null}
              </div>
            );
          }

          if (label === "組織") {
            const isGroupOpen = isOrgOpen;

            return (
              <div
                key={path}
                className={`group-block ${isGroupOpen ? "group-open" : ""}`}
              >
                <button
                  type="button"
                  onClick={() => toggleExclusive("org")}
                  className={`sidebar-item parent ${isActiveTop ? "active" : ""}`}
                  aria-expanded={isGroupOpen}
                  aria-controls="submenu-org"
                >
                  <Icon className="icon-left" aria-hidden />

                  <span className="label">{label}</span>

                  <span className="right">
                    {typeof badgeCount === "number" && badgeCount > 0 ? (
                      <span className="badge">{badgeCount}</span>
                    ) : null}

                    <ChevronRight className="chevron" aria-hidden />
                  </span>
                </button>

                {isGroupOpen ? (
                  <div id="submenu-org" className="submenu-container">
                    {orgSubItems.map((subItem) => {
                      const activeSub =
                        location.pathname === subItem.path ||
                        location.pathname.startsWith(`${subItem.path}/`);

                      return (
                        <button
                          key={subItem.path}
                          type="button"
                          onClick={() => navigate(subItem.path)}
                          className={`submenu-item ${activeSub ? "active" : ""}`}
                        >
                          <span className="submenu-label">{subItem.label}</span>
                        </button>
                      );
                    })}
                  </div>
                ) : null}
              </div>
            );
          }

          if (label === "財務") {
            const isGroupOpen = isFinanceOpen;

            return (
              <div
                key={path}
                className={`group-block ${isGroupOpen ? "group-open" : ""}`}
              >
                <button
                  type="button"
                  onClick={() => toggleExclusive("finance")}
                  className={`sidebar-item parent ${isActiveTop ? "active" : ""}`}
                  aria-expanded={isGroupOpen}
                  aria-controls="submenu-finance"
                >
                  <Icon className="icon-left" aria-hidden />

                  <span className="label">{label}</span>

                  <span className="right">
                    {typeof badgeCount === "number" && badgeCount > 0 ? (
                      <span className="badge">{badgeCount}</span>
                    ) : null}

                    <ChevronRight className="chevron" aria-hidden />
                  </span>
                </button>

                {isGroupOpen ? (
                  <div id="submenu-finance" className="submenu-container">
                    {financeSubItems.map((subItem) => {
                      const activeSub =
                        location.pathname === subItem.path ||
                        location.pathname.startsWith(`${subItem.path}/`);

                      return (
                        <button
                          key={subItem.path}
                          type="button"
                          onClick={() => navigate(subItem.path)}
                          className={`submenu-item ${activeSub ? "active" : ""}`}
                        >
                          <span className="submenu-label">{subItem.label}</span>
                        </button>
                      );
                    })}
                  </div>
                ) : null}
              </div>
            );
          }

          return (
            <button
              key={path}
              type="button"
              onClick={() => navigateAndCloseAll(path)}
              className={`sidebar-item ${isActiveTop ? "active" : ""}`}
              aria-current={isActiveTop ? "page" : undefined}
            >
              <Icon className="icon-left" aria-hidden />

              <span className="label">{label}</span>

              <span className="right">
                {typeof badgeCount === "number" && badgeCount > 0 ? (
                  <span className="badge">{badgeCount}</span>
                ) : null}

                {hasSubmenu ? (
                  <ChevronRight className="chevron" aria-hidden />
                ) : null}
              </span>
            </button>
          );
        })}
      </nav>
    </aside>
  );
}