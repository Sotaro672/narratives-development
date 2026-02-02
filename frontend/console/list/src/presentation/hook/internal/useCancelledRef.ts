//frontend\console\list\src\presentation\hook\internal\useCancelledRef.ts
import * as React from "react";

/**
 * ✅ 非同期処理の完了後に setState しないための guard
 */
export function useCancelledRef() {
  const cancelledRef = React.useRef(false);

  React.useEffect(() => {
    cancelledRef.current = false;
    return () => {
      cancelledRef.current = true;
    };
  }, []);

  return cancelledRef;
}
