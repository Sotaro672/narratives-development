// frontend/admin/shell/src/features/resale/presentation/components/ResaleStatusTab.tsx

import Tab from "../../../../shared/ui/Tab/Tab";
import {
  getResaleStatusLabel,
  getResaleStatusTone,
} from "../model/resalePresentation";

type ResaleStatusTabProps = {
  status: string;
};

export default function ResaleStatusTab({
  status,
}: ResaleStatusTabProps) {
  const label = getResaleStatusLabel(status);

  return (
    <Tab
      tone={getResaleStatusTone(status)}
      aria-label={`Resale状態 ${label}`}
    >
      {label}
    </Tab>
  );
}