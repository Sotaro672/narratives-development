// frontend/amol/src/pages/PaymentPage.tsx
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import Layout from "../components/layout/Layout";
import {
  fetchCartItemsWithCatalog,
  fetchCurrentAvatarId,
  getFirebaseIdToken,
} from "../features/cart/api/cartApi";
import type { CartDisplayItem } from "../features/cart/types";
import {
  calculateCartTotalAmount,
  formatYen,
  getModelPrice,
  getModelVariation,
} from "../features/cart/utils/cartUtils";
import { fetchShippingAddressPageInitialData } from "../features/shipping-address/api/shippingAddressApi";
import type {
  ShippingAddress,
  UserProfile,
} from "../features/shipping-address/types";
import "../styles/payment-page.css";

type PaymentMethod = {
  id: string;
  userId: string;
  stripeCustomerId: string;
  stripePaymentMethodId: string;
  brand: string;
  last4: string;
  expMonth: number;
  expYear: number;
  cardholderName: string;
  isDefault: boolean;
  createdAt?: string;
  updatedAt?: string;
};

type PaymentContext = {
  avatarId?: string;
  avatarUid?: string;
};

type PaymentMethodListResponse = {
  data?: PaymentMethod[];
  error?: string;
};

type PaymentMethodDefaultResponse = {
  data?: PaymentMethod | null;
  error?: string;
};

type CreatedPayment = {
  id?: string;
  paymentId?: string;
  paymentMethodId?: string;
  stripeCustomerId?: string;
  stripePaymentMethodId?: string;
  stripePaymentIntentId?: string;
  amount?: number;
  status?: string;
  clientSecret?: string;
  requiresAction?: boolean;
  createdAt?: string;
};

type CreatedOrder = {
  id?: string;
  userId?: string;
  avatarId?: string;
  cartId?: string;
  paid?: boolean;
  createdAt?: string;
};

type CanonicalCartDisplayItem = CartDisplayItem & {
  avatarId: string;
  inventoryId: string;
  listId: string;
  modelId: string;
  qty: number;
};

type CanonicalShippingAddress = ShippingAddress & {
  zipCode: string;
  state: string;
  city: string;
  street: string;
  street2: string;
};

type OrderShippingSnapshot = {
  zipCode: string;
  state: string;
  city: string;
  street: string;
  street2: string;
  country: "JP";
};

type OrderPaymentMethodSnapshot = {
  customerId: string;
  brand: string;
  last4: string;
  expMonth: number;
  expYear: number;
  cardholderName: string;
  isDefault: boolean;
};

type OrderItemSnapshot = {
  inventoryId: string;
  isCanceled: false;
  isDispatched: false;
  listId: string;
  modelId: string;
  price: number;
  qty: number;
  transferred: false;
};

type CreateOrderRequest = {
  id: string;
  avatarId: string;
  cartId: string;
  shippingSnapshot: OrderShippingSnapshot;
  paymentMethodSnapshot: OrderPaymentMethodSnapshot;
  items: OrderItemSnapshot[];
};

type CreatePaymentRequest = {
  paymentId: string;
  paymentMethodId: string;
  stripeCustomerId: string;
  stripePaymentMethodId: string;
  amount: number;
};

const MOBILE_PORTRAIT_MEDIA_QUERY =
  "(max-width: 959px) and (orientation: portrait)";

const API_BASE_URL = getApiBaseUrl();

function getApiBaseUrl(): string {
  const env = import.meta.env.VITE_API_BASE_URL;

  if (typeof env === "string" && env.trim() !== "") {
    return env.replace(/\/$/, "");
  }

  return "";
}

function getResponseErrorMessage(body: unknown, fallback: string): string {
  if (!body || typeof body !== "object") {
    return fallback;
  }

  const errorBody = body as {
    error?: string;
    detail?: string;
    message?: string;
    errorMessage?: string;
  };

  return (
    errorBody.errorMessage ??
    errorBody.detail ??
    errorBody.message ??
    errorBody.error ??
    fallback
  );
}

