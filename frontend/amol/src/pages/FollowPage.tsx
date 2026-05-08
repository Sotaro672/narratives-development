// frontend/amol/src/pages/FollowPage.tsx
import { useCallback, useEffect, useMemo, useState } from "react";
import { getAuth, onAuthStateChanged } from "firebase/auth";
import type { User } from "firebase/auth";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";

import "../styles/page-layout.css";
import "../styles/follow-page.css";
import "../styles/wallet-page.css";

import Layout from "../components/layout/Layout";
import Button from "../components/ui/Button";
import MediaIcon from "../components/ui/MediaIcon";
import TextState from "../components/ui/TextState";
import { formatDateTime } from "../components/utils/date";
import { LANDING_PATH } from "../lib/navigation";

type FollowTabKey = "following" | "followers";

type FollowUser = {
  avatarId: string;
  avatarName: string;
  avatarIcon: string;
  followedAt: string;
};

type FollowResponse = {
  avatarId: string;
  followerCount: number;
  followingCount: number;
  postCount: number;
  followers: FollowUser[];
  following: FollowUser[];
  lastActiveAt: string;
  updatedAt: string;
};

const BACKEND_BASE_URL = import.meta.env.VITE_API_BASE_URL;

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function unwrapData(value: unknown): unknown {
  if (!isRecord(value)) {
    return value;
  }

  return value.data ?? value;
}

function toStringValue(value: unknown): string {
  return (value ?? "").toString().trim();
}

function toNumberValue(value: unknown): number {
  if (typeof value === "number") {
    return value;
  }

  if (typeof value === "string") {
    return Number.parseInt(value, 10) || 0;
  }

  return 0;
}

function parseFollowUser(value: unknown): FollowUser | null {
  if (!isRecord(value)) {
    return null;
  }

  const avatarId = toStringValue(value.avatarId);

  if (!avatarId) {
    return null;
  }

  return {
    avatarId,
    avatarName: toStringValue(value.avatarName),
    avatarIcon: toStringValue(value.avatarIcon),
    followedAt: toStringValue(value.followedAt),
  };
}

function parseFollowUsers(value: unknown): FollowUser[] {
  if (!Array.isArray(value)) {
    return [];
  }

  return value
    .map((item) => parseFollowUser(item))
    .filter((item): item is FollowUser => item !== null);
}

function parseFollowResponse(
  value: unknown,
  fallbackAvatarId: string,
): FollowResponse {
  const body = unwrapData(value);

  if (!isRecord(body)) {
    throw new Error("フォロー情報APIのレスポンス形式が不正です。");
  }

  return {
    avatarId: toStringValue(body.avatarId) || fallbackAvatarId,
    followerCount: toNumberValue(body.followerCount),
    followingCount: toNumberValue(body.followingCount),
    postCount: toNumberValue(body.postCount),
    followers: parseFollowUsers(body.followers),
    following: parseFollowUsers(body.following),
    lastActiveAt: toStringValue(body.lastActiveAt),
    updatedAt: toStringValue(body.updatedAt),
  };
}

function getInitialTab(value: string | null): FollowTabKey {
  return value === "followers" ? "followers" : "following";
}

function getInitial(value: string): string {
  const trimmed = value.trim();

  if (!trimmed) {
    return "?";
  }

  return trimmed.slice(0, 1).toUpperCase();
}

async function fetchAvatarFollowState(params: {
  backendUrl: string;
  idToken: string;
  avatarId: string;
}): Promise<FollowResponse> {
  const encodedAvatarId = encodeURIComponent(params.avatarId);

  const response = await fetch(
    `${params.backendUrl}/mall/avatars/${encodedAvatarId}/state`,
    {
      method: "GET",
      headers: {
        Accept: "application/json",
        Authorization: `Bearer ${params.idToken}`,
      },
    },
  );

  const contentType = response.headers.get("content-type") || "";
  const responseBody: unknown = contentType.includes("application/json")
    ? await response.json()
    : null;

  if (!response.ok) {
    if (isRecord(responseBody)) {
      const message = toStringValue(responseBody.error || responseBody.message);

      if (message) {
        throw new Error(message);
      }
    }

    throw new Error("フォロー情報の取得に失敗しました。");
  }

  return parseFollowResponse(responseBody, params.avatarId);
}

function FollowTabs(props: {
  activeTab: FollowTabKey;
  followingCount: number;
  followerCount: number;
  onChange: (tab: FollowTabKey) => void;
}) {
  const { activeTab, followingCount, followerCount, onChange } = props;

  return (
    <div
      className="wallet-page-tabs"
      role="tablist"
      aria-label="フォロー表示切替"
    >
      <button
        type="button"
        role="tab"
        aria-selected={activeTab === "following"}
        className={[
          "wallet-page-tabs__button",
          activeTab === "following" ? "wallet-page-tabs__button--active" : "",
        ]
          .filter(Boolean)
          .join(" ")}
        onClick={() => onChange("following")}
      >
        Following {followingCount}
      </button>

      <button
        type="button"
        role="tab"
        aria-selected={activeTab === "followers"}
        className={[
          "wallet-page-tabs__button",
          activeTab === "followers" ? "wallet-page-tabs__button--active" : "",
        ]
          .filter(Boolean)
          .join(" ")}
        onClick={() => onChange("followers")}
      >
        Followers {followerCount}
      </button>
    </div>
  );
}

function FollowEmpty(props: { title: string; description: string }) {
  return (
    <div className="follow-page-empty">
      <div className="follow-page-empty__icon">👥</div>

      <TextState className="follow-page-empty__title">{props.title}</TextState>

      <TextState className="follow-page-empty__description">
        {props.description}
      </TextState>
    </div>
  );
}

