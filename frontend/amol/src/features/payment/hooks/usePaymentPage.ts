// frontend/amol/src/features/payment/hooks/usePaymentPage.ts

import { useCallback, useEffect, useMemo, useState } from "react";
import type { NavigateFunction } from "react-router-dom";

import { formatPrice } from "../../../components/utils/price";
import { getApiBaseUrl } from "../../../lib/apiBaseUrl";
import { getFirebaseIdToken } from "../../../lib/authToken";
import { requestJson } from "../../../lib/http";
import { loadCartPage } from "../../cart/application/loadCartPage";
import { calculateCartTotalAmount } from "../../cart/utils/cartUtils";
import { fetchShippingAddressPageInitialData } from "../../shipping-address/api/shippingAddressApi";
import type { CartDisplayItem } from "../../shared/types/cart";
import type {
  CanonicalShippingAddress,
  CreateOrderRequest,
} from "../../shared/types/payment";
import type { CardPaymentMethod } from "../../shared/types/paymentMethods";
import type { UserProfile } from "../../shared/types/shippingAddress";
import {
  createOrder,
  fetchPaymentMethods,
} from "../api/paymentApi";
import {
  getShippingAddressLabel,
  getUserFullName,
} from "../utils/format";
import {
  buildOrderItems,
  selectPrimaryPaymentMethod,
  validateOrderItems,
} from "../utils/order";

const API_BASE_URL = getApiBaseUrl();

type UsePaymentPageParams = {
  listId?: string;
  navigate: NavigateFunction;
};

type ShippingQuoteItemRequest = {
  listId: string;
  modelId: string;
  qty: number;
};

type ShippingQuoteItemResponse = {
  listId: string;
  modelId: string;
  qty: number;
  carrier: string;
  transportationId?: string;
  size: number;
  unitAmount: number;
  amount: number;
  currency: string;
};

type ShippingQuoteResponse = {
  items: ShippingQuoteItemResponse[];
  shippingAmount: number;
  currency: string;
};

type PaymentAmountSummary = {
  taxAmount: number;
  totalAmount: number;
};

function buildShippingQuoteItems(
  cartItems: CartDisplayItem[],
): ShippingQuoteItemRequest[] {
  return cartItems.map((item) => {
    if (item.type !== "list") {
      throw new Error(
        "再販商品の送料計算には現在対応していません。",
      );
    }

    if (!item.listId) {
      throw new Error(
        "送料計算に必要なリストIDを取得できませんでした。",
      );
    }

    if (!item.modelId) {
      throw new Error(
        "送料計算に必要なモデルIDを取得できませんでした。",
      );
    }

    if (item.qty <= 0) {
      throw new Error(
        "送料計算に必要な数量が不正です。",
      );
    }

    return {
      listId: item.listId,
      modelId: item.modelId,
      qty: item.qty,
    };
  });
}

async function fetchShippingQuote(
  cartItems: CartDisplayItem[],
  shippingAddressId: string,
): Promise<number> {
  if (!shippingAddressId) {
    throw new Error(
      "配送先住所IDを取得できませんでした。",
    );
  }

  if (cartItems.length === 0) {
    throw new Error(
      "送料計算対象の商品がありません。",
    );
  }

  const items = buildShippingQuoteItems(
    cartItems,
  );

  const result =
    await requestJson<ShippingQuoteResponse>(
      "/mall/me/shipping-quotes",
      {
        method: "POST",
        auth: "required",
        credentials: "include",
        json: {
          items,
          shippingAddressId,
        },
        messages: {
          requestErrorMessage:
            "送料の取得に失敗しました。",
        },
      },
    );

  if (
    !Number.isSafeInteger(
      result.shippingAmount,
    ) ||
    result.shippingAmount < 0
  ) {
    throw new Error(
      "取得した送料が不正です。",
    );
  }

  if (result.currency !== "JPY") {
    throw new Error(
      "送料の通貨が不正です。",
    );
  }

  return result.shippingAmount;
}