function selectPrimaryPaymentMethod(
  methods: PaymentMethod[],
  defaultMethod: PaymentMethod | null,
): PaymentMethod | null {
  if (defaultMethod) {
    return defaultMethod;
  }

  return methods.find((method) => method.isDefault) ?? methods[0] ?? null;
}

function formatCardBrand(brand: string): string {
  if (!brand) {
    return "カード";
  }

  switch (brand.toLowerCase()) {
    case "visa":
      return "Visa";
    case "mastercard":
      return "Mastercard";
    case "amex":
      return "American Express";
    case "jcb":
      return "JCB";
    case "diners":
      return "Diners Club";
    case "discover":
      return "Discover";
    default:
      return brand;
  }
}

function formatCardholderName(method: PaymentMethod): string {
  return method.cardholderName.trim() || "-";
}

function formatCardLast4(method: PaymentMethod): string {
  return method.last4 ? `•••• ${method.last4}` : "-";
}

function formatCardExpiry(method: PaymentMethod): string {
  if (!method.expMonth || !method.expYear) {
    return "-";
  }

  return `${String(method.expMonth).padStart(2, "0")}/${method.expYear}`;
}

function getUserFullName(userProfile: UserProfile | null): string {
  if (!userProfile) {
    return "";
  }

  const lastName =
    "last_name" in userProfile && typeof userProfile.last_name === "string"
      ? userProfile.last_name
      : "";

  const firstName =
    "first_name" in userProfile && typeof userProfile.first_name === "string"
      ? userProfile.first_name
      : "";

  return `${lastName} ${firstName}`.trim();
}

function getShippingAddressLabel(address: CanonicalShippingAddress): string {
  const zipLine = address.zipCode ? `〒${address.zipCode}` : "";
  const addressLine =
    `${address.state}${address.city}${address.street}${address.street2}`.trim();

  return [zipLine, addressLine].filter(Boolean).join("\n");
}

function buildShippingSnapshot(
  address: CanonicalShippingAddress,
): OrderShippingSnapshot {
  return {
    zipCode: address.zipCode,
    state: address.state,
    city: address.city,
    street: address.street,
    street2: address.street2,
    country: "JP",
  };
}

function buildPaymentMethodSnapshot(
  method: PaymentMethod,
): OrderPaymentMethodSnapshot {
  return {
    customerId: method.stripeCustomerId,
    brand: method.brand,
    last4: method.last4,
    expMonth: method.expMonth,
    expYear: method.expYear,
    cardholderName: method.cardholderName,
    isDefault: method.isDefault,
  };
}

function buildOrderItems(
  cartItems: CanonicalCartDisplayItem[],
): OrderItemSnapshot[] {
  return cartItems.map((item) => {
    const price = getModelPrice(item.catalog, item.modelId);

    return {
      inventoryId: item.inventoryId,
      isCanceled: false,
      isDispatched: false,
      listId: item.listId,
      modelId: item.modelId,
      price: price ?? 0,
      qty: item.qty,
      transferred: false,
    };
  });
}

function validateOrderItems(items: OrderItemSnapshot[]): string | null {
  if (items.length === 0) {
    return "注文対象の商品がありません。";
  }

  for (const item of items) {
    if (!item.inventoryId) {
      return "注文商品の inventoryId を取得できませんでした。";
    }

    if (!item.listId) {
      return "注文商品の listId を取得できませんでした。";
    }

    if (!item.modelId) {
      return "注文商品の modelId を取得できませんでした。";
    }

    if (!item.qty || item.qty <= 0) {
      return "注文商品の数量が不正です。";
    }

    if (item.price < 0) {
      return "注文商品の価格が不正です。";
    }
  }

  return null;
}

function isPaymentSucceeded(payment: CreatedPayment): boolean {
  const normalizedStatus = payment.status?.trim().toLowerCase();

  return normalizedStatus === "succeeded";
}

function isPaymentRequiresAction(payment: CreatedPayment): boolean {
  const normalizedStatus = payment.status?.trim().toLowerCase();

  return (
    payment.requiresAction === true ||
    normalizedStatus === "requires_action" ||
    normalizedStatus === "requires_source_action"
  );
}

