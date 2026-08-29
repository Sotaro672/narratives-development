// frontend/amol/src/features/resale/presentation/utils/resaleDetailFormatters.ts

import type {
  ResaleStatus,
} from "../../../shared/types/resale";

/**
 * 再販ステータスを表示用の文言へ変換する。
 */
export function formatResaleStatus(
  status: ResaleStatus,
): string {
  switch (status) {
    case "listing":
      return "出品中";
    case "suspended":
      return "公開停止";
    case "sold":
      return "売却済み";
  }
}