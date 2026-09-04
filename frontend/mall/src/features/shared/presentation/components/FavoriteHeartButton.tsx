//frontend\amol\src\features\shared\presentation\components\FavoriteHeartButton.tsx
import { Heart } from "lucide-react";

import "../../styles/favorite-heart-button.css";

type FavoriteHeartButtonProps = {
  isLiked: boolean;
  disabled?: boolean;
  onClick: () => void | Promise<void>;
};

export default function FavoriteHeartButton({
  isLiked,
  disabled = false,
  onClick,
}: FavoriteHeartButtonProps) {
  const label = isLiked
    ? "お気に入りから解除"
    : "お気に入りに追加";

  return (
    <button
      type="button"
      className={`favorite-heart-button${
        isLiked ? " favorite-heart-button--liked" : ""
      }`}
      disabled={disabled}
      aria-label={label}
      aria-pressed={isLiked}
      title={label}
      onClick={onClick}
    >
      <Heart
        size={28}
        strokeWidth={2}
        fill={isLiked ? "currentColor" : "none"}
        aria-hidden="true"
      />
    </button>
  );
}