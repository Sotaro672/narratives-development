// frontend/amol/src/pages/WalletPage.tsx
import { useNavigate } from "react-router-dom";

import "../styles/page-layout.css";
import "../styles/wallet-page.css";
import "../styles/follow-page.css";

import Layout from "../components/layout/Layout";
import MediaIcon from "../components/ui/MediaIcon";
import TextState from "../components/ui/TextState";
import { formatDateTime } from "../components/utils/date";
import WalletHistoryPanel from "../features/wallet/components/WalletHistoryPanel";
import WalletProfile from "../features/wallet/components/WalletProfile";
import WalletProfileActions from "../features/wallet/components/WalletProfileActions";
import WalletTabs from "../features/wallet/components/WalletTabs";
import WalletTokenContentsCard from "../features/wallet/components/WalletTokenContentsCard";
import WalletTokenEmpty from "../features/wallet/components/WalletTokenEmpty";
import { useWalletPage } from "../features/wallet/hooks/useWalletPage";
import type { PublicWalletFollowUser } from "../features/wallet/types/followTypes";
import type { WalletTokenItem } from "../features/wallet/types/tokenTypes";

function getInitial(value: string): string {
  const trimmed = value.trim();

  if (!trimmed) {
    return "?";
  }

  return trimmed.slice(0, 1).toUpperCase();
}

function PublicFollowList(props: {
  users: PublicWalletFollowUser[];
  emptyTitle: string;
  emptyDescription: string;
  onAvatarTap: (avatarId: string) => void;
}) {
  if (props.users.length === 0) {
    return (
      <div className="follow-page-empty">
        <div className="follow-page-empty__icon">👥</div>

        <TextState className="follow-page-empty__title">
          {props.emptyTitle}
        </TextState>

        <TextState className="follow-page-empty__description">
          {props.emptyDescription}
        </TextState>
      </div>
    );
  }

  return (
    <div className="follow-page-list">
      {props.users.map((user) => {
        const displayName = user.avatarName || "アバター";
        const followedAt = formatDateTime(user.followedAt);

        return (
          <button
            key={`${user.avatarId}-${user.followedAt}`}
            type="button"
            className="follow-page-user"
            onClick={() => props.onAvatarTap(user.avatarId)}
          >
            <span className="follow-page-user__avatar">
              <MediaIcon
                src={user.avatarIcon}
                alt={displayName}
                fallback={getInitial(displayName)}
                size="md"
                shape="circle"
                className="follow-page-user__avatar-image"
              />
            </span>

            <span className="follow-page-user__body">
              <span className="follow-page-user__name">{displayName}</span>
              <span className="follow-page-user__meta">
                Followed at {followedAt}
              </span>
            </span>
          </button>
        );
      })}
    </div>
  );
}

