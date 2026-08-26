// frontend/amol/src/features/order/hooks/useOrderDetail.ts    
    
import {    
  useCallback,    
  useEffect,    
  useRef,    
  useState,    
} from "react";    
import { useParams } from "react-router-dom";    
    
import { getApiBaseUrl } from "../../../lib/apiBaseUrl";    
import { getFirebaseIdToken } from "../../../lib/authToken";    
    
import {    
  cancelOrderItem,    
  fetchOrderDetail,    
  returnOrderItem,    
} from "../api/orderDetailApi";    
    
import type {    
  OrderDetail,    
} from "../../shared/types/orderDetailTypes";    
    
export type ReturnPackageState =    
  | "unopened"    
  | "opened";    
    
function getErrorMessage(    
  caught: unknown,    
  defaultMessage: string,    
): string {    
  return caught instanceof Error    
    ? caught.message    
    : defaultMessage;    
}    
    
export function useOrderDetail() {    
  const {    
    orderId: routeOrderId,    
  } = useParams<{    
    orderId: string;    
  }>();    
    
  const orderId =    
    routeOrderId?.trim() || "";    
    
  const [order, setOrder] =    
    useState<OrderDetail | null>(null);    
    
  const [loading, setLoading] =    
    useState(true);    
    
  const [    
    cancellingItemIndex,    
    setCancellingItemIndex,    
  ] = useState<number | null>(null);    
    
  const [    
    returningItemIndex,    
    setReturningItemIndex,    
  ] = useState<number | null>(null);    
    
  const [error, setError] =    
    useState("");    
    
  const requestIdRef =    
    useRef(0);    
    
  const loadOrder = useCallback(    
    async () => {    
      const requestId =    
        ++requestIdRef.current;    
    
      setLoading(true);    
      setError("");    
    
      if (!orderId) {    
        setOrder(null);    
        setError(    
          "注文IDが指定されていません。",    
        );    
        setLoading(false);    
    
        return;    
      }    
    
      try {    
        const backendUrl =    
          getApiBaseUrl();    
    
        if (!backendUrl) {    
          throw new Error(    
            "VITE_API_BASE_URLが設定されていません。",    
          );    
        }    
    
        const idToken =    
          await getFirebaseIdToken();    
    
        const nextOrder =    
          await fetchOrderDetail({    
            backendUrl,    
            idToken,    
            orderId,    
          });    
    
        if (    
          requestIdRef.current !==    
          requestId    
        ) {    
          return;    
        }    
    
        setOrder(nextOrder);    
        setError("");    
      } catch (caught) {    
        if (    
          requestIdRef.current !==    
          requestId    
        ) {    
          return;    
        }    
    
        setOrder(null);    
        setError(    
          getErrorMessage(    
            caught,    
            "注文情報の取得に失敗しました。",    
          ),    
        );    
      } finally {    
        if (    
          requestIdRef.current ===    
          requestId    
        ) {    
          setLoading(false);    
        }    
      }    
    },    
    [    
      orderId,    
    ],    
  );    
    
  useEffect(() => {    
    void loadOrder();    
    
    return () => {    
      requestIdRef.current += 1;    
    };    
  }, [    
    loadOrder,    
  ]);    
    
  const reload = useCallback(    
    async () => {    
      await loadOrder();    
    },    
    [    
      loadOrder,    
    ],    
  );    
    
  const cancelItem = useCallback(    
    async (    
      itemIndex: number,    
    ) => {    
      if (!orderId) {    
        setError(    
          "注文IDが指定されていません。",    
        );    
    
        return;    
      }    
    
      if (    
        !Number.isInteger(itemIndex) ||    
        itemIndex < 0    
      ) {    
        setError(    
          "注文商品のインデックスが不正です。",    
        );    
    
        return;    
      }    
    
      if (    
        cancellingItemIndex !== null ||    
        returningItemIndex !== null    
      ) {    
        return;    
      }    
    
      setCancellingItemIndex(itemIndex);    
      setError("");    
    
      try {    
        const backendUrl =    
          getApiBaseUrl();    
    
        if (!backendUrl) {    
          throw new Error(    
            "VITE_API_BASE_URLが設定されていません。",    
          );    
        }    
    
        const idToken =    
          await getFirebaseIdToken();    
    
        const nextOrder =    
          await cancelOrderItem({    
            backendUrl,    
            idToken,    
            orderId,    
            itemIndex,    
          });    
    
        setOrder(nextOrder);    
        setError("");    
      } catch (caught) {    
        setError(    
          getErrorMessage(    
            caught,    
            "商品のキャンセルに失敗しました。",    
          ),    
        );    
      } finally {    
        setCancellingItemIndex(null);    
      }    
    },    
    [    
      cancellingItemIndex,    
      orderId,    
      returningItemIndex,    
    ],    
  );    
    
  const returnItem = useCallback(    
    async (    
      itemIndex: number,    
      packageState: ReturnPackageState,    
      reason: string,    
    ): Promise<boolean> => {    
      if (!orderId) {    
        setError(    
          "注文IDが指定されていません。",    
        );    
    
        return false;    
      }    
    
      if (    
        !Number.isInteger(itemIndex) ||    
        itemIndex < 0    
      ) {    
        setError(    
          "注文商品のインデックスが不正です。",    
        );    
    
        return false;    
      }    
    
      if (    
        packageState !== "unopened" &&    
        packageState !== "opened"    
      ) {    
        setError(    
          "商品の開封状態を選択してください。",    
        );    
    
        return false;    
      }    
    
      const normalizedReason =    
        reason.trim();    
    
      if (    
        packageState === "opened" &&    
        !normalizedReason    
      ) {    
        setError(    
          "返品理由を入力してください。",    
        );    
    
        return false;    
      }    
    
      const targetItem =    
        order?.items[itemIndex];    
    
      if (!targetItem) {    
        setError(    
          "返品対象の商品が見つかりません。",    
        );    
    
        return false;    
      }    
    
      if (targetItem.isCancelled) {    
        setError(    
          "キャンセル済みの商品は返品できません。",    
        );    
    
        return false;    
      }    
    
      if (!targetItem.isDispatched) {    
        setError(    
          "未発送の商品は返品できません。",    
        );    
    
        return false;    
      }    
    
      if (targetItem.isReturnRequested) {    
        setError(    
          "この商品は返品申請済みです。",    
        );    
    
        return false;    
      }    
    
      if (    
        packageState === "unopened" &&    
        (    
          targetItem.tokenTransferVerifiedAt ||    
          targetItem.transferred    
        )    
      ) {    
        setError(    
          "この商品は開封確認済みのため、開封前として返品を申請できません。",    
        );    
    
        return false;    
      }    
    
      if (    
        cancellingItemIndex !== null ||    
        returningItemIndex !== null    
      ) {    
        return false;    
      }    
    
      setReturningItemIndex(itemIndex);    
      setError("");    
    
      try {    
        const backendUrl =    
          getApiBaseUrl();    
    
        if (!backendUrl) {    
          throw new Error(    
            "VITE_API_BASE_URLが設定されていません。",    
          );    
        }    
    
        const idToken =    
          await getFirebaseIdToken();    
    
        const nextOrder =    
          await returnOrderItem({    
            backendUrl,    
            idToken,    
            orderId,    
            itemIndex,    
            packageState,    
            reason: normalizedReason,    
          });    
    
        setOrder(nextOrder);    
        setError("");    
    
        return true;    
      } catch (caught) {    
        setError(    
          getErrorMessage(    
            caught,    
            "商品の返品受付に失敗しました。",    
          ),    
        );    
    
        return false;    
      } finally {    
        setReturningItemIndex(null);    
      }    
    },    
    [    
      cancellingItemIndex,    
      order,    
      orderId,    
      returningItemIndex,    
    ],    
  );    
    
  return {    
    orderId,    
    order,    
    loading,    
    cancellingItemIndex,    
    returningItemIndex,    
    error,    
    reload,    
    cancelItem,    
    returnItem,    
  };    
}