// frontend/amol/src/features/shared/presentation/components/ChatMetaSection.tsx

import type { ReactNode } from "react";

export type ChatMetaItem = {
  label: string;
  value: ReactNode;
};

export type ChatMetaSectionProps = {
  title: string;
  items?: ChatMetaItem[] | null;
  className?: string;
};

function joinClassNames(
  ...classNames: Array<string | undefined | false>
): string {
  return classNames.filter(Boolean).join(" ");
}

export default function ChatMetaSection({
  title,
  items,
  className,
}: ChatMetaSectionProps) {
  if (!items?.length) {
    return null;
  }

  return (
    <section
      className={joinClassNames(
        "chat-detail-page__product-meta",
        className,
      )}
    >
      <h3 className="chat-detail-page__product-meta-title">
        {title}
      </h3>

      <dl className="chat-detail-page__product-meta-list">
        {items.map((item, index) => (
          <div
            key={`${item.label}-${index}`}
            className="chat-detail-page__product-meta-row"
          >
            <dt>{item.label}</dt>
            <dd>{item.value}</dd>
          </div>
        ))}
      </dl>
    </section>
  );
}