function normalizeCartItems(items: CartDisplayItem[]): CanonicalCartDisplayItem[] {
  return items.map((item) => item as CanonicalCartDisplayItem);
}

function normalizeShippingAddress(
  address: ShippingAddress | null,
): CanonicalShippingAddress | null {
  if (!address) {
    return null;
  }

  return address as CanonicalShippingAddress;
}

async function getAuthHeaders(): Promise<HeadersInit> {
  const idToken = await getFirebaseIdToken();

  return {
    Accept: "application/json",
    "Content-Type": "application/json",
    Authorization: `Bearer ${idToken}`,
  };
}

async function parseJsonOrThrow<T>(response: Response): Promise<T> {
  const text = await response.text();

  let body: unknown = null;

  if (text) {
    try {
      body = JSON.parse(text);
    } catch {
      throw new Error("APIがJSON以外を返しました。");
    }
  }

  if (!response.ok) {
    throw new Error(
      getResponseErrorMessage(
        body,
        `APIエラーが発生しました。status=${response.status}`,
      ),
    );
  }

  return body as T;
}

async function parseJsonOrNull<T>(response: Response): Promise<T | null> {
  const contentType = response.headers.get("content-type") || "";

  if (!contentType.includes("application/json")) {
    return null;
  }

  return (await response.json()) as T;
}

async function fetchPaymentContext(): Promise<PaymentContext> {
  const headers = await getAuthHeaders();

  const response = await fetch(`${API_BASE_URL}/mall/me/payment`, {
    method: "GET",
    headers,
    credentials: "include",
  });

  return parseJsonOrThrow<PaymentContext>(response);
}

async function fetchPaymentMethods(): Promise<{
  methods: PaymentMethod[];
  defaultMethod: PaymentMethod | null;
}> {
  const headers = await getAuthHeaders();

  const [listResponse, defaultResponse] = await Promise.all([
    fetch(`${API_BASE_URL}/mall/me/payment-methods`, {
      method: "GET",
      headers,
      credentials: "include",
    }),
    fetch(`${API_BASE_URL}/mall/me/payment-methods/default`, {
      method: "GET",
      headers,
      credentials: "include",
    }),
  ]);

  const listBody =
    await parseJsonOrNull<PaymentMethodListResponse>(listResponse);

  const defaultBody =
    await parseJsonOrNull<PaymentMethodDefaultResponse>(defaultResponse);

  if (!listResponse.ok) {
    throw new Error(
      getResponseErrorMessage(listBody, "支払い方法の取得に失敗しました。"),
    );
  }

  if (!defaultResponse.ok && defaultResponse.status !== 404) {
    throw new Error(
      getResponseErrorMessage(
        defaultBody,
        "既定の支払い方法の取得に失敗しました。",
      ),
    );
  }

  return {
    methods: Array.isArray(listBody?.data) ? listBody.data : [],
    defaultMethod: defaultBody?.data ?? null,
  };
}

async function createOrder(input: CreateOrderRequest): Promise<CreatedOrder> {
  const headers = await getAuthHeaders();

  const response = await fetch(`${API_BASE_URL}/mall/me/orders`, {
    method: "POST",
    headers,
    credentials: "include",
    body: JSON.stringify(input),
  });

  const order = await parseJsonOrThrow<CreatedOrder>(response);

  return {
    ...order,
    id: order.id ?? input.id,
    avatarId: order.avatarId ?? input.avatarId,
    cartId: order.cartId ?? input.cartId,
    paid: order.paid ?? false,
  };
}

async function createPayment(
  input: CreatePaymentRequest,
): Promise<CreatedPayment> {
  const headers = await getAuthHeaders();

  const response = await fetch(`${API_BASE_URL}/mall/me/payments`, {
    method: "POST",
    headers,
    credentials: "include",
    body: JSON.stringify({
      paymentId: input.paymentId,
      paymentMethodId: input.paymentMethodId,
      stripeCustomerId: input.stripeCustomerId,
      stripePaymentMethodId: input.stripePaymentMethodId,
      amount: input.amount,
    }),
  });

  const data = await parseJsonOrThrow<CreatedPayment>(response);

  return {
    ...data,
    paymentId: data.paymentId ?? data.id ?? input.paymentId,
    paymentMethodId: data.paymentMethodId ?? input.paymentMethodId,
    stripeCustomerId: data.stripeCustomerId ?? input.stripeCustomerId,
    stripePaymentMethodId:
      data.stripePaymentMethodId ?? input.stripePaymentMethodId,
    amount: data.amount ?? input.amount,
  };
}

