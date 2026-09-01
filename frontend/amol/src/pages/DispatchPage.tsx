// frontend/amol/src/pages/DispatchPage.tsx

import { useMemo, useState } from "react";
import { Check } from "lucide-react";
import { useNavigate, useParams } from "react-router-dom";

import Layout from "../components/layout/Layout";
import FooterNav from "../components/layout/FooterNav";
import { useContactViewport } from "../features/contact/hooks/useContactViewport";
import {
  dispatchTrade,
  type TradeDispatchBoxSize,
  type TradeDispatchCarrier,
} from "../features/trade/infrastructure/tradeApi";

import "../styles/page-layout.css";
import "../styles/settings-page.css";

type DispatchRouteParams = {
  tradeId: string;
};

const BOX_SIZES: TradeDispatchBoxSize[] = [
  60,
  80,
  100,
  120,
  140,
  160,
];

type CarrierOption = {
  value: TradeDispatchCarrier;
  label: string;
  description: string;
};

const CARRIER_OPTIONS: CarrierOption[] = [
  {
    value: "post",
    label: "日本郵便",
    description: "郵便で発送します。",
  },
  {
    value: "yamato",
    label: "ヤマト運輸",
    description: "ヤマト運輸で発送します。",
  },
];

// 画面表示用の料金表。
// 実際の決済額はbackend側でcarrier / boxSizeから再計算して確定する。
const SHIPPING_FEE_BY_BOX_SIZE: Record<TradeDispatchBoxSize, number> = {
  60: 750,
  80: 850,
  100: 1050,
  120: 1200,
  140: 1450,
  160: 1700,
};

function formatJPY(amount: number): string {
  return new Intl.NumberFormat("ja-JP", {
    style: "currency",
    currency: "JPY",
    maximumFractionDigits: 0,
  }).format(amount);
}

function getErrorMessage(
  caught: unknown,
  fallbackMessage: string,
): string {
  if (caught instanceof Error && caught.message) {
    return caught.message;
  }

  return fallbackMessage;
}

export default function DispatchPage() {
  const navigate = useNavigate();
  const { tradeId } = useParams<DispatchRouteParams>();
  const { isDesktop } = useContactViewport();

  const [carrier, setCarrier] = useState<TradeDispatchCarrier | null>(null);
  const [boxSize, setBoxSize] = useState<TradeDispatchBoxSize | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [submissionError, setSubmissionError] = useState("");

  const normalizedTradeId = tradeId?.trim() ?? "";

  const shippingFee = useMemo(() => {
    if (boxSize === null) {
      return null;
    }

    return SHIPPING_FEE_BY_BOX_SIZE[boxSize];
  }, [boxSize]);

  const selectedCarrier = useMemo(
    () =>
      CARRIER_OPTIONS.find(
        (option) => option.value === carrier,
      ) ?? null,
    [carrier],
  );

  const chatPath = normalizedTradeId
    ? `/chats/trades/${encodeURIComponent(normalizedTradeId)}`
    : "/chats";

  const actionButtonDisabled =
    submitting ||
    !normalizedTradeId ||
    carrier === null ||
    boxSize === null ||
    shippingFee === null;

  const handleBack = () => {
    if (submitting) {
      return;
    }

    navigate(chatPath);
  };

  const handleConfirm = async (): Promise<void> => {
    if (
      submitting ||
      !normalizedTradeId ||
      carrier === null ||
      boxSize === null
    ) {
      return;
    }

    setSubmitting(true);
    setSubmissionError("");

    try {
      await dispatchTrade({
        tradeId: normalizedTradeId,
        carrier,
        boxSize,
      });

      navigate(chatPath, {
        replace: true,
      });
    } catch (caught) {
      setSubmissionError(
        getErrorMessage(
          caught,
          "商品の発送処理に失敗しました。",
        ),
      );
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Layout
      title="発送"
      titleClickable={false}
      showBackButton
      onBackButtonClick={handleBack}
      mode="mypage"
      actionButtonLabel={
        isDesktop
          ? submitting
            ? "発送処理中..."
            : "発送を確定"
          : undefined
      }
      onActionButtonClick={
        isDesktop
          ? () => {
              void handleConfirm();
            }
          : undefined
      }
      actionButtonDisabled={actionButtonDisabled}
    >
      <section className="page-section content-page-section settings-page">
        {!normalizedTradeId ? (
          <div role="alert">
            取引IDが見つかりません。
          </div>
        ) : null}

        <p className="content-page-description">
          配送会社と梱包する箱のサイズを選択してください。配送料は配送先にかかわらず全国一律で、箱のサイズだけで決まります。
        </p>

        <section>
          <h2>配送会社</h2>

          <div className="settings-list" role="list">
            {CARRIER_OPTIONS.map((option) => {
              const selected = carrier === option.value;

              return (
                <button
                  key={option.value}
                  type="button"
                  className="settings-item"
                  onClick={() => {
                    if (submitting) {
                      return;
                    }

                    setCarrier(option.value);
                    setSubmissionError("");
                  }}
                  disabled={submitting}
                  aria-pressed={selected}
                  role="listitem"
                >
                  <span>
                    <strong>{option.label}</strong>
                    <br />
                    <span>{option.description}</span>
                  </span>

                  <span aria-hidden="true">
                    {selected ? (
                      <Check
                        size={20}
                        strokeWidth={2.5}
                      />
                    ) : null}
                  </span>
                </button>
              );
            })}
          </div>
        </section>

        <section>
          <h2>箱のサイズ</h2>

          <p className="content-page-description">
            梱包後の箱の3辺合計に収まるサイズを選択してください。重量や配送地域による料金差はありません。
          </p>

          <div className="settings-list" role="list">
            {BOX_SIZES.map((size) => {
              const selected = boxSize === size;
              const fee = SHIPPING_FEE_BY_BOX_SIZE[size];

              return (
                <button
                  key={size}
                  type="button"
                  className="settings-item"
                  onClick={() => {
                    if (submitting) {
                      return;
                    }

                    setBoxSize(size);
                    setSubmissionError("");
                  }}
                  disabled={submitting}
                  aria-pressed={selected}
                  role="listitem"
                >
                  <span>
                    <strong>{size}サイズ</strong>
                    <br />
                    <span>
                      3辺合計 {size}cm以内
                    </span>
                  </span>

                  <span>
                    <strong>{formatJPY(fee)}</strong>

                    {selected ? (
                      <>
                        {" "}
                        <Check
                          size={20}
                          strokeWidth={2.5}
                          aria-hidden="true"
                        />
                      </>
                    ) : null}
                  </span>
                </button>
              );
            })}
          </div>
        </section>

        <section>
          <h2>発送内容</h2>

          <dl>
            <div>
              <dt>配送会社</dt>
              <dd>
                {selectedCarrier?.label ?? "未選択"}
              </dd>
            </div>

            <div>
              <dt>箱サイズ</dt>
              <dd>
                {boxSize !== null
                  ? `${boxSize}サイズ`
                  : "未選択"}
              </dd>
            </div>

            <div>
              <dt>配送料</dt>
              <dd>
                <strong>
                  {shippingFee !== null
                    ? formatJPY(shippingFee)
                    : "—"}
                </strong>
              </dd>
            </div>
          </dl>

          <p className="content-page-description">
            日本郵便・ヤマト運輸のどちらを選択しても、同じ箱サイズであれば配送料は同額です。
          </p>
        </section>

        {submissionError ? (
          <div role="alert">
            {submissionError}
          </div>
        ) : null}
      </section>

      {!isDesktop ? (
        <FooterNav
          variant="action"
          buttonLabel={
            submitting
              ? "発送処理中..."
              : "発送を確定"
          }
          disabled={actionButtonDisabled}
          onButtonClick={() => {
            void handleConfirm();
          }}
        />
      ) : null}
    </Layout>
  );
}