// frontend/amol/src/features/shared/presentation/components/ChatMetaSection.tsx

import type { ReactNode } from "react";

import Card from "../../../../components/ui/Card";
import InfoList from "../../../../components/ui/InfoList";
import SectionHeader from "../../../../components/ui/SectionHeader";

export type ChatMetaItem = {
  label: string;
  value: ReactNode;
};

export type ChatMetaSectionProps = {
  title: string;
  items?: ChatMetaItem[] | null;
  className?: string;
};

export default function ChatMetaSection({
  title,
  items,
  className,
}: ChatMetaSectionProps) {
  if (!items?.length) {
    return null;
  }

  return (
    <Card
      as="section"
      variant="panel"
      padding="sm"
      className={className}
    >
      <SectionHeader
        title={title}
        titleAs="h3"
        className="ui-section-header--title-sm"
      />

      <InfoList
        rows={items.map((item, index) => ({
          key: `${item.label}-${index}`,
          label: item.label,
          value: item.value,
        }))}
      />
    </Card>
  );
}