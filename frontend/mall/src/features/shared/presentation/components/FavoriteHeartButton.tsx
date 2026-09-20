// frontend/mall/src/features/shared/presentation/components/FavoriteHeartButton.tsx

import { Heart } from "lucide-react";

import IconButton from "../../../../components/ui/IconButton";

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
    <IconButton
      variant="ghost"
      size="md"
      className={isLiked ? "favorite-heart-button--liked" : ""}
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
    </IconButton>
  );
}