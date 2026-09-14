// frontend/mall/src/features/landing/components/AntiCopySection.tsx

export default function AntiCopySection() {
  return (
    <div className="landing-page-anti-copy">
      <div className="landing-page-anti-copy__header">
        <h2 className="landing-page-section__title landing-page-anti-copy__title">
          QRコードがコピーされたらどう本物を判別する？
        </h2>
      </div>

      <div className="landing-page-anti-copy__body">
        <div className="landing-page-anti-copy__image-wrap">
          <img
            src="/antiCopy.png"
            alt="QRコードのコピーとブロックチェーントークンによる真贋判定のイメージ"
            className="landing-page-anti-copy__image"
            loading="lazy"
          />
        </div>

        <div className="landing-page-anti-copy__content">
          <p className="landing-page-anti-copy__text">
            QRコード自体はコピーできますが、
            ブロックチェーントークンの移譲履歴は枝分かれできません。
            模倣品偽造業者が１点の本物からコピー品を量産したとしても、正常に決済処理できるのは１点のみです。
            よって偽造業者は模造品から利益を上げることができません。
          </p>
        </div>
      </div>
    </div>
  );
}