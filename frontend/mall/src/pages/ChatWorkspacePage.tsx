// frontend/mall/src/pages/ChatWorkspacePage.tsx

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
import ChatListPane from "../features/inquiry/presentation/components/ChatListPane";
import {
  ChatWorkspaceProvider,
  useChatWorkspace,
} from "../features/shared/presentation/context/ChatWorkspaceContext";

import "../styles/page-layout.css";
import "../features/inquiry/presentation/styles/inquiry-list-page.css";
import "../features/shared/styles/chat-workspace.css";

const MOBILE_CHAT_MEDIA_QUERY = "(max-width: 959px)";
const MOBILE_SLIDE_DURATION_MS = 320;

type MobilePane = "list" | "detail";

function isChatListPath(pathname: string): boolean {
  return pathname === "/chats" || pathname === "/chats/";
}

function useMobileChatViewport(): boolean {
  const [isMobile, setIsMobile] = useState(() => {
    if (typeof window === "undefined") {
      return false;
    }

    return window.matchMedia(MOBILE_CHAT_MEDIA_QUERY).matches;
  });

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    const mediaQuery = window.matchMedia(MOBILE_CHAT_MEDIA_QUERY);

    const handleChange = () => {
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

export default function ChatWorkspacePage() {
  return (
    <ChatWorkspaceProvider>
      <ChatWorkspaceContent />
    </ChatWorkspaceProvider>
  );
}

function ChatWorkspaceContent() {
  const location = useLocation();
  const navigate = useNavigate();
  const isMobile = useMobileChatViewport();
  const { action } = useChatWorkspace();

  const isChatListRoute = isChatListPath(location.pathname);
  const [mobilePane, setMobilePane] = useState<MobilePane>(() =>
    isChatListRoute ? "list" : "detail",
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
    navigate("/chats", { replace: true });
  }, [clearBackNavigationTimer, navigate]);

  useEffect(() => {
    if (!isMobile) {
      pendingBackNavigationRef.current = false;
      clearBackNavigationTimer();
      setMobilePane(isChatListRoute ? "list" : "detail");
      return;
    }

    if (pendingBackNavigationRef.current) {
      return;
    }

    setMobilePane(isChatListRoute ? "list" : "detail");
  }, [
    clearBackNavigationTimer,
    isChatListRoute,
    isMobile,
  ]);

  useEffect(() => {
    return () => {
      clearBackNavigationTimer();
    };
  }, [clearBackNavigationTimer]);

  const handleBackToList = useCallback(() => {
    if (isChatListRoute) {
      return;
    }

    if (!isMobile) {
      navigate("/chats", { replace: true });
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
    isChatListRoute,
    isMobile,
    navigate,
  ]);

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
    "chat-workspace-page",
    mobilePane === "detail"
      ? "chat-workspace-page--detail"
      : "chat-workspace-page--list",
  ]
    .filter(Boolean)
    .join(" ");

  const hasMobileDetailAction =
    isMobile &&
    !isChatListRoute &&
    action !== null &&
    action.label !== "";

  return (
    <Layout
      title="AMOL"
      showFooter={isMobile}
      mode="mypage"
      mainClassName="chat-workspace-page-layout"
      disableFooterPaddingOnDesktop
      showBackButton={isMobile && !isChatListRoute}
      backButtonLabel="チャット一覧に戻る"
      onBackButtonClick={handleBackToList}
      footerProps={
        hasMobileDetailAction
          ? {
              variant: "default",
              centerActionLabel: action.label,
              centerActionDisabled: action.disabled ?? false,
              onCenterActionClick: action.onClick,
            }
          : {
              variant: "default",
            }
      }
    >
      <div className={workspaceClassName}>
        <div
          className="chat-workspace-page__rail"
          onTransitionEnd={handleRailTransitionEnd}
        >
          <aside
            className="chat-workspace-page__list"
            aria-label="チャット一覧"
            aria-hidden={
              isMobile && mobilePane === "detail"
                ? true
                : undefined
            }
          >
            <ChatListPane />
          </aside>

          <section
            className="chat-workspace-page__detail"
            aria-label="チャット詳細"
            aria-hidden={
              isMobile && mobilePane === "list"
                ? true
                : undefined
            }
          >
            {isChatListRoute ? (
              <div className="chat-workspace-page__empty">
                <StatePanel
                  variant="empty"
                  title="チャットを選択してください。"
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