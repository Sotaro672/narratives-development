//frontend\amol\src\features\payment\hooks\usePaymentPage.ts
import { useCallback, useEffect, useMemo, useState } from "react";
import type { NavigateFunction } from "react-router-dom";

import {
  fetchCartItemsWithCatalog,
  fetchCurrentAvatarId,
  getFirebaseIdToken,
} from "../../cart/api/cartApi";
import { calculateCartTotalAmount, formatYen } from "../../cart/utils/cartUtils";
import { fetchShippingAddressPageInitialData } from "../../shipping-address/api/shippingAddressApi";
import type { UserProfile } from "../../shipping-address/types";
import { createOrder, createPayment, fetchPaymentContext, fetchPaymentMethods } from "../api/paymentApi";
import { API_BASE_URL } from "../api/paymentHttp";
import type {
  CanonicalCartDisplayItem,
  CanonicalShippingAddress,
  CreateOrderRequest,
  PaymentContext,
  PaymentMethod,
} from "../types";
import { getShippingAddressLabel, getUserFullName } from "../utils/format";
import {
  isPaymentRequiresAction,
  isPaymentSucceeded,
  normalizeCartItems,
  normalizeShippingAddress,
} from "../utils/guards";
import {
  buildOrderItems,
  buildPaymentMethodSnapshot,
  buildShippingSnapshot,
  selectPrimaryPaymentMethod,
  validateOrderItems,
} from "../utils/order";

type UsePaymentPageParams = {
  listId?: string;
  navigate: NavigateFunction;
};