export default function WalletPage() {
  const navigate = useNavigate();

  const {
    avatarId,
    isOwnAvatar,
    avatarName,
    avatarIcon,
    profile,
    followerCount,
    followingCount,
    walletTokens,
    orderHistory,
    activeTab,
    setActiveTab,
    loading,
    error,
    tokenLoading,
    tokenError,
    orderLoading,
    orderError,
    hasItems,
    hasTokens,
    pageTitle,
    isFollowing,
    followPosting,
    handleFollowAvatar,
    publicFollowActiveTab,
    setPublicFollowActiveTab,
    publicFollowing,
    publicFollowers,
    publicFollowLoading,
    publicFollowError,
  } = useWalletPage();

  const handleOpenContents = (token: WalletTokenItem) => {
    const params = new URLSearchParams();

    params.set("mintAddress", token.mintAddress);

    if (token.productId) params.set("productId", token.productId);
    if (token.brandId) params.set("brandId", token.brandId);
    if (token.brandName) params.set("brandName", token.brandName);
    if (token.productName) params.set("productName", token.productName);
    if (token.productBlueprintId) {
      params.set("productBlueprintId", token.productBlueprintId);
    }
    if (token.tokenBlueprintId) {
      params.set("tokenBlueprintId", token.tokenBlueprintId);
    }
    if (token.metadataUri) params.set("metadataUri", token.metadataUri);

    const tokenName = token.metadata?.name || "";
    const tokenIconUrl = token.metadata?.image || "";

    if (tokenName) params.set("tokenName", tokenName);
    if (tokenIconUrl) params.set("tokenIconUrl", tokenIconUrl);

    navigate(`/contents?${params.toString()}`);
  };

  const handleOpenAvatar = (nextAvatarId: string) => {
    navigate(`/avatars/${encodeURIComponent(nextAvatarId)}`);
  };

  const handleOpenBrand = (brandId: string) => {
    const id = brandId.trim();

    if (!id) {
      return;
    }

    navigate(`/brands/${encodeURIComponent(id)}`);
  };

  return (
    <Layout title="AMOL" mode="mypage">
      <section className="content-page-section wallet-page">
        <div className="wallet-page-layout">
          <aside className="wallet-page-layout__profile">
            <WalletProfile
              avatarId={avatarId}
              avatarName={avatarName}
              avatarIcon={avatarIcon}
              profile={profile}
              followerCount={followerCount}
              followingCount={followingCount}
              isOwnAvatar={isOwnAvatar}
            />

            <WalletProfileActions
              avatarId={avatarId}
              isOwnAvatar={isOwnAvatar}
              isFollowing={isFollowing}
              followPosting={followPosting}
              onFollowClick={handleFollowAvatar}
            />
          </aside>

          <div className="wallet-page-layout__main">
            {isOwnAvatar ? (
              <>
                <WalletTabs activeTab={activeTab} onChange={setActiveTab} />

                {activeTab === "history" ? (
                  <WalletHistoryPanel
                    loading={loading || orderLoading}
                    error={error || orderError}
                    hasItems={hasItems}
                    orderHistory={orderHistory}
                    onBrandClick={handleOpenBrand}
                  />
                ) : null}

                {activeTab === "tokens" ? (
                  <div className="wallet-page-token-list">
                    {tokenLoading ? (
                      <p className="wallet-page__message">読み込み中です...</p>
                    ) : null}

                    {!tokenLoading && tokenError ? (
                      <div role="alert" className="wallet-page__message">
                        <p>{tokenError}</p>
                      </div>
                    ) : null}

                    {!tokenLoading && !tokenError && !hasTokens ? (
                      <WalletTokenEmpty />
                    ) : null}

                    {!tokenLoading && !tokenError && hasTokens
                      ? walletTokens.map((token) => (
                          <div
                            key={token.mintAddress}
                            className="wallet-page-token-list__item"
                          >
                            <WalletTokenContentsCard
                              tokenIconUrl={token.metadata?.image || null}
                              tokenName={token.metadata?.name || ""}
                              productName={token.productName}
                              onClick={() => handleOpenContents(token)}
                            />
                          </div>
                        ))
                      : null}
                  </div>
                ) : null}
              </>
            ) : (
              <div className="wallet-page-public-follow">
                <div
                  className="wallet-page-tabs"
                  role="tablist"
                  aria-label="フォロー表示切替"
                >
                  <button
                    type="button"
                    role="tab"
                    aria-selected={publicFollowActiveTab === "following"}
                    className={[
                      "wallet-page-tabs__button",
                      publicFollowActiveTab === "following"
                        ? "wallet-page-tabs__button--active"
                        : "",
                    ]
                      .filter(Boolean)
                      .join(" ")}
                    onClick={() => setPublicFollowActiveTab("following")}
                  >
                    フォロー {followingCount}
                  </button>

                  <button
                    type="button"
                    role="tab"
                    aria-selected={publicFollowActiveTab === "followers"}
                    className={[
                      "wallet-page-tabs__button",
                      publicFollowActiveTab === "followers"
                        ? "wallet-page-tabs__button--active"
                        : "",
                    ]
                      .filter(Boolean)
                      .join(" ")}
                    onClick={() => setPublicFollowActiveTab("followers")}
                  >
                    フォロワー {followerCount}
                  </button>
                </div>

                {publicFollowLoading ? (
                  <TextState
                    variant="loading"
                    className="wallet-page__message"
                  >
                    読み込み中です...
                  </TextState>
                ) : null}

                {!publicFollowLoading && publicFollowError ? (
                  <div role="alert" className="wallet-page__message">
                    <p>{publicFollowError}</p>
                  </div>
                ) : null}

                {!publicFollowLoading &&
                !publicFollowError &&
                publicFollowActiveTab === "following" ? (
                  <PublicFollowList
                    users={publicFollowing}
                    emptyTitle="No following users found"
                    emptyDescription="This avatar is not following anyone yet."
                    onAvatarTap={handleOpenAvatar}
                  />
                ) : null}

                {!publicFollowLoading &&
                !publicFollowError &&
                publicFollowActiveTab === "followers" ? (
                  <PublicFollowList
                    users={publicFollowers}
                    emptyTitle="No followers found"
                    emptyDescription="This avatar has no followers yet."
                    onAvatarTap={handleOpenAvatar}
                  />
                ) : null}
              </div>
            )}
          </div>
        </div>
      </section>
    </Layout>
  );
}