function calculatePaymentAmount(
  cartItems: CartDisplayItem[],
  subtotalAmount: number,
  shippingAmount: number,
): PaymentAmountSummary {
  if (
    !Number.isSafeInteger(
      subtotalAmount,
    ) ||
    subtotalAmount < 0
  ) {
    return {
      taxAmount: 0,
      totalAmount: 0,
    };
  }

  if (
    !Number.isSafeInteger(
      shippingAmount,
    ) ||
    shippingAmount < 0
  ) {
    return {
      taxAmount: 0,
      totalAmount: 0,
    };
  }

  let taxableAmount8 = 0;
  let taxableAmount10 = shippingAmount;
  let calculatedSubtotalAmount = 0;

  for (const item of cartItems) {
    const price = item.price;

    if (
      price === undefined ||
      !Number.isSafeInteger(price) ||
      price < 0
    ) {
      return {
        taxAmount: 0,
        totalAmount: 0,
      };
    }

    if (
      !Number.isSafeInteger(
        item.qty,
      ) ||
      item.qty <= 0
    ) {
      return {
        taxAmount: 0,
        totalAmount: 0,
      };
    }

    const lineAmount =
      price *
      item.qty;

    if (
      !Number.isSafeInteger(
        lineAmount,
      )
    ) {
      return {
        taxAmount: 0,
        totalAmount: 0,
      };
    }

    const nextSubtotalAmount =
      calculatedSubtotalAmount +
      lineAmount;

    if (
      !Number.isSafeInteger(
        nextSubtotalAmount,
      )
    ) {
      return {
        taxAmount: 0,
        totalAmount: 0,
      };
    }

    calculatedSubtotalAmount =
      nextSubtotalAmount;

    switch (
      item.consumptionTaxRate
    ) {
      case 8: {
        const nextTaxableAmount8 =
          taxableAmount8 +
          lineAmount;

        if (
          !Number.isSafeInteger(
            nextTaxableAmount8,
          )
        ) {
          return {
            taxAmount: 0,
            totalAmount: 0,
          };
        }

        taxableAmount8 =
          nextTaxableAmount8;

        break;
      }

      case 10: {
        const nextTaxableAmount10 =
          taxableAmount10 +
          lineAmount;

        if (
          !Number.isSafeInteger(
            nextTaxableAmount10,
          )
        ) {
          return {
            taxAmount: 0,
            totalAmount: 0,
          };
        }

        taxableAmount10 =
          nextTaxableAmount10;

        break;
      }

      default:
        return {
          taxAmount: 0,
          totalAmount: 0,
        };
    }
  }

  if (
    calculatedSubtotalAmount !==
    subtotalAmount
  ) {
    return {
      taxAmount: 0,
      totalAmount: 0,
    };
  }

  if (
    taxableAmount8 >
      Number.MAX_SAFE_INTEGER /
        8 ||
    taxableAmount10 >
      Number.MAX_SAFE_INTEGER /
        10
  ) {
    return {
      taxAmount: 0,
      totalAmount: 0,
    };
  }

  const taxAmount8 =
    Math.floor(
      taxableAmount8 *
        8 /
        100,
    );

  const taxAmount10 =
    Math.floor(
      taxableAmount10 *
        10 /
        100,
    );

  const taxAmount =
    taxAmount8 +
    taxAmount10;

  if (
    !Number.isSafeInteger(
      taxAmount,
    ) ||
    taxAmount < 0
  ) {
    return {
      taxAmount: 0,
      totalAmount: 0,
    };
  }

  const totalAmount =
    subtotalAmount +
    shippingAmount +
    taxAmount;

  if (
    !Number.isSafeInteger(
      totalAmount,
    ) ||
    totalAmount <= 0
  ) {
    return {
      taxAmount: 0,
      totalAmount: 0,
    };
  }

  return {
    taxAmount,
    totalAmount,
  };
}