export function usePaymentPage({ listId, navigate }: UsePaymentPageParams) {
  const [paymentContext, setPaymentContext] = useState<PaymentContext | null>(
    null,
  );
  const [paymentMethods, setPaymentMethods] = useState<PaymentMethod[]>([]);
  const [cartItems, setCartItems] = useState<CanonicalCartDisplayItem[]>([]);
  const [userProfile, setUserProfile] = useState<UserProfile | null>(null);
  const [shippingAddresses, setShippingAddresses] = useState<
    CanonicalShippingAddress[]
  >([]);
  const [selectedPaymentMethodId, setSelectedPaymentMethodId] = useState("");
  const [isLoading, setIsLoading] = useState(true);
  const [isPaying, setIsPaying] = useState(false);
  const [modalMessage, setModalMessage] = useState("");

  const primaryShippingAddress = useMemo(() => {
    return shippingAddresses[0] ?? null;
  }, [shippingAddresses]);

  const userFullName = useMemo(() => {
    return getUserFullName(userProfile);
  }, [userProfile]);

  const shippingAddressLabel = useMemo(() => {
    if (!primaryShippingAddress) {
      return "";
    }

    return getShippingAddressLabel(primaryShippingAddress);
  }, [primaryShippingAddress]);

  const orderId = useMemo(() => {
    if (!cartItems.length) {
      return "";
    }

    const firstItem = cartItems[0];

    return `${firstItem.avatarId}__${Date.now()}`;
  }, [cartItems]);

  const amount = useMemo(() => {
    return calculateCartTotalAmount(cartItems);
  }, [cartItems]);

  const selectedPaymentMethod = useMemo(() => {
    if (!selectedPaymentMethodId) {
      return null;
    }

    return (
      paymentMethods.find((method) => method.id === selectedPaymentMethodId) ??
      null
    );
  }, [paymentMethods, selectedPaymentMethodId]);

  const backTo =
    listId === "cart" ? "/cart" : listId ? `/lists/${listId}` : "/lists";

  const paymentButtonLabel = isPaying
    ? "決済中..."
    : `${formatYen(amount)}を支払う`;

  const isPaymentDisabled =
    isPaying ||
    paymentMethods.length === 0 ||
    !selectedPaymentMethodId ||
    !selectedPaymentMethod ||
    !orderId ||
    !primaryShippingAddress ||
    amount <= 0 ||
    cartItems.length === 0;

  const showErrorModal = useCallback((message: string) => {
    setModalMessage(message);
  }, []);

  const closeErrorModal = useCallback(() => {
    setModalMessage("");
  }, []);

  const loadPaymentPage = useCallback(async () => {
    setIsLoading(true);
    setModalMessage("");

    try {
      const idToken = await getFirebaseIdToken();

      const [context, paymentMethodResult, shippingAddressInitialData] =
        await Promise.all([
          fetchPaymentContext(),
          fetchPaymentMethods(),
          fetchShippingAddressPageInitialData({
            backendUrl: API_BASE_URL,
            idToken,
          }),
        ]);

      setPaymentContext(context);
      setPaymentMethods(paymentMethodResult.methods);
      setUserProfile(shippingAddressInitialData.userProfile);

      const nextShippingAddress = normalizeShippingAddress(
        shippingAddressInitialData.shippingAddresses[0] ?? null,
      );

      setShippingAddresses(nextShippingAddress ? [nextShippingAddress] : []);

      const selectedMethod = selectPrimaryPaymentMethod(
        paymentMethodResult.methods,
        paymentMethodResult.defaultMethod,
      );

      setSelectedPaymentMethodId(selectedMethod?.id ?? "");

      const avatarId =
        typeof context.avatarId === "string" && context.avatarId.trim() !== ""
          ? context.avatarId.trim()
          : await fetchCurrentAvatarId(API_BASE_URL);

      const items = await fetchCartItemsWithCatalog({
        apiBaseUrl: API_BASE_URL,
        avatarId,
      });

      setCartItems(normalizeCartItems(items));
    } catch (error) {
      const message =
        error instanceof Error
          ? error.message
          : "決済情報の取得に失敗しました。";

      showErrorModal(message);
      setPaymentContext(null);
      setPaymentMethods([]);
      setSelectedPaymentMethodId("");
      setCartItems([]);
      setUserProfile(null);
      setShippingAddresses([]);
    } finally {
      setIsLoading(false);
    }
  }, [showErrorModal]);

  useEffect(() => {
    void loadPaymentPage();
  }, [loadPaymentPage]);

  const handleSubmitPayment = async () => {
    if (isPaying) {
      return;
    }

    if (!orderId) {
      showErrorModal("注文IDを生成できませんでした。");
      return;
    }

    if (!selectedPaymentMethod) {
      showErrorModal("支払い方法を選択してください。");
      return;
    }

    if (!selectedPaymentMethod.id) {
      showErrorModal("支払い方法IDを取得できませんでした。");
      return;
    }

    if (!selectedPaymentMethod.stripeCustomerId) {
      showErrorModal("Stripe customer ID を取得できませんでした。");
      return;
    }

    if (!selectedPaymentMethod.stripePaymentMethodId) {
      showErrorModal("Stripe payment method ID を取得できませんでした。");
      return;
    }

    if (!primaryShippingAddress) {
      showErrorModal("配送先情報を登録してください。");
      return;
    }

    if (amount <= 0) {
      showErrorModal("決済金額が不正です。");
      return;
    }

    const avatarId = cartItems[0]?.avatarId || paymentContext?.avatarId || "";

    if (!avatarId) {
      showErrorModal("avatarIdを取得できませんでした。");
      return;
    }

    const cartId = avatarId;
    const orderItems = buildOrderItems(cartItems);
    const orderItemsError = validateOrderItems(orderItems);

    if (orderItemsError) {
      showErrorModal(orderItemsError);
      return;
    }

    setIsPaying(true);
    setModalMessage("");

    try {
      const orderPayload: CreateOrderRequest = {
        id: orderId,
        avatarId,
        cartId,
        shippingSnapshot: buildShippingSnapshot(primaryShippingAddress),
        paymentMethodSnapshot: buildPaymentMethodSnapshot(selectedPaymentMethod),
        items: orderItems,
      };

      const order = await createOrder(orderPayload);
      const resolvedOrderId = order.id ?? orderId;

      const payment = await createPayment({
        paymentId: resolvedOrderId,
        paymentMethodId: selectedPaymentMethod.id,
        stripeCustomerId: selectedPaymentMethod.stripeCustomerId,
        stripePaymentMethodId: selectedPaymentMethod.stripePaymentMethodId,
        amount,
      });

      if (isPaymentRequiresAction(payment)) {
        showErrorModal(
          "追加認証が必要な決済です。現在の画面では3Dセキュア認証に未対応です。",
        );
        return;
      }

      if (!isPaymentSucceeded(payment)) {
        showErrorModal(
          `決済が完了しませんでした。status=${payment.status ?? "UNKNOWN"}`,
        );
        return;
      }

      navigate("/order-confirmed", {
        replace: true,
        state: {
          payment,
          order,
          paymentId: payment.paymentId ?? resolvedOrderId,
          orderId: resolvedOrderId,
          paymentMethodId: payment.paymentMethodId,
          stripePaymentIntentId: payment.stripePaymentIntentId,
          amount: payment.amount,
          cartItems,
          shippingAddress: primaryShippingAddress,
        },
      });
    } catch (error) {
      const message =
        error instanceof Error
          ? error.message
          : "注文または決済処理に失敗しました。";

      showErrorModal(message);
    } finally {
      setIsPaying(false);
    }
  };

  const handleGoToPaymentMethod = () => {
    navigate("/settings/payment-method");
  };

  const handleGoToShippingAddress = () => {
    navigate("/settings/shipping-address");
  };

  return {
    amount,
    backTo,
    cartItems,
    closeErrorModal,
    handleGoToPaymentMethod,
    handleGoToShippingAddress,
    handleSubmitPayment,
    isLoading,
    isPaymentDisabled,
    modalMessage,
    paymentButtonLabel,
    paymentMethods,
    primaryShippingAddress,
    selectedPaymentMethodId,
    setSelectedPaymentMethodId,
    shippingAddressLabel,
    userFullName,
  };
}