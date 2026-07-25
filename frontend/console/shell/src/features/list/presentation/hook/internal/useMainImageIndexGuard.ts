//frontend\console\list\src\presentation\hook\internal\useMainImageIndexGuard.ts
import * as React from "react";

/**
 * ✅ 画像配列が更新されたときに mainImageIndex を安全に保つ
 */
export function useMainImageIndexGuard(args: {
  imageUrls: string[];
  mainImageIndex: number;
  setMainImageIndex: React.Dispatch<React.SetStateAction<number>>;
}) {
  const { imageUrls, mainImageIndex, setMainImageIndex } = args;

  React.useEffect(() => {
    if (imageUrls.length === 0) {
      if (mainImageIndex !== 0) setMainImageIndex(0);
      return;
    }
    if (mainImageIndex < 0 || mainImageIndex > imageUrls.length - 1) {
      setMainImageIndex(0);
    }
  }, [imageUrls.length, mainImageIndex, setMainImageIndex]);
}
