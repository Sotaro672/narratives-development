// frontend/amol/src/features/scan/presentation/components/ScanView.tsx

import {
  useScanPage,
} from "../hooks/useScanPage";

import "../styles/scan-page.css";

type ScanStatusOverlayProps = {
  message: string;
};

function ScanStatusOverlay({
  message,
}: ScanStatusOverlayProps) {
  if (!message) {
    return null;
  }

  return (
    <div className="scan-page__error-overlay">
      <div className="scan-page__error-box">
        <p className="scan-page__error-text">
          {message}
        </p>
      </div>
    </div>
  );
}

export default function ScanView() {
  const {
    videoRef,
    canvasRef,
    error,
    scannerError,
    startingCamera,
    handleBack,
  } = useScanPage();

  return (
    <div className="scan-page">
      <video
        ref={videoRef}
        autoPlay
        playsInline
        muted
        className="scan-page__video"
      />

      <canvas
        ref={canvasRef}
        className="scan-page__hidden-canvas"
      />

      <div className="scan-page__back-layer">
        <button
          type="button"
          onClick={
            handleBack
          }
          aria-label="戻る"
          className="scan-page__back-button"
        >
          ←
        </button>
      </div>

      <div className="scan-page__overlay">
        <div className="scan-page__center-guide">
          <div className="scan-page__frame" />

          <p className="scan-page__guide-text">
            商品 QR コードを枠内に合わせてください
          </p>
        </div>
      </div>

      {startingCamera ? (
        <ScanStatusOverlay
          message="カメラを起動しています..."
        />
      ) : null}

      <ScanStatusOverlay
        message={error}
      />

      <ScanStatusOverlay
        message={
          scannerError
        }
      />
    </div>
  );
}