export function usePaymentPage({
  listId,
  navigate,
}: UsePaymentPageParams) {
  const [paymentMethods, setPaymentMethods] = useState<CardPaymentMethod[]>([]);
  const [cartItems, setCartItems] = useState<CartDisplayItem[]>([]);
  const [userProfile, setUserProfile] = useState<UserProfile | null>(null);
  const [shippingAddresses, setShippingAddresses] = useState<CanonicalShippingAddress[]>([]);
  const [selectedPaymentMethodId, setSelectedPaymentMethodId] = useState("");
  const [shippingAmount, setShippingAmount] = useState(0);
  const [isShippingQuoteReady, setIsShippingQuoteReady] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [isPaying, setIsPaying] = useState(false);
  const [modalMessage, setModalMessage] = useState("");

  const primaryShippingAddress = useMemo(
    () => shippingAddresses[0] ?? null,
    [shippingAddresses],
  );

  const userFullName = useMemo(
    () => getUserFullName(userProfile),
    [userProfile],
  );

  const shippingAddressLabel = useMemo(() => {
    if (!primaryShippingAddress) {
      return "";
    }

    return getShippingAddressLabel(primaryShippingAddress);
  }, [primaryShippingAddress]);

  const orderId = useMemo(() => {
    if (cartItems.length === 0) {
      return "";
    }

    const firstItem = cartItems[0];

    if (!firstItem.avatarId) {
      return "";
    }

    return `${firstItem.avatarId}__${Date.now()}`;
  }, [cartItems]);

  const subtotalAmount = useMemo(
    () => calculateCartTotalAmount(cartItems),
    [cartItems],
  );

  const paymentAmount = useMemo(
    () =>
      calculatePaymentAmount(
        cartItems,
        subtotalAmount,
        shippingAmount,
      ),
    [
      cartItems,
      subtotalAmount,
      shippingAmount,
    ],
  );

  const taxAmount =
    paymentAmount.taxAmount;

  const amount =
    paymentAmount.totalAmount;

  const selectedPaymentMethod = useMemo(() => {
    if (!selectedPaymentMethodId) {
      return null;
    }

    return (
      paymentMethods.find(
        (method) => method.id === selectedPaymentMethodId,
      ) ?? null
    );
  }, [paymentMethods, selectedPaymentMethodId]);

  const backTo =
    listId === "cart"
      ? "/cart"
      : listId
        ? `/lists/${encodeURIComponent(listId)}`
        : "/lists";

  const paymentPagePath = listId
    ? `/payments/${encodeURIComponent(listId)}`
    : "/lists";

  const paymentButtonLabel = isPaying
    ? "購入処理中..."
    : `${formatPrice(amount)}で購入する`;

  const isPaymentDisabled =
    isPaying ||
    paymentMethods.length === 0 ||
    !selectedPaymentMethodId ||
    !selectedPaymentMethod ||
    !orderId ||
    !primaryShippingAddress ||
    !isShippingQuoteReady ||
    amount <= 0 ||
    cartItems.length === 0;

  const showErrorModal = useCallback((message: string) => {
    setModalMessage(message);
  }, []);

  const closeErrorModal = useCallback(() => {
    setModalMessage("");
  }, []);

  const loadPaymentPage = useCallback(async (): Promise<void> => {
    setIsLoading(true);
    setModalMessage("");
    setShippingAmount(0);
    setIsShippingQuoteReady(false);

    try {
      const idToken = await getFirebaseIdToken();

      const [
        paymentMethodResult,
        shippingAddressInitialData,
        cartPageResult,
      ] = await Promise.all([
        fetchPaymentMethods(),
        fetchShippingAddressPageInitialData({
          backendUrl: API_BASE_URL,
          idToken,
        }),
        loadCartPage(),
      ]);

      setPaymentMethods(paymentMethodResult.methods);
      setUserProfile(shippingAddressInitialData.userProfile);

      const shippingAddress =
        shippingAddressInitialData.shippingAddresses[0];

      const normalizedShippingAddress =
        shippingAddress
          ? {
              ...shippingAddress,
              street2:
                shippingAddress.street2 ??
                "",
            }
          : null;

      setShippingAddresses(
        normalizedShippingAddress
          ? [normalizedShippingAddress]
          : [],
      );

      const selectedMethod = selectPrimaryPaymentMethod(
        paymentMethodResult.methods,
        paymentMethodResult.defaultMethod,
      );

      setSelectedPaymentMethodId(
        selectedMethod?.id ??
          "",
      );

      for (
        const item of
        cartPageResult.items
      ) {
        if (
          item.consumptionTaxRate !== 8 &&
          item.consumptionTaxRate !== 10
        ) {
          throw new Error(
            "商品の消費税率を取得できませんでした。",
          );
        }
      }

      setCartItems(
        cartPageResult.items,
      );

      if (
        normalizedShippingAddress &&
        cartPageResult.items.length > 0
      ) {
        const resolvedShippingAmount =
          await fetchShippingQuote(
            cartPageResult.items,
            normalizedShippingAddress.id,
          );

        setShippingAmount(
          resolvedShippingAmount,
        );

        setIsShippingQuoteReady(true);
      }
    } catch (error) {
      const message =
        error instanceof Error
          ? error.message
          : "購入情報の取得に失敗しました。";

      showErrorModal(message);
      setPaymentMethods([]);
      setSelectedPaymentMethodId("");
      setCartItems([]);
      setUserProfile(null);
      setShippingAddresses([]);
      setShippingAmount(0);
      setIsShippingQuoteReady(false);
    } finally {
      setIsLoading(false);
    }
  }, [showErrorModal]);

  useEffect(() => {
    void loadPaymentPage();
  }, [loadPaymentPage]);

  const handleSubmitPayment = async (): Promise<void> => {
    if (isPaying) {
      return;
    }

    if (!orderId) {
      showErrorModal(
        "注文IDを生成できませんでした。",
      );
      return;
    }

    if (!selectedPaymentMethod) {
      showErrorModal(
        "支払い方法を選択してください。",
      );
      return;
    }

    if (!selectedPaymentMethod.id) {
      showErrorModal(
        "支払い方法IDを取得できませんでした。",
      );
      return;
    }

    if (!primaryShippingAddress) {
      showErrorModal(
        "配送先情報を登録してください。",
      );
      return;
    }

    if (!isShippingQuoteReady) {
      showErrorModal(
        "送料を取得できていません。",
      );
      return;
    }

    if (amount <= 0) {
      showErrorModal(
        "購入金額が不正です。",
      );
      return;
    }

    const orderItems =
      buildOrderItems(
        cartItems,
      );

    const orderItemsError =
      validateOrderItems(
        orderItems,
      );

    if (orderItemsError) {
      showErrorModal(
        orderItemsError,
      );
      return;
    }

    setIsPaying(true);
    setModalMessage("");

    try {
      const orderPayload: CreateOrderRequest = {
        id: orderId,
        shippingAddressId:
          primaryShippingAddress.id,
        paymentMethodId:
          selectedPaymentMethod.id,
        items:
          orderItems,
      };

      const order =
        await createOrder(
          orderPayload,
        );

      const resolvedOrderId =
        order.id ??
        orderId;

      const resolvedShippingAmount =
        order.shippingQuoteSnapshot?.amount;

      if (
        !Number.isSafeInteger(
          resolvedShippingAmount,
        ) ||
        resolvedShippingAmount === undefined ||
        resolvedShippingAmount < 0
      ) {
        showErrorModal(
          "注文の確定送料を取得できませんでした。",
        );
        return;
      }

      if (
        order.shippingQuoteSnapshot?.currency !==
        "JPY"
      ) {
        showErrorModal(
          "注文の送料通貨が不正です。",
        );
        return;
      }

      const resolvedPaymentAmount =
        calculatePaymentAmount(
          cartItems,
          subtotalAmount,
          resolvedShippingAmount,
        );

      const resolvedAmount =
        resolvedPaymentAmount.totalAmount;

      if (resolvedAmount <= 0) {
        showErrorModal(
          "注文金額が不正です。",
        );
        return;
      }

      setShippingAmount(
        resolvedShippingAmount,
      );

      navigate("/order-confirmed", {
        replace: true,
        state: {
          orderId:
            resolvedOrderId,
          amount:
            resolvedAmount,
          cartItems,
          shippingAddress:
            primaryShippingAddress,
        },
      });
    } catch (error) {
      const message =
        error instanceof Error
          ? error.message
          : "注文処理に失敗しました。";

      showErrorModal(message);
    } finally {
      setIsPaying(false);
    }
  };

  const handleGoToPaymentMethod = () => {
    navigate("/settings/payment-method", {
      state: {
        paymentBackTo: paymentPagePath,
      },
    });
  };

  const handleGoToShippingAddress = () => {
    navigate("/settings/shipping-address", {
      state: {
        paymentBackTo: paymentPagePath,
      },
    });
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
    shippingAmount,
    subtotalAmount,
    taxAmount,
    userFullName,
  };
}