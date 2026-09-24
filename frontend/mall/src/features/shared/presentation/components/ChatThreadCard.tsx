// frontend/mall/src/features/shared/presentation/components/ChatThreadCard.tsx

import type { HTMLAttributes, ReactNode } from "react";

import Card from "../../../../components/ui/Card";

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

export default function ChatThreadCard({
  children,
  variant,
  className,
  ...articleProps
}: ChatThreadCardProps) {
  return (
    <Card
      as="article"
      padding="md"
      className={className}
      data-chat-variant={variant}
      {...articleProps}
    >
      {children}
    </Card>
  );
}