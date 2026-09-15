//frontend\mall\src\features\howToUse\application\howToUseVideos.ts
import { getHowToUseVideoUrl } from "../infrastructure/howToUseVideoUrl";

export const howToUseVideos = {
  console: {
    inquiry: {
      responseRefund: getHowToUseVideoUrl(
        "console/inquiry/response-refund.mp4",
      ),
    },
  },

  mall: {
    refund: {
      requestRefund: getHowToUseVideoUrl(
        "mall/refund/request-refund.mp4",
      ),
    },
  },
} as const;