function FollowUserTile(props: {
  user: FollowUser;
  onAvatarTap: (avatarId: string) => void;
}) {
  const displayName = props.user.avatarName || "アバター";
  const followedAt = formatDateTime(props.user.followedAt);

  return (
    <button
      type="button"
      className="follow-page-user"
      onClick={() => props.onAvatarTap(props.user.avatarId)}
    >
      <span className="follow-page-user__avatar">
        <MediaIcon
          src={props.user.avatarIcon}
          alt={displayName}
          fallback={getInitial(displayName)}
          size="md"
          shape="circle"
          className="follow-page-user__avatar-image"
        />
      </span>

      <span className="follow-page-user__body">
        <span className="follow-page-user__name">{displayName}</span>
        <span className="follow-page-user__meta">Followed at {followedAt}</span>
      </span>
    </button>
  );
}

function FollowList(props: {
  users: FollowUser[];
  emptyTitle: string;
  emptyDescription: string;
  onAvatarTap: (avatarId: string) => void;
}) {
  if (props.users.length === 0) {
    return (
      <FollowEmpty
        title={props.emptyTitle}
        description={props.emptyDescription}
      />
    );
  }

  return (
    <div className="follow-page-list">
      {props.users.map((user) => (
        <FollowUserTile
          key={`${user.avatarId}-${user.followedAt}`}
          user={user}
          onAvatarTap={props.onAvatarTap}
        />
      ))}
    </div>
  );
}

export default function FollowPage() {
  const navigate = useNavigate();
  const { avatarId } = useParams<{ avatarId: string }>();
  const [searchParams, setSearchParams] = useSearchParams();

  const [currentUser, setCurrentUser] = useState<User | null>(null);
  const [authResolved, setAuthResolved] = useState(false);

  const [activeTab, setActiveTab] = useState<FollowTabKey>(() =>
    getInitialTab(searchParams.get("tab")),
  );
  const [response, setResponse] = useState<FollowResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");

  const resolvedAvatarId = avatarId || "";

  const following = response?.following ?? [];
  const followers = response?.followers ?? [];
  const followingCount = response?.followingCount ?? 0;
  const followerCount = response?.followerCount ?? 0;

  const pageTitle = useMemo(() => {
    return activeTab === "followers" ? "フォロワー" : "フォロー";
  }, [activeTab]);

  useEffect(() => {
    const auth = getAuth();

    const unsubscribe = onAuthStateChanged(auth, (user) => {
      setCurrentUser(user);
      setAuthResolved(true);
    });

    return () => unsubscribe();
  }, []);

  useEffect(() => {
    if (authResolved && !currentUser) {
      navigate(LANDING_PATH, { replace: true });
    }
  }, [authResolved, currentUser, navigate]);

  useEffect(() => {
    setActiveTab(getInitialTab(searchParams.get("tab")));
  }, [searchParams]);

  const load = useCallback(
    async (options?: { silent?: boolean }) => {
      if (!resolvedAvatarId) {
        setErrorMessage("アバターIDを取得できませんでした。");
        setLoading(false);
        return;
      }

      if (!authResolved || !currentUser) {
        return;
      }

      if (options?.silent) {
        setRefreshing(true);
      } else {
        setLoading(true);
      }

      setErrorMessage("");

      try {
        if (!BACKEND_BASE_URL) {
          throw new Error("VITE_API_BASE_URL is not configured.");
        }

        const idToken = await currentUser.getIdToken();

        const nextResponse = await fetchAvatarFollowState({
          backendUrl: BACKEND_BASE_URL,
          idToken,
          avatarId: resolvedAvatarId,
        });

        setResponse(nextResponse);
      } catch (error) {
        setErrorMessage(
          error instanceof Error
            ? error.message
            : "フォロー情報の取得に失敗しました。",
        );
      } finally {
        setLoading(false);
        setRefreshing(false);
      }
    },
    [authResolved, currentUser, resolvedAvatarId],
  );

  useEffect(() => {
    void load();
  }, [load]);

  const handleChangeTab = (nextTab: FollowTabKey) => {
    setActiveTab(nextTab);
    setSearchParams({ tab: nextTab });
  };

  const handleAvatarTap = (nextAvatarId: string) => {
    navigate(`/avatars/${encodeURIComponent(nextAvatarId)}`);
  };

  return (
    <Layout
      title={pageTitle}
      mode="mypage"
      showBackButton
      backTo="/wallet"
      hideSettingsButton
    >
      <section className="content-page-section follow-page">
        <div className="follow-page-layout">
          <div className="follow-page-layout__main">
            <FollowTabs
              activeTab={activeTab}
              followingCount={followingCount}
              followerCount={followerCount}
              onChange={handleChangeTab}
            />

            {loading ? (
              <TextState variant="loading" className="follow-page__message">
                読み込み中です...
              </TextState>
            ) : null}

            {!loading && errorMessage ? (
              <div role="alert" className="follow-page-error">
                <TextState variant="error" className="follow-page-error__text">
                  Failed to refresh latest follow data.
                </TextState>

                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  onClick={() => load({ silent: true })}
                  disabled={refreshing}
                >
                  {refreshing ? "Retrying..." : "Retry"}
                </Button>
              </div>
            ) : null}

            {!loading && !errorMessage && activeTab === "following" ? (
              <FollowList
                users={following}
                emptyTitle="No following users found"
                emptyDescription="This avatar is not following anyone yet."
                onAvatarTap={handleAvatarTap}
              />
            ) : null}

            {!loading && !errorMessage && activeTab === "followers" ? (
              <FollowList
                users={followers}
                emptyTitle="No followers found"
                emptyDescription="This avatar has no followers yet."
                onAvatarTap={handleAvatarTap}
              />
            ) : null}
          </div>
        </div>
      </section>
    </Layout>
  );
}