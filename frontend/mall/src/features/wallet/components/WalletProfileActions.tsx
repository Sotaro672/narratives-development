// frontend/amol/src/features/wallet/components/WalletProfileActions.tsx

import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

import Button from "../../../components/ui/Button";
import { fetchAvatarReviews } from "../../avatar-review/api/avatarReviewApi";

type Props = {
  avatarId: string;
};

export default function WalletProfileActions({
  avatarId,
}: Props) {
  const navigate = useNavigate();

  const [goodCount, setGoodCount] = useState(0);
  const [disappointedCount, setDisappointedCount] = useState(0);
  const [reviewLoading, setReviewLoading] = useState(false);
  const [reviewError, setReviewError] = useState(false);

  useEffect(() => {
    const id = avatarId.trim();

    if (!id) {
      setGoodCount(0);
      setDisappointedCount(0);
      setReviewError(false);
      return;
    }

    let active = true;

    const loadReviewSummary = async () => {
      setReviewLoading(true);
      setReviewError(false);

      try {
        const result = await fetchAvatarReviews({
          avatarId: id,
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
  }, [avatarId]);

  const handleOpenReviews = () => {
    const id = avatarId.trim();

    if (!id) {
      return;
    }

    navigate(
      `/avatars/${encodeURIComponent(id)}/reviews`,
    );
  };

  const reviewButtonLabel = reviewLoading
    ? "評価 読み込み中"
    : reviewError
      ? "評価 - / -"
      : `良かった ${goodCount} / 残念だった ${disappointedCount}`;

  return (
    <div className="wallet-page-profile-actions-bar">
      <div className="wallet-page-profile-actions-bar__inner">
        <Button
          type="button"
          variant="secondary"
          size="sm"
          onClick={() => navigate("/avatar")}
        >
          アバター編集
        </Button>

        <Button
          type="button"
          variant="secondary"
          size="sm"
          disabled={!avatarId.trim() || reviewLoading}
          onClick={handleOpenReviews}
        >
          {reviewButtonLabel}
        </Button>
      </div>
    </div>
  );
}