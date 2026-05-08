// frontend/amol/src/features/token-commnet/components/TokenTransferTargetList.tsx
import TextState from "../../../components/ui/TextState";
import type { TokenTransferTargetListProps } from "../types/tokenTransferTypes";
import TokenTransferTargetItem from "./TokenTransferTargetItem";

export default function TokenTransferTargetList({
  targets,
  selectedTargetAvatarId,
  emptyTitle,
  emptyDescription,
  onSelectTarget,
}: TokenTransferTargetListProps) {
  if (targets.length === 0) {
    return (
      <div className="follow-page-empty">
        <div className="follow-page-empty__icon">👥</div>

        <TextState className="follow-page-empty__title">{emptyTitle}</TextState>

        <TextState className="follow-page-empty__description">
          {emptyDescription}
        </TextState>
      </div>
    );
  }

  return (
    <div className="follow-page-list">
      {targets.map((target) => (
        <TokenTransferTargetItem
          key={`${target.avatarId}-${target.followedAt}`}
          target={target}
          selected={selectedTargetAvatarId === target.avatarId}
          onSelect={onSelectTarget}
        />
      ))}
    </div>
  );
}