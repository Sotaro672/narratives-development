// frontend/console/shell/src/features/mint/presentation/formatter/inspectionStatusLabel.ts

import type {
  InspectionStatus,
} from "../../../../shared/types/inspections";

export type InspectionStatusLabelValue =
  | InspectionStatus
  | "notYet"
  | null
  | undefined;

export function inspectionStatusLabel(
  status: InspectionStatusLabelValue,
): string {
  switch (status) {
    case "inspecting":
      return "検査中";

    case "completed":
      return "検査完了";

    case "notYet":
    case null:
    case undefined:
      return "未検査";
  }
}