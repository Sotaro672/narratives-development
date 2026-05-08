// frontend/amol/src/features/token-commnet/components/TokenTransferSheet.tsx
import "../../../styles/follow-page.css";
import "../../../styles/wallet-page.css";

import TextState from "../../../components/ui/TextState";
import type {
  TokenTransferSheetProps,
  TokenTransferTargetTabKey,
} from "../types/tokenTransferTypes";
import TokenTransferTargetList from "./TokenTransferTargetList";

function TokenTransferTabs(props: {
  activeTab: TokenTransferTargetTabKey;
  followingCount: number;
  followerCount: number;
  onChange: (tab: TokenTransferTargetTabKey) => void;
}) {
  const { activeTab, followingCount, followerCount, onChange } = props;

  return (
    <div
      className="wallet-page-tabs token-transfer-sheet__tabs"
      role="tablist"
      aria-label="渡す相手の表示切替"
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

export default function TokenTransferSheet(props: TokenTransferSheetProps) {
  const {
    open,
    activeTab,
    followState,
    loading,
    refreshing,
    submitting,
    errorMessage,
    selectedTargetAvatarId,
    onClose,
    onChangeTab,
    onRefresh,
    onSelectTarget,
    onSubmit,
  } = props;

  if (!open) {
    return null;
  }

  const following = followState?.following ?? [];
  const followers = followState?.followers ?? [];
  const followingCount = followState?.followingCount ?? 0;
  const followerCount = followState?.followerCount ?? 0;

  const targets = activeTab === "followers" ? followers : following;
  const submitDisabled = !selectedTargetAvatarId || submitting || loading;

  return (
    <div className="token-transfer-sheet" role="presentation">
      <button
        type="button"
        className="token-transfer-sheet__backdrop"
        aria-label="閉じる"
        onClick={onClose}
      />

      <section
        className="token-transfer-sheet__panel"
        role="dialog"
        aria-modal="true"
        aria-label="トークンを渡す"
      >
        <div className="token-transfer-sheet__handle" aria-hidden="true" />

        <div className="token-transfer-sheet__header">
          <div>
            <p className="token-transfer-sheet__eyebrow">Token Transfer</p>
            <h2 className="token-transfer-sheet__title">渡す相手を選択</h2>
          </div>

          <button
            type="button"
            className="token-transfer-sheet__close"
            aria-label="閉じる"
            onClick={onClose}
          >
            ×
          </button>
        </div>

        <TokenTransferTabs
          activeTab={activeTab}
          followingCount={followingCount}
          followerCount={followerCount}
          onChange={onChangeTab}
        />

        <div className="token-transfer-sheet__body">
          {loading ? (
            <TextState variant="loading">
              フォロー情報を読み込んでいます...
            </TextState>
          ) : null}

          {!loading && errorMessage ? (
            <div className="token-transfer-sheet__error" role="alert">
              <TextState variant="error">{errorMessage}</TextState>

              <button
                type="button"
                className="token-transfer-sheet__retry-button"
                disabled={refreshing}
                onClick={() => void onRefresh()}
              >
                {refreshing ? "Retrying..." : "Retry"}
              </button>
            </div>
          ) : null}

          {!loading && !errorMessage ? (
            <TokenTransferTargetList
              targets={targets}
              selectedTargetAvatarId={selectedTargetAvatarId}
              emptyTitle={
                activeTab === "followers"
                  ? "No followers found"
                  : "No following users found"
              }
              emptyDescription={
                activeTab === "followers"
                  ? "このアバターにはまだフォロワーがいません。"
                  : "このアバターはまだ誰もフォローしていません。"
              }
              onSelectTarget={onSelectTarget}
            />
          ) : null}
        </div>
      </section>

      <footer className="token-transfer-sheet__footer">
        <button
          type="button"
          className="token-transfer-sheet__submit-button"
          disabled={submitDisabled}
          onClick={() => void onSubmit()}
        >
          {submitting ? "送信中..." : "この相手に渡す"}
        </button>
      </footer>
    </div>
  );
}