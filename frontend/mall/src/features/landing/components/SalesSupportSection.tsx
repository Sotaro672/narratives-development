// frontend/mall/src/features/landing/components/SalesSupportSection.tsx

import type { RefObject } from "react";
import { useNavigate } from "react-router-dom";

import Button from "../../../components/ui/Button";

import afterConnection1Image from "../assets/AfterConnection1.png";
import afterConnection2Image from "../assets/AfterConnection2.png";
import beforeConnection1Image from "../assets/BeforeConnection1.png";
import beforeConnection2Image from "../assets/BeforeConnection2.png";

type SalesSupportSectionProps = {
  salesSupportEyebrowRef: RefObject<HTMLParagraphElement>;
};

export default function SalesSupportSection({
  salesSupportEyebrowRef,
}: SalesSupportSectionProps) {
  const navigate = useNavigate();

  return (
    <section
      id="sales-support"
      className="landing-page-section landing-page-sales-support"
    >
      <div className="landing-page-section__inner">
        <div className="landing-page-sales-support__header">
          <p
            ref={salesSupportEyebrowRef}
            className="landing-page-sales-support__eyebrow"
          >
            営業支援
          </p>

          <h2 className="landing-page-section__title landing-page-sales-support__title">
            購入後もお客様とつながる
          </h2>

          <p className="landing-page-card__text landing-page-sales-support__lead">
            電子名札を通じて、生産者は販売した商品の現在の所有者と繋がることができます。
          </p>
        </div>

        <div className="landing-page-sales-support__benefits">
          <article className="landing-page-sales-support-benefit landing-page-sales-support-benefit--with-image">
            <div className="landing-page-sales-support-benefit__content">
              <p className="landing-page-sales-support-card__label">
                これまでの課題
              </p>

              <h3 className="landing-page-sales-support-benefit__title">
                従来のSNSではお客様からフォローしてもらう必要があった。
              </h3>

              <p className="landing-page-sales-support-benefit__text">
                従来のSNSでは折角商品を気に入ってもらっても、お客様からフォローをして頂かないと商品を忘れてしまう可能性があります。
              </p>

              <div className="landing-page-sales-support-benefit__image-wrap">
                <img
                  src={beforeConnection1Image}
                  alt="従来のSNSを用いたお客様との繋がり"
                  className="landing-page-sales-support-benefit__image"
                  loading="lazy"
                />
              </div>

              <h3 className="landing-page-sales-support-benefit__title">
                二次流通を追跡できない
              </h3>

              <p className="landing-page-sales-support-benefit__text">
                従来のSNSでは販売した商品が年々拡大する二次流通市場でどの様なお客様に購入されているのかを追跡することができません。
              </p>

              <div className="landing-page-sales-support-benefit__image-wrap">
                <img
                  src={beforeConnection2Image}
                  alt="二次流通でお客様との繋がりを追跡できない状態"
                  className="landing-page-sales-support-benefit__image"
                  loading="lazy"
                />
              </div>
            </div>
          </article>

          <article className="landing-page-sales-support-benefit landing-page-sales-support-benefit--with-image">
            <div className="landing-page-sales-support-benefit__content">
              <p className="landing-page-sales-support-card__label">
                AMOLでできること
              </p>

              <h3 className="landing-page-sales-support-benefit__title">
                電子名札を介したお客様との繋がり
              </h3>

              <p className="landing-page-sales-support-benefit__text">
                商品を購入していただき、オプトイン操作をしていただいた全てのお客様にお知らせを送ることができます。
              </p>

              <div className="landing-page-sales-support-benefit__image-wrap">
                <img
                  src={afterConnection1Image}
                  alt="電子名札を介したお客様との繋がり"
                  className="landing-page-sales-support-benefit__image"
                  loading="lazy"
                />
              </div>

              <h3 className="landing-page-sales-support-benefit__title">
                二次流通市場でも現在の所有者と繋がれる
              </h3>

              <p className="landing-page-sales-support-benefit__text">
                お客様間で電子名札を譲渡することも可能です。二次流通市場で購入されたお客様にもお知らせをお届けすることができます。
              </p>

              <div className="landing-page-sales-support-benefit__image-wrap">
                <img
                  src={afterConnection2Image}
                  alt="二次流通市場でも現在の所有者と繋がれる状態"
                  className="landing-page-sales-support-benefit__image"
                  loading="lazy"
                />
              </div>
            </div>
          </article>
        </div>

        <div className="page-actions">
          <Button
            variant="primary"
            onClick={() => navigate("/how-to-use")}
          >
            使い方解説
          </Button>
        </div>
      </div>
    </section>
  );
}