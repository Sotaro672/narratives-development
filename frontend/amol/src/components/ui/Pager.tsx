// frontend/amol/src/components/ui/Pager.tsx
import Button from "./Button";

import "./pager.css";

type PagerProps = {
  page: number;
  hasNext: boolean;
  busy?: boolean;
  prevLabel?: string;
  nextLabel?: string;
  onPrev: () => void;
  onNext: () => void;
  className?: string;
};

export default function Pager(props: PagerProps) {
  const {
    page,
    hasNext,
    busy = false,
    prevLabel = "前へ",
    nextLabel = "次へ",
    onPrev,
    onNext,
    className,
  } = props;

  return (
    <div className={["ui-pager", className].filter(Boolean).join(" ")}>
      <Button
        type="button"
        variant="secondary"
        disabled={page <= 1 || busy}
        onClick={onPrev}
      >
        {prevLabel}
      </Button>

      <span className="ui-pager__page">{page}</span>

      <Button
        type="button"
        variant="secondary"
        disabled={!hasNext || busy}
        onClick={onNext}
      >
        {nextLabel}
      </Button>
    </div>
  );
}