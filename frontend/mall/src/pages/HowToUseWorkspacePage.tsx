// frontend/mall/src/pages/HowToUseWorkspacePage.tsx

import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type TransitionEvent,
} from "react";
import {
  Outlet,
  useLocation,
  useNavigate,
} from "react-router-dom";

import Layout from "../components/layout/Layout";
import StatePanel from "../components/ui/StatePanel";
import Tab from "../components/ui/Tab";
import {
  findHowToUseItem,
  isHowToUseCategory,
  type HowToUseItem,
} from "../features/howToUse/application/howToUseSteps";
import HowToUseListPane from "../features/howToUse/presentation/components/HowToUseListPane";

import "../styles/page-layout.css";
import "../styles/how-to-use-page.css";
import "../features/howToUse/styles/how-to-use-workspace.css";

const MOBILE_HOW_TO_USE_MEDIA_QUERY = "(max-width: 959px)";
const MOBILE_SLIDE_DURATION_MS = 320;

type MobilePane = "list" | "detail";

function isHowToUseListPath(pathname: string): boolean {
  return pathname === "/how-to-use" || pathname === "/how-to-use/";
}

function decodePathSegment(value: string): string {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}

function decodeHash(hash: string): string | null {
  const value = hash.startsWith("#") ? hash.slice(1) : hash;

  if (!value) {
    return null;
  }

  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}

function getHowToUseItemFromPath(pathname: string): HowToUseItem | undefined {
  const segments = pathname
    .split("/")
    .filter(Boolean)
    .map(decodePathSegment);

  if (segments[0] !== "how-to-use" || segments.length !== 3) {
    return undefined;
  }

  const category = segments[1];
  const slug = segments[2];

  if (!isHowToUseCategory(category) || !slug) {
    return undefined;
  }

  return findHowToUseItem(category, slug);
}

function useMobileHowToUseViewport(): boolean {
  const [isMobile, setIsMobile] = useState(() => {
    if (typeof window === "undefined") {
      return false;
    }

    return window.matchMedia(MOBILE_HOW_TO_USE_MEDIA_QUERY).matches;
  });

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    const mediaQuery = window.matchMedia(MOBILE_HOW_TO_USE_MEDIA_QUERY);

    const handleChange = (): void => {
      setIsMobile(mediaQuery.matches);
    };

    handleChange();

    if (typeof mediaQuery.addEventListener === "function") {
      mediaQuery.addEventListener("change", handleChange);
    } else {
      mediaQuery.addListener(handleChange);
    }

    return () => {
      if (typeof mediaQuery.removeEventListener === "function") {
        mediaQuery.removeEventListener("change", handleChange);
      } else {
        mediaQuery.removeListener(handleChange);
      }
    };
  }, []);

  return isMobile;
}

export default function HowToUseWorkspacePage() {
  const location = useLocation();
  const navigate = useNavigate();
  const isMobile = useMobileHowToUseViewport();

  const isHowToUseListRoute = isHowToUseListPath(location.pathname);
  const howToUseItem = getHowToUseItemFromPath(location.pathname);
  const selectedSectionId = decodeHash(location.hash);

  const [mobilePane, setMobilePane] = useState<MobilePane>(() =>
    isHowToUseListRoute ? "list" : "detail",
  );

  const pendingBackNavigationRef = useRef(false);
  const backNavigationTimerRef = useRef<number | null>(null);

  const clearBackNavigationTimer = useCallback(() => {
    if (backNavigationTimerRef.current === null) {
      return;
    }

    window.clearTimeout(backNavigationTimerRef.current);
    backNavigationTimerRef.current = null;
  }, []);

  const completeBackNavigation = useCallback(() => {
    if (!pendingBackNavigationRef.current) {
      return;
    }

    pendingBackNavigationRef.current = false;
    clearBackNavigationTimer();
    navigate("/how-to-use", { replace: true });
  }, [clearBackNavigationTimer, navigate]);

  useEffect(() => {
    if (!isMobile) {
      pendingBackNavigationRef.current = false;
      clearBackNavigationTimer();
      setMobilePane(isHowToUseListRoute ? "list" : "detail");
      return;
    }

    if (pendingBackNavigationRef.current) {
      return;
    }

    setMobilePane(isHowToUseListRoute ? "list" : "detail");
  }, [
    clearBackNavigationTimer,
    isHowToUseListRoute,
    isMobile,
  ]);

  useEffect(() => {
    return () => {
      clearBackNavigationTimer();
    };
  }, [clearBackNavigationTimer]);

  const handleBackToList = useCallback(() => {
    if (isHowToUseListRoute) {
      return;
    }

    if (!isMobile) {
      navigate("/how-to-use", { replace: true });
      return;
    }

    if (pendingBackNavigationRef.current) {
      return;
    }

    pendingBackNavigationRef.current = true;
    setMobilePane("list");
    clearBackNavigationTimer();

    const prefersReducedMotion =
      typeof window !== "undefined" &&
      window.matchMedia("(prefers-reduced-motion: reduce)").matches;

    if (prefersReducedMotion) {
      completeBackNavigation();
      return;
    }

    backNavigationTimerRef.current = window.setTimeout(
      completeBackNavigation,
      MOBILE_SLIDE_DURATION_MS,
    );
  }, [
    clearBackNavigationTimer,
    completeBackNavigation,
    isHowToUseListRoute,
    isMobile,
    navigate,
  ]);

  const handleSectionClick = useCallback(
    (sectionId: string) => {
      navigate({
        pathname: location.pathname,
        hash: `#${encodeURIComponent(sectionId)}`,
      });
    },
    [location.pathname, navigate],
  );

  const handleRailTransitionEnd = useCallback(
    (event: TransitionEvent<HTMLDivElement>) => {
      if (
        event.target !== event.currentTarget ||
        event.propertyName !== "transform" ||
        mobilePane !== "list"
      ) {
        return;
      }

      completeBackNavigation();
    },
    [completeBackNavigation, mobilePane],
  );

  const workspaceClassName = [
    "how-to-use-workspace-page",
    mobilePane === "detail"
      ? "how-to-use-workspace-page--detail"
      : "how-to-use-workspace-page--list",
  ]
    .filter(Boolean)
    .join(" ");

  const headerMobileContent =
    isMobile &&
    !isHowToUseListRoute &&
    howToUseItem ? (
      <nav
        className="how-to-use-header-nav"
        aria-label={`${howToUseItem.title}の項目`}
      >
        {howToUseItem.sections.map((section, index) => {
          const selected =
            selectedSectionId === section.id ||
            (!selectedSectionId && index === 0);

          return (
            <Tab
              key={section.id}
              variant="compact"
              selected={selected}
              aria-current={selected ? "location" : undefined}
              onClick={() => {
                handleSectionClick(section.id);
              }}
            >
              {section.title}
            </Tab>
          );
        })}
      </nav>
    ) : undefined;

  return (
    <Layout
      title="AMOL"
      mode="landing"
      hideAnnouncementButton
      hideSettingsButton={!isHowToUseListRoute}
      headerMobileContent={headerMobileContent}
      mainClassName="how-to-use-workspace-page-layout"
      disableFooterPaddingOnDesktop
      showBackButton={isMobile && !isHowToUseListRoute}
      backButtonLabel="使い方一覧に戻る"
      onBackButtonClick={handleBackToList}
    >
      <div className={workspaceClassName}>
        <div
          className="how-to-use-workspace-page__rail"
          onTransitionEnd={handleRailTransitionEnd}
        >
          <aside
            className="how-to-use-workspace-page__list"
            aria-label="使い方一覧"
            aria-hidden={
              isMobile && mobilePane === "detail"
                ? true
                : undefined
            }
          >
            <HowToUseListPane />
          </aside>

          <section
            className="how-to-use-workspace-page__detail"
            aria-label="使い方詳細"
            aria-hidden={
              isMobile && mobilePane === "list"
                ? true
                : undefined
            }
          >
            {isHowToUseListRoute ? (
              <div className="how-to-use-workspace-page__empty">
                <StatePanel
                  variant="empty"
                  title="使い方を選択してください。"
                />
              </div>
            ) : (
              <Outlet />
            )}
          </section>
        </div>
      </div>
    </Layout>
  );
}