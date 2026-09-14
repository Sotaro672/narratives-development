// frontend/mall/src/features/landing/components/LandingHero.tsx

import { useNavigate } from "react-router-dom";

import Button from "../../../components/ui/Button";

export default function LandingHero() {
  const navigate = useNavigate();

  return (
    <section className="landing-page-hero">
      <div className="landing-page-hero__inner">
        <div className="landing-page-hero__content">
          <p className="landing-page-hero__eyebrow">
            二次流通まで繋げる真贋証明
          </p>

          <h1 className="landing-page-hero__title">
            AMOL
          </h1>

          <div className="page-actions">
            <Button
              variant="primary"
              onClick={() =>
                navigate("/how-to-use")
              }
            >
              使い方解説
            </Button>
          </div>
        </div>

        <div className="landing-page-hero__video-wrap">
          <iframe
            className="landing-page-hero__video"
            src="https://www.youtube.com/embed/fOH4hQUXwhc"
            title="AMOL 紹介動画"
            allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
            referrerPolicy="strict-origin-when-cross-origin"
            allowFullScreen
          />
        </div>
      </div>
    </section>
  );
}