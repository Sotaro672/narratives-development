// frontend/mall/src/pages/AnnouncementWorkspacePage.tsx

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
import MobileSwipeRightDismissPage from "../components/layout/MobileSwipeRightDismissPage";
import StatePanel from "../components/ui/StatePanel";
import AnnouncementListPane from "../features/announcement/presentation/components/AnnouncementListPane";

import "../styles/page-layout.css";
import "../styles/announcement-page.css";
import "../features/announcement/styles/announcement-workspace.css";

const MOBILE_ANNOUNCEMENT_MEDIA_QUERY = "(max-width: 959px)";
const MOBILE_SLIDE_DURATION_MS = 320;

type MobilePane = "list" | "detail";

function isAnnouncementListPath(pathname: string): boolean {
  return pathname === "/announcements" || pathname === "/announcements/";
}

function useMobileAnnouncementViewport(): boolean {
  const [isMobile, setIsMobile] = useState(() => {
    if (typeof window === "undefined") {
      return false;
    }

    return window.matchMedia(MOBILE_ANNOUNCEMENT_MEDIA_QUERY).matches;
  });

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    const mediaQuery = window.matchMedia(MOBILE_ANNOUNCEMENT_MEDIA_QUERY);

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

export default function AnnouncementWorkspacePage() {
  const location = useLocation();
  const navigate = useNavigate();
  const isMobile = useMobileAnnouncementViewport();

  const isAnnouncementListRoute = isAnnouncementListPath(location.pathname);
  const useMobileListSwipe = isMobile && isAnnouncementListRoute;

  const [mobilePane, setMobilePane] = useState<MobilePane>(() =>
    isAnnouncementListRoute ? "list" : "detail",
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
    navigate("/announcements", { replace: true });
  }, [clearBackNavigationTimer, navigate]);

  useEffect(() => {
    if (!isMobile) {
      pendingBackNavigationRef.current = false;
      clearBackNavigationTimer();
      setMobilePane(isAnnouncementListRoute ? "list" : "detail");
      return;
    }

    if (pendingBackNavigationRef.current) {
      return;
    }

    setMobilePane(isAnnouncementListRoute ? "list" : "detail");
  }, [
    clearBackNavigationTimer,
    isAnnouncementListRoute,
    isMobile,
  ]);

  useEffect(() => {
    return () => {
      clearBackNavigationTimer();
    };
  }, [clearBackNavigationTimer]);

  const handleBackToList = useCallback(() => {
    if (isAnnouncementListRoute) {
      return;
    }

    if (!isMobile) {
      navigate("/announcements", { replace: true });
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
    isAnnouncementListRoute,
    isMobile,
    navigate,
  ]);

  const handleDismissList = useCallback(() => {
    navigate(-1);
  }, [navigate]);

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
    [
      completeBackNavigation,
      mobilePane,
    ],
  );

  const workspaceClassName = [
    "announcement-workspace-page",
    mobilePane === "detail"
      ? "announcement-workspace-page--detail"
      : "announcement-workspace-page--list",
  ]
    .filter(Boolean)
    .join(" ");

  const workspaceContent = (
    <Layout
      title="告知"
      titleClickable={false}
      showHeader={!useMobileListSwipe}
      showFooter={isMobile}
      mode="mypage"
      mainClassName="announcement-workspace-page-layout"
      disableFooterPaddingOnDesktop
      showBackButton={isMobile && !isAnnouncementListRoute}
      backButtonLabel="告知一覧に戻る"
      onBackButtonClick={handleBackToList}
      footerProps={{
        variant: "default",
      }}
    >
      <div className={workspaceClassName}>
        <div
          className="announcement-workspace-page__rail"
          onTransitionEnd={handleRailTransitionEnd}
        >
          <aside
            className="announcement-workspace-page__list"
            aria-label="告知一覧"
            aria-hidden={
              isMobile && mobilePane === "detail"
                ? true
                : undefined
            }
          >
            <AnnouncementListPane />
          </aside>

          <section
            className="announcement-workspace-page__detail"
            aria-label="告知詳細"
            aria-hidden={
              isMobile && mobilePane === "list"
                ? true
                : undefined
            }
          >
            {isAnnouncementListRoute ? (
              <div className="announcement-workspace-page__empty">
                <StatePanel
                  variant="empty"
                  title="告知を選択してください。"
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

  if (useMobileListSwipe) {
    return (
      <MobileSwipeRightDismissPage
        title="告知"
        className="announcement-workspace-page__mobile-swipe"
        onDismiss={handleDismissList}
      >
        {workspaceContent}
      </MobileSwipeRightDismissPage>
    );
  }

  return workspaceContent;
}