export default function PaymentPage() {
  const navigate = useNavigate();
  const { listId } = useParams<{ listId: string }>();

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
  const [isMobilePortrait, setIsMobilePortrait] = useState(false);

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

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    const mobilePortraitQuery = window.matchMedia(MOBILE_PORTRAIT_MEDIA_QUERY);

    const updateMobilePortraitState = () => {
      setIsMobilePortrait(mobilePortraitQuery.matches);
    };

    updateMobilePortraitState();

    if (typeof mobilePortraitQuery.addEventListener === "function") {
      mobilePortraitQuery.addEventListener("change", updateMobilePortraitState);

      return () => {
        mobilePortraitQuery.removeEventListener(
          "change",
          updateMobilePortraitState,
        );
      };
    }

    mobilePortraitQuery.addListener(updateMobilePortraitState);

    return () => {
      mobilePortraitQuery.removeListener(updateMobilePortraitState);
    };
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

  return (
    <>
      {isLoading ? (
        <Layout
          title="お支払い"
          mode="mypage"
          showBackButton
          backTo={backTo}
          showFooter={false}
          hideHamburgerMenu
          hideSettingsButton
          mainClassName="payment-page"
        >
          <section className="payment-page__section">
            <p>決済情報を読み込んでいます。</p>
          </section>
        </Layout>
      ) : (
        <Layout
          title="お支払い"
          mode="mypage"
          showBackButton
          backTo={backTo}
          showFooter={isMobilePortrait}
          hideHamburgerMenu
          hideSettingsButton
          mainClassName="payment-page"
          actionButtonLabel={!isMobilePortrait ? paymentButtonLabel : undefined}
          onActionButtonClick={
            !isMobilePortrait ? handleSubmitPayment : undefined
          }
          actionButtonDisabled={isPaymentDisabled}
          footerProps={
            isMobilePortrait
              ? {
                  variant: "action",
                  buttonLabel: paymentButtonLabel,
                  disabled: isPaymentDisabled,
                  onButtonClick: handleSubmitPayment,
                }
              : undefined
          }
        >
          <section className="payment-page__section">
            <div className="payment-page__content">
              <div className="payment-page__left-column">
                <section className="payment-page__card">
                  <h2 className="payment-page__section-title">注文内容</h2>

                  {cartItems.length > 0 ? (
                    <ul className="payment-page__items">
                      {cartItems.map((item) => {
                        const catalog = item.catalog;
                        const model = getModelVariation(catalog, item.modelId);
                        const price = getModelPrice(catalog, item.modelId);
                        const lineAmount =
                          price === null ? null : price * item.qty;
                        const title =
                          catalog?.productBlueprint.productName ||
                          catalog?.list.title ||
                          "商品名未設定";

                        return (
                          <li className="payment-page__item" key={item.itemKey}>
                            <div>
                              <p className="payment-page__item-title">
                                {title}
                              </p>

                              <p className="payment-page__item-meta">
                                {model?.colorName
                                  ? `カラー: ${model.colorName}`
                                  : ""}
                                {model?.colorName && model?.size ? " / " : ""}
                                {model?.size ? `サイズ: ${model.size}` : ""}
                              </p>

                              <p className="payment-page__item-meta">
                                数量: {item.qty}
                              </p>
                            </div>

                            <p className="payment-page__item-price">
                              {lineAmount === null
                                ? "価格未設定"
                                : formatYen(lineAmount)}
                            </p>
                          </li>
                        );
                      })}
                    </ul>
                  ) : (
                    <p className="payment-page__empty">
                      カート情報がありません。
                    </p>
                  )}

                  <div className="payment-page__total">
                    <span>合計</span>
                    <strong>{formatYen(amount)}</strong>
                  </div>
                </section>
              </div>

              <div className="payment-page__right-column">
                <section className="payment-page__card">
                  <div className="payment-page__section-header">
                    <h2 className="payment-page__section-title">配送先情報</h2>
                    <button
                      type="button"
                      className="payment-page__text-button"
                      onClick={handleGoToShippingAddress}
                    >
                      配送先を管理
                    </button>
                  </div>

                  {primaryShippingAddress ? (
                    <div className="payment-page__shipping-address">
                      {userFullName ? (
                        <p className="payment-page__shipping-address-name">
                          {userFullName}
                        </p>
                      ) : null}

                      {shippingAddressLabel.split("\n").map((line) => (
                        <p
                          className="payment-page__shipping-address-line"
                          key={line}
                        >
                          {line}
                        </p>
                      ))}
                    </div>
                  ) : (
                    <div className="payment-page__empty-block">
                      <p>配送先情報が登録されていません。</p>
                      <button
                        type="button"
                        className="payment-page__primary-button"
                        onClick={handleGoToShippingAddress}
                      >
                        配送先情報を登録する
                      </button>
                    </div>
                  )}
                </section>

                <section className="payment-page__card">
                  <div className="payment-page__section-header">
                    <h2 className="payment-page__section-title">支払い方法</h2>
                    <button
                      type="button"
                      className="payment-page__text-button"
                      onClick={handleGoToPaymentMethod}
                    >
                      カードを管理
                    </button>
                  </div>

                  {paymentMethods.length > 0 ? (
                    <div className="payment-page__payment-methods">
                      {paymentMethods.map((method) => {
                        return (
                          <label
                            className="payment-page__payment-method"
                            key={method.id}
                          >
                            <input
                              type="radio"
                              name="paymentMethod"
                              value={method.id}
                              checked={selectedPaymentMethodId === method.id}
                              onChange={() =>
                                setSelectedPaymentMethodId(method.id)
                              }
                            />

                            <span className="payment-page__payment-method-body">
                              <span className="payment-page__payment-method-row">
                                <span className="payment-page__payment-method-label">
                                  ブランド
                                </span>
                                <span className="payment-page__payment-method-value">
                                  {formatCardBrand(method.brand)}
                                </span>
                              </span>

                              <span className="payment-page__payment-method-row">
                                <span className="payment-page__payment-method-label">
                                  口座名義
                                </span>
                                <span className="payment-page__payment-method-value">
                                  {formatCardholderName(method)}
                                </span>
                              </span>

                              <span className="payment-page__payment-method-row">
                                <span className="payment-page__payment-method-label">
                                  番号下4桁
                                </span>
                                <span className="payment-page__payment-method-value">
                                  {formatCardLast4(method)}
                                </span>
                              </span>

                              <span className="payment-page__payment-method-row">
                                <span className="payment-page__payment-method-label">
                                  有効期限
                                </span>
                                <span className="payment-page__payment-method-value">
                                  {formatCardExpiry(method)}
                                </span>
                              </span>
                            </span>
                          </label>
                        );
                      })}
                    </div>
                  ) : (
                    <div className="payment-page__empty-block">
                      <p>登録済みの支払い方法がありません。</p>
                      <button
                        type="button"
                        className="payment-page__primary-button"
                        onClick={handleGoToPaymentMethod}
                      >
                        支払い方法を登録する
                      </button>
                    </div>
                  )}
                </section>
              </div>
            </div>
          </section>
        </Layout>
      )}

      {modalMessage ? (
        <div
          className="payment-page__modal-backdrop"
          role="presentation"
          onClick={closeErrorModal}
        >
          <div
            className="payment-page__modal"
            role="alertdialog"
            aria-modal="true"
            aria-labelledby="payment-error-modal-title"
            onClick={(event) => event.stopPropagation()}
          >
            <h2
              id="payment-error-modal-title"
              className="payment-page__modal-title"
            >
              注文または決済処理に失敗しました
            </h2>

            <p className="payment-page__modal-message">{modalMessage}</p>

            <button
              type="button"
              className="payment-page__primary-button"
              onClick={closeErrorModal}
            >
              閉じる
            </button>
          </div>
        </div>
      ) : null}
    </>
  );
}