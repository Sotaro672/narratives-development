// frontend/amol/src/features/wallet/components/WalletProfile.tsx

import { useEffect, useState, type KeyboardEvent } from "react";
import { ThumbsDown, ThumbsUp } from "lucide-react";

import { fetchAvatarReviews } from "../../avatar-review/api/avatarReviewApi";

type WalletProfileProps = {
  avatarId: string;
  avatarName: string;
  avatarIcon: string;
  profile: string;
  isOwnAvatar: boolean;
  onClick?: () => void;
};

export default function WalletProfile({
  avatarId,
  avatarName,
  avatarIcon,
  profile,
  isOwnAvatar,
  onClick,
}: WalletProfileProps) {
  const [goodCount, setGoodCount] = useState(0);
  const [disappointedCount, setDisappointedCount] = useState(0);
  const [reviewLoading, setReviewLoading] = useState(false);
  const [reviewError, setReviewError] = useState(false);

  const normalizedAvatarId = avatarId.trim();
  const clickable = Boolean(normalizedAvatarId && onClick);
  const shouldShowProfile = !isOwnAvatar && Boolean(profile);

  useEffect(() => {
    if (!normalizedAvatarId) {
      setGoodCount(0);
      setDisappointedCount(0);
      setReviewLoading(false);
      setReviewError(false);
      return;
    }

    let active = true;

    const loadReviewSummary = async () => {
      setReviewLoading(true);
      setReviewError(false);

      try {
        const result = await fetchAvatarReviews({
          avatarId: normalizedAvatarId,
          page: 1,
          perPage: 1,
        });

        if (!active) {
          return;
        }

        setGoodCount(result.goodCount);
        setDisappointedCount(result.disappointedCount);
      } catch {
        if (!active) {
          return;
        }

        setReviewError(true);
      } finally {
        if (active) {
          setReviewLoading(false);
        }
      }
    };

    void loadReviewSummary();

    return () => {
      active = false;
    };
  }, [normalizedAvatarId]);

  const handleKeyDown = (event: KeyboardEvent<HTMLElement>) => {
    if (!clickable || !onClick) {
      return;
    }

    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      onClick();
    }
  };

  const goodLabel =
    reviewLoading || reviewError ? "-" : String(goodCount);

  const disappointedLabel =
    reviewLoading || reviewError ? "-" : String(disappointedCount);

  return (
    <section
      className={`wallet-page-profile ${
        clickable ? "wallet-page-profile--button" : ""
      }`}
      role={clickable ? "button" : undefined}
      tabIndex={clickable ? 0 : undefined}
      aria-label={
        clickable
          ? `${avatarName || "アバター"}の評価を見る`
          : undefined
      }
      onClick={clickable ? onClick : undefined}
      onKeyDown={handleKeyDown}
    >
      <div className="wallet-page-profile__avatar-area">
        <div className="wallet-page-profile__avatar-wrap">
          {avatarIcon ? (
            <img
              src={avatarIcon}
              alt={avatarName || "アバター画像"}
              className="wallet-page-profile__avatar"
            />
          ) : (
            <div className="wallet-page-profile__avatar wallet-page-profile__avatar--fallback">
              👤
            </div>
          )}
        </div>
      </div>

      <div className="wallet-page-profile__body">
        {avatarName ? (
          <div className="wallet-page-profile__name">{avatarName}</div>
        ) : null}

        <div
          className="wallet-page-profile__review-summary"
          aria-label={`良かった ${goodLabel}件、残念だった ${disappointedLabel}件`}
        >
          <span className="wallet-page-profile__review-item">
            <ThumbsUp
              className="wallet-page-profile__review-icon"
              size={18}
              strokeWidth={1.8}
              aria-hidden="true"
            />
            <span className="wallet-page-profile__review-count">
              {goodLabel}
            </span>
          </span>

          <span className="wallet-page-profile__review-item">
            <ThumbsDown
              className="wallet-page-profile__review-icon"
              size={18}
              strokeWidth={1.8}
              aria-hidden="true"
            />
            <span className="wallet-page-profile__review-count">
              {disappointedLabel}
            </span>
          </span>
        </div>

        {shouldShowProfile ? (
          <p className="wallet-page-profile__text">{profile}</p>
        ) : null}
      </div>
    </section>
  );
}