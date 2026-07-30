// frontend\console\shell\src\features\mint\presentation\formatter\inspectionStatusLabel.ts
import type { InspectionStatus } from "../../../../shared/types/inspections";

export function inspectionStatusLabel(
  s: InspectionStatus | null | undefined,
): string {
  switch (s) {
    case "inspecting":
      return "検査中";
    case "completed":
      return "検査完了";
    default:
      return "未検査";
  }
}
