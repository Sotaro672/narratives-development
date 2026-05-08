// frontend/amol/src/pages/ScanPage.tsx
import { useCallback, useEffect, useRef, useState } from "react";
import { getAuth, onAuthStateChanged, User } from "firebase/auth";
import { useNavigate } from "react-router-dom";
import jsQR from "jsqr";

import "../styles/scan-page.css";

type BarcodeDetectorLike = {
  detect: (source: ImageBitmapSource) => Promise<Array<{ rawValue?: string }>>;
};

type ScanTarget =
  | {
      type: "product";
      productId: string;
    }
  | {
      type: "avatar";
      avatarId: string;
    };

declare global {
  interface Window {
    BarcodeDetector?: {
      new (options?: { formats?: string[] }): BarcodeDetectorLike;
      getSupportedFormats?: () => Promise<string[]>;
    };
  }
}

const SAFE_ID_PATTERN = /^[A-Za-z0-9_-]+$/;

function normalizeSafeId(raw: string): string | null {
  const value = raw.trim();
  if (!value) return null;

  const decoded = decodeURIComponent(value).trim();
  if (!decoded) return null;

  if (!SAFE_ID_PATTERN.test(decoded)) {
    return null;
  }

  return decoded;
}

/**
 * Product QR 正:
 *   https://amol.jp/{productId}
 *
 * Product QR 互換:
 *   https://amol.jp/scan/result?productId={productId}
 *   https://amol.jp/scan/result/{productId}
 *   /scan/result?productId={productId}
 *   /scan/result/{productId}
 *   productId={productId}
 *   raw productId
 *
 * Avatar Share QR:
 *   https://amol.jp/avatars/{avatarId}
 *   /avatars/{avatarId}
 */
function safeExtractScanTarget(rawText: string): ScanTarget | null {
  const trimmed = rawText.trim();
  if (!trimmed) return null;

  try {
    const url = new URL(trimmed);

    const pathParts = url.pathname.split("/").filter(Boolean);

    if (pathParts[0] === "avatars" && pathParts[1]) {
      const avatarId = normalizeSafeId(pathParts[1]);

      if (avatarId) {
        return {
          type: "avatar",
          avatarId,
        };
      }
    }

    const queryProductId =
      url.searchParams.get("productId") ||
      url.searchParams.get("productID") ||
      url.searchParams.get("id");

    if (queryProductId) {
      const productId = normalizeSafeId(queryProductId);

      if (productId) {
        return {
          type: "product",
          productId,
        };
      }
    }

    if (pathParts.length === 1) {
      const productId = normalizeSafeId(pathParts[0]);

      if (productId) {
        return {
          type: "product",
          productId,
        };
      }
    }

    const scanResultIndex = pathParts.findIndex(
      (part, index) => part === "scan" && pathParts[index + 1] === "result"
    );

    if (scanResultIndex >= 0) {
      const candidate = pathParts[scanResultIndex + 2];

      if (candidate) {
        const productId = normalizeSafeId(candidate);

        if (productId) {
          return {
            type: "product",
            productId,
          };
        }
      }
    }
  } catch {
    // URL でない場合は下の relative / plain text 処理へ
  }

  const avatarPathMatch = trimmed.match(/\/avatars\/([^/?#]+)(?:[?#].*)?$/);
  if (avatarPathMatch?.[1]) {
    const avatarId = normalizeSafeId(avatarPathMatch[1]);

    if (avatarId) {
      return {
        type: "avatar",
        avatarId,
      };
    }
  }

  const queryMatch = trimmed.match(/[?&](?:productId|productID|id)=([^&#]+)/);
  if (queryMatch?.[1]) {
    const productId = normalizeSafeId(queryMatch[1]);

    if (productId) {
      return {
        type: "product",
        productId,
      };
    }
  }

  const pathMatch = trimmed.match(/\/scan\/result\/([^/?#]+)(?:[?#].*)?$/);
  if (pathMatch?.[1]) {
    const productId = normalizeSafeId(pathMatch[1]);

    if (productId) {
      return {
        type: "product",
        productId,
      };
    }
  }

  const plainProductIdMatch = trimmed.match(/^\/?([A-Za-z0-9_-]+)$/);
  if (plainProductIdMatch?.[1]) {
    const productId = normalizeSafeId(plainProductIdMatch[1]);

    if (productId) {
      return {
        type: "product",
        productId,
      };
    }
  }

  return null;
}

export default function ScanPage() {
  const navigate = useNavigate();

  const videoRef = useRef<HTMLVideoElement | null>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const detectorRef = useRef<BarcodeDetectorLike | null>(null);
  const animationFrameRef = useRef<number | null>(null);
  const canvasRef = useRef<HTMLCanvasElement | null>(null);

  const [currentUser, setCurrentUser] = useState<User | null>(null);
  const [authResolved, setAuthResolved] = useState(false);

  const [error, setError] = useState("");
  const [scannerError, setScannerError] = useState("");
  const [startingCamera, setStartingCamera] = useState(true);
  const [scanningLocked, setScanningLocked] = useState(false);

  const stopCamera = useCallback(() => {
    if (animationFrameRef.current !== null) {
      cancelAnimationFrame(animationFrameRef.current);
      animationFrameRef.current = null;
    }

    if (streamRef.current) {
      streamRef.current.getTracks().forEach((track) => track.stop());
      streamRef.current = null;
    }

    if (videoRef.current) {
      videoRef.current.srcObject = null;
    }
  }, []);

  useEffect(() => {
    const auth = getAuth();

    const unsubscribe = onAuthStateChanged(auth, (user) => {
      setCurrentUser(user);
      setAuthResolved(true);
    });

    return () => unsubscribe();
  }, []);

  useEffect(() => {
    if (authResolved && !currentUser) {
      navigate("/signin", { replace: true });
    }
  }, [authResolved, currentUser, navigate]);

  const startCamera = useCallback(async () => {
    setError("");
    setScannerError("");
    setStartingCamera(true);

    if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
      setError("このブラウザではカメラを利用できません。");
      setStartingCamera(false);
      return;
    }

    try {
      stopCamera();

      let stream: MediaStream;

      try {
        stream = await navigator.mediaDevices.getUserMedia({
          video: {
            facingMode: { ideal: "environment" },
          },
          audio: false,
        });
      } catch {
        stream = await navigator.mediaDevices.getUserMedia({
          video: true,
          audio: false,
        });
      }

      streamRef.current = stream;

      if (videoRef.current) {
        videoRef.current.srcObject = stream;

        try {
          await videoRef.current.play();
        } catch {
          setError("カメラ映像の再生に失敗しました。");
        }
      }

      if (window.BarcodeDetector) {
        try {
          detectorRef.current = new window.BarcodeDetector({
            formats: ["qr_code"],
          });
        } catch {
          detectorRef.current = null;
        }
      } else {
        detectorRef.current = null;
      }
    } catch (e) {
      if (e instanceof Error) {
        setError(e.message);
      } else {
        setError("カメラの起動に失敗しました。");
      }
    } finally {
      setStartingCamera(false);
    }
  }, [stopCamera]);

  useEffect(() => {
    if (!authResolved) return;

    if (!currentUser) {
      setStartingCamera(false);
      return;
    }

    void startCamera();

    return () => {
      stopCamera();
    };
  }, [authResolved, currentUser, startCamera, stopCamera]);

  useEffect(() => {
    if (!authResolved || !currentUser) return;
    if (!videoRef.current || scanningLocked) return;

    let cancelled = false;

    const navigateToScanTarget = (target: ScanTarget) => {
      setScanningLocked(true);
      stopCamera();

      if (target.type === "avatar") {
        const encodedAvatarId = encodeURIComponent(target.avatarId);

        navigate(`/avatars/${encodedAvatarId}`, {
          replace: true,
        });
        return;
      }

      const encodedProductId = encodeURIComponent(target.productId);

      navigate(`/scan/result?productId=${encodedProductId}`, {
        replace: true,
      });
    };

    const tryHandleRawValue = (rawValue: string) => {
      const target = safeExtractScanTarget(rawValue);

      if (!target) {
        return false;
      }

      navigateToScanTarget(target);
      return true;
    };

    const scanWithJsQR = () => {
      const video = videoRef.current;
      const canvas = canvasRef.current;

      if (!video || !canvas) return false;

      const width = video.videoWidth;
      const height = video.videoHeight;

      if (!width || !height) return false;

      canvas.width = width;
      canvas.height = height;

      const context = canvas.getContext("2d", {
        willReadFrequently: true,
      });

      if (!context) return false;

      context.drawImage(video, 0, 0, width, height);

      const imageData = context.getImageData(0, 0, width, height);
      const result = jsQR(imageData.data, imageData.width, imageData.height, {
        inversionAttempts: "attemptBoth",
      });

      if (!result?.data) return false;

      return tryHandleRawValue(result.data);
    };

    const scanLoop = async () => {
      if (cancelled || !videoRef.current) return;

      try {
        if (videoRef.current.readyState >= 2) {
          let detected = false;

          if (detectorRef.current) {
            try {
              const barcodes = await detectorRef.current.detect(
                videoRef.current
              );
              const rawValue = barcodes[0]?.rawValue ?? "";

              if (rawValue) {
                detected = tryHandleRawValue(rawValue);
              }
            } catch {
              detected = false;
            }
          }

          if (!detected) {
            detected = scanWithJsQR();
          }

          if (!detected && scannerError) {
            setScannerError("");
          }
        }
      } catch {
        setScannerError("QRコードの読み取りに失敗しました。");
      }

      animationFrameRef.current = requestAnimationFrame(() => {
        void scanLoop();
      });
    };

    animationFrameRef.current = requestAnimationFrame(() => {
      void scanLoop();
    });

    return () => {
      cancelled = true;

      if (animationFrameRef.current !== null) {
        cancelAnimationFrame(animationFrameRef.current);
        animationFrameRef.current = null;
      }
    };
  }, [
    authResolved,
    currentUser,
    navigate,
    scanningLocked,
    scannerError,
    stopCamera,
  ]);

  return (
    <div className="scan-page">
      <video
        ref={videoRef}
        autoPlay
        playsInline
        muted
        className="scan-page__video"
      />

      <canvas ref={canvasRef} className="scan-page__hidden-canvas" />

      <div className="scan-page__back-layer">
        <button
          type="button"
          onClick={() => navigate("/lists")}
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
            商品またはプロフィール QR コードを枠内に合わせてください
          </p>
        </div>
      </div>

      {startingCamera ? (
        <div className="scan-page__error-overlay">
          <div className="scan-page__error-box">
            <p className="scan-page__error-text">カメラを起動しています...</p>
          </div>
        </div>
      ) : null}

      {error ? (
        <div className="scan-page__error-overlay">
          <div className="scan-page__error-box">
            <p className="scan-page__error-text">{error}</p>
          </div>
        </div>
      ) : null}

      {scannerError ? (
        <div className="scan-page__error-overlay">
          <div className="scan-page__error-box">
            <p className="scan-page__error-text">{scannerError}</p>
          </div>
        </div>
      ) : null}
    </div>
  );
}