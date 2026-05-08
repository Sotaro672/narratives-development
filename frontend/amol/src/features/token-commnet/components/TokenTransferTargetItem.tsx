// frontend/amol/src/features/token-commnet/components/TokenTransferTargetItem.tsx
import MediaIcon from "../../../components/ui/MediaIcon";
import { formatDateTime } from "../../../components/utils/date";
import type { TokenTransferTargetItemProps } from "../types/tokenTransferTypes";

function getInitial(value: string): string {
  const trimmed = value.trim();

  if (!trimmed) {
    return "?";
  }

  return trimmed.slice(0, 1).toUpperCase();
}

export default function TokenTransferTargetItem({
  target,
  selected,
  onSelect,
}: TokenTransferTargetItemProps) {
  const displayName = target.avatarName || "アバター";
  const followedAt = formatDateTime(target.followedAt);

  return (
    <button
      type="button"
      className={[
        "follow-page-user",
        selected ? "token-transfer-target-item--selected" : "",
      ]
        .filter(Boolean)
        .join(" ")}
      aria-pressed={selected}
      onClick={() => onSelect(target.avatarId)}
    >
      <span className="follow-page-user__avatar">
        <MediaIcon
          src={target.avatarIcon}
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

      {selected ? (
        <span
          className="token-transfer-target-item__selected-mark"
          aria-hidden="true"
        >
          ✓
        </span>
      ) : null}
    </button>
  );
}