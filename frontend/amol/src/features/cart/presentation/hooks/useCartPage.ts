// frontend/amol/src/features/cart/presentation/hooks/useCartPage.ts

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import {
  removeCartItem,
} from "../../api/cartApi";

import {
  loadCartPage,
} from "../../application/loadCartPage";

import type {
  CartDisplayItem,
} from "../../types/cart";

import type {
  CartPageState,
} from "../../types/cartPage";

import {
  calculateCartTotalAmount,
} from "../../utils/cartUtils";

const initialState: CartPageState = {
  status: "idle",
  items: [],
  error: "",
};

function getErrorMessage(
  error: unknown,
  fallbackMessage: string,
): string {
  if (
    error instanceof Error &&
    error.message.trim()
  ) {
    return error.message;
  }

  return fallbackMessage;
}

export function useCartPage() {
  const [
    state,
    setState,
  ] = useState<CartPageState>(
    initialState,
  );

  const [
    removingItemKey,
    setRemovingItemKey,
  ] = useState("");

  const mountedRef =
    useRef(false);

  const requestIdRef =
    useRef(0);

  const removingItemKeyRef =
    useRef("");

  const reload = useCallback(
    async (): Promise<void> => {
      const requestId =
        requestIdRef.current + 1;

      requestIdRef.current =
        requestId;

      if (mountedRef.current) {
        setState((currentState) => ({
          status: "loading",
          items:
            currentState.items,
          error: "",
        }));
      }

      try {
        const result =
          await loadCartPage();

        if (
          !mountedRef.current ||
          requestId !==
            requestIdRef.current
        ) {
          return;
        }

        setState({
          status: "success",
          items: result.items,
          error: "",
        });
      } catch (error) {
        if (
          !mountedRef.current ||
          requestId !==
            requestIdRef.current
        ) {
          return;
        }

        setState({
          status: "error",
          items: [],
          error:
            getErrorMessage(
              error,
              "カートの取得中にエラーが発生しました。",
            ),
        });
      }
    },
    [],
  );

  const removeItem = useCallback(
    async (
      item: CartDisplayItem,
    ): Promise<void> => {
      const itemKey =
        item.itemKey.trim();

      if (
        !itemKey ||
        removingItemKeyRef.current
      ) {
        return;
      }

      removingItemKeyRef.current =
        itemKey;

      if (mountedRef.current) {
        setRemovingItemKey(
          itemKey,
        );

        setState((currentState) => ({
          ...currentState,
          error: "",
        }));
      }

      try {
        await removeCartItem({
          item,
        });

        if (!mountedRef.current) {
          return;
        }

        setState((currentState) => ({
          status: "success",
          items:
            currentState.items.filter(
              (currentItem) =>
                currentItem.itemKey !==
                itemKey,
            ),
          error: "",
        }));
      } catch (error) {
        if (!mountedRef.current) {
          return;
        }

        setState((currentState) => ({
          ...currentState,
          error:
            getErrorMessage(
              error,
              "カート商品の削除中にエラーが発生しました。",
            ),
        }));
      } finally {
        removingItemKeyRef.current =
          "";

        if (mountedRef.current) {
          setRemovingItemKey("");
        }
      }
    },
    [],
  );

  useEffect(() => {
    mountedRef.current = true;

    void reload();

    return () => {
      mountedRef.current = false;
      requestIdRef.current += 1;
      removingItemKeyRef.current =
        "";
    };
  }, [reload]);

  const totalAmount =
    useMemo(
      () =>
        calculateCartTotalAmount(
          state.items,
        ),
      [state.items],
    );

  const loading =
    state.status === "idle" ||
    state.status === "loading";

  const isPurchaseDisabled =
    state.items.length === 0 ||
    loading ||
    removingItemKey !== "";

  return {
    items: state.items,
    totalAmount,

    loading,
    error: state.error,

    removingItemKey,
    isPurchaseDisabled,

    removeItem,
    reload,
  };
}