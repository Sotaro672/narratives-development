// frontend/amol/src/features/shared/presentation/components/ChatThreadCard.tsx

import type {
  HTMLAttributes,
  ReactNode,
} from "react";

import "../../styles/chat-thread-card.css";

export type ChatThreadCardVariant =
  | "inquiry"
  | "trade"
  | "resale";

export type ChatThreadCardProps = Omit<
  HTMLAttributes<HTMLElement>,
  "children" | "className"
> & {
  children: ReactNode;
  variant?: ChatThreadCardVariant;
  className?: string;
};

function joinClassNames(
  ...classNames: Array<string | undefined | false>
): string {
  return classNames.filter(Boolean).join(" ");
}

export default function ChatThreadCard({
  children,
  variant,
  className,
  ...articleProps
}: ChatThreadCardProps) {
  return (
    <article
      {...articleProps}
      className={joinClassNames(
        "chat-thread-card",
        variant && `chat-thread-card--${variant}`,
        className,
      )}
    >
      {children}
    </article>
  );
}
