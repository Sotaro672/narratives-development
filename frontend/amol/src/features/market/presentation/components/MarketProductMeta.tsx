// frontend/amol/src/features/market/presentation/components/MarketProductMeta.tsx

type MarketProductMetaProps = {
  condition?: string;

  hasModelInfo: boolean;

  modelKind: string;
  modelKindLabel: string;
  modelNumber: string;
  modelSize: string;

  hasColorInfo: boolean;
  modelColorName: string;
  modelColorCssValue: string;

  measurementsLabel: string;
  modelVolumeLabel: string;
};

export default function MarketProductMeta({
  condition,

  hasModelInfo,

  modelKind,
  modelKindLabel,
  modelNumber,
  modelSize,

  hasColorInfo,
  modelColorName,
  modelColorCssValue,

  measurementsLabel,
  modelVolumeLabel,
}: MarketProductMetaProps) {
  if (
    !condition &&
    !hasModelInfo
  ) {
    return null;
  }

  return (
    <dl className="market-detail-page__meta">
      {condition ? (
        <div className="market-detail-page__meta-row">
          <dt>
            状態
          </dt>

          <dd>
            {condition}
          </dd>
        </div>
      ) : null}

      {hasModelInfo ? (
        <>
          {modelKind ? (
            <div className="market-detail-page__meta-row">
              <dt>
                種別
              </dt>

              <dd>
                {modelKindLabel}
              </dd>
            </div>
          ) : null}

          {modelNumber ? (
            <div className="market-detail-page__meta-row">
              <dt>
                モデル番号
              </dt>

              <dd>
                {modelNumber}
              </dd>
            </div>
          ) : null}

          {modelSize ? (
            <div className="market-detail-page__meta-row">
              <dt>
                サイズ
              </dt>

              <dd>
                {modelSize}
              </dd>
            </div>
          ) : null}

          {hasColorInfo ? (
            <div className="market-detail-page__meta-row">
              <dt>
                カラー
              </dt>

              <dd>
                <span className="market-detail-page__color-value">
                  {modelColorCssValue ? (
                    <span
                      className="market-detail-page__color-swatch"
                      style={{
                        backgroundColor:
                          modelColorCssValue,
                      }}
                      aria-hidden="true"
                    />
                  ) : null}

                  <span>
                    {modelColorName ||
                      modelColorCssValue ||
                      "カラー未設定"}
                  </span>
                </span>
              </dd>
            </div>
          ) : null}

          {measurementsLabel !==
          "-" ? (
            <div className="market-detail-page__meta-row">
              <dt>
                採寸
              </dt>

              <dd>
                {measurementsLabel}
              </dd>
            </div>
          ) : null}

          {modelVolumeLabel !==
          "-" ? (
            <div className="market-detail-page__meta-row">
              <dt>
                容量
              </dt>

              <dd>
                {modelVolumeLabel}
              </dd>
            </div>
          ) : null}
        </>
      ) : null}
    </dl>
  );
}