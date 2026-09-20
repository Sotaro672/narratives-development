// frontend/mall/src/pages/DispatchPage.tsx

import { useMemo, useState } from "react";
import { Check } from "lucide-react";
import { useNavigate, useParams } from "react-router-dom";

import FooterNav from "../components/layout/FooterNav";
import Layout from "../components/layout/Layout";
import Alert from "../components/ui/Alert";
import Card from "../components/ui/Card";
import InfoList from "../components/ui/InfoList";
import SectionHeader from "../components/ui/SectionHeader";
import { useContactViewport } from "../features/contact/hooks/useContactViewport";
import {
  dispatchTrade,
  type TradeDispatchBoxSize,
  type TradeDispatchCarrier,
} from "../features/trade/infrastructure/tradeApi";

import "../styles/page-layout.css";
import "../styles/dispatch-page.css";

type DispatchRouteParams = {
  tradeId: string;
};

const BOX_SIZES: TradeDispatchBoxSize[] = [60, 80, 100, 120, 140, 160];

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

function getErrorMessage(caught: unknown, fallbackMessage: string): string {
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
    () => CARRIER_OPTIONS.find((option) => option.value === carrier) ?? null,
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

      navigate(chatPath, { replace: true });
    } catch (caught) {
      setSubmissionError(
        getErrorMessage(caught, "商品の発送処理に失敗しました。"),
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
      showFooter={false}
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
      <section className="page-section content-page-section dispatch-page">
        {!normalizedTradeId ? (
          <Alert variant="error" className="dispatch-page__alert">
            取引IDが見つかりません。
          </Alert>
        ) : null}

        <p className="content-page-description">
          配送会社と梱包する箱のサイズを選択してください。配送料は配送先にかかわらず全国一律で、箱のサイズだけで決まります。
        </p>

        <section>
          <SectionHeader title="配送会社" titleAs="h2" />

          <div className="dispatch-page__option-list" role="radiogroup" aria-label="配送会社">
            {CARRIER_OPTIONS.map((option) => {
              const selected = carrier === option.value;

              return (
                <Card
                  key={option.value}
                  interactive
                  highlighted={selected}
                  busy={submitting}
                  padding="md"
                  className="dispatch-page__option"
                  role="radio"
                  aria-checked={selected}
                  aria-disabled={submitting || undefined}
                  onClick={() => {
                    if (submitting) {
                      return;
                    }

                    setCarrier(option.value);
                    setSubmissionError("");
                  }}
                >
                  <span className="dispatch-page__option-content">
                    <strong className="dispatch-page__option-title">
                      {option.label}
                    </strong>
                    <span className="dispatch-page__option-description">
                      {option.description}
                    </span>
                  </span>

                  <span className="dispatch-page__option-aside" aria-hidden="true">
                    {selected ? <Check size={20} strokeWidth={2.5} /> : null}
                  </span>
                </Card>
              );
            })}
          </div>
        </section>

        <section>
          <SectionHeader title="箱のサイズ" titleAs="h2" />

          <p className="content-page-description">
            梱包後の箱の3辺合計に収まるサイズを選択してください。重量や配送地域による料金差はありません。
          </p>

          <div className="dispatch-page__option-list" role="radiogroup" aria-label="箱のサイズ">
            {BOX_SIZES.map((size) => {
              const selected = boxSize === size;
              const fee = SHIPPING_FEE_BY_BOX_SIZE[size];

              return (
                <Card
                  key={size}
                  interactive
                  highlighted={selected}
                  busy={submitting}
                  padding="md"
                  className="dispatch-page__option"
                  role="radio"
                  aria-checked={selected}
                  aria-disabled={submitting || undefined}
                  onClick={() => {
                    if (submitting) {
                      return;
                    }

                    setBoxSize(size);
                    setSubmissionError("");
                  }}
                >
                  <span className="dispatch-page__option-content">
                    <strong className="dispatch-page__option-title">
                      {size}サイズ
                    </strong>
                    <span className="dispatch-page__option-description">
                      3辺合計 {size}cm以内
                    </span>
                  </span>

                  <span className="dispatch-page__option-aside">
                    <strong className="dispatch-page__option-price">
                      {formatJPY(fee)}
                    </strong>
                    {selected ? (
                      <Check size={20} strokeWidth={2.5} aria-hidden="true" />
                    ) : null}
                  </span>
                </Card>
              );
            })}
          </div>
        </section>

        <Card
          as="section"
          variant="panel"
          className="dispatch-page__summary-card"
        >
          <SectionHeader title="発送内容" titleAs="h2" />

          <InfoList
            className="dispatch-page__summary-list"
            rows={[
              {
                key: "carrier",
                label: "配送会社",
                value: selectedCarrier?.label ?? "未選択",
              },
              {
                key: "box-size",
                label: "箱サイズ",
                value: boxSize !== null ? `${boxSize}サイズ` : "未選択",
              },
              {
                key: "shipping-fee",
                label: "配送料",
                value: shippingFee !== null ? formatJPY(shippingFee) : "—",
              },
            ]}
          />

          <p className="content-page-description dispatch-page__summary-description">
            日本郵便・ヤマト運輸のどちらを選択しても、同じ箱サイズであれば配送料は同額です。
          </p>
        </Card>

        {submissionError ? (
          <Alert variant="error" className="dispatch-page__alert">
            {submissionError}
          </Alert>
        ) : null}
      </section>

      {!isDesktop ? (
        <FooterNav
          variant="action"
          buttonLabel={submitting ? "発送処理中..." : "発送を確定"}
          disabled={actionButtonDisabled}
          onButtonClick={() => {
            void handleConfirm();
          }}
        />
      ) : null}
    </Layout>
  );
}