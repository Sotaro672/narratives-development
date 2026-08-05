// frontend/amol/src/features/scan/presentation/hooks/useScanPage.ts

import {
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";
import {
  useNavigate,
} from "react-router-dom";
import jsQR from "jsqr";

import {
  useAuthState,
} from "../../../shared/hooks/useAuthState";

import {
  extractProductIdFromQr,
} from "../../utils/extractProductIdFromQr";

type BarcodeDetectorLike = {
  detect: (
    source: ImageBitmapSource,
  ) => Promise<
    Array<{
      rawValue?: string;
    }>
  >;
};

declare global {
  interface Window {
    BarcodeDetector?: {
      new (
        options?: {
          formats?: string[];
        },
      ): BarcodeDetectorLike;
    };
  }
}

export function useScanPage() {
  const navigate =
    useNavigate();

  const {
    authResolved,
    isLoggedIn,
  } = useAuthState();

  const videoRef =
    useRef<HTMLVideoElement | null>(
      null,
    );

  const streamRef =
    useRef<MediaStream | null>(
      null,
    );

  const detectorRef =
    useRef<BarcodeDetectorLike | null>(
      null,
    );

  const animationFrameRef =
    useRef<number | null>(
      null,
    );

  const canvasRef =
    useRef<HTMLCanvasElement | null>(
      null,
    );

  const [
    error,
    setError,
  ] = useState("");

  const [
    scannerError,
    setScannerError,
  ] = useState("");

  const [
    startingCamera,
    setStartingCamera,
  ] = useState(true);

  const [
    scanningLocked,
    setScanningLocked,
  ] = useState(false);

  const stopCamera =
    useCallback(() => {
      if (
        animationFrameRef.current !==
        null
      ) {
        cancelAnimationFrame(
          animationFrameRef.current,
        );

        animationFrameRef.current =
          null;
      }

      if (
        streamRef.current
      ) {
        streamRef.current
          .getTracks()
          .forEach(
            (track) => {
              track.stop();
            },
          );

        streamRef.current =
          null;
      }

      if (
        videoRef.current
      ) {
        videoRef.current.srcObject =
          null;
      }

      detectorRef.current =
        null;
    }, []);

  const handleBack =
    useCallback(() => {
      navigate("/lists");
    }, [
      navigate,
    ]);

  useEffect(() => {
    if (
      authResolved &&
      !isLoggedIn
    ) {
      navigate(
        "/signin",
        {
          replace: true,
        },
      );
    }
  }, [
    authResolved,
    isLoggedIn,
    navigate,
  ]);

  const startCamera =
    useCallback(
      async () => {
        setError("");
        setScannerError("");
        setStartingCamera(
          true,
        );

        if (
          !navigator.mediaDevices ||
          !navigator.mediaDevices
            .getUserMedia
        ) {
          setError(
            "このブラウザではカメラを利用できません。",
          );

          setStartingCamera(
            false,
          );

          return;
        }

        try {
          stopCamera();

          let stream:
            MediaStream;

          try {
            stream =
              await navigator.mediaDevices.getUserMedia(
                {
                  video: {
                    facingMode: {
                      ideal:
                        "environment",
                    },
                  },
                  audio: false,
                },
              );
          } catch {
            stream =
              await navigator.mediaDevices.getUserMedia(
                {
                  video: true,
                  audio: false,
                },
              );
          }

          streamRef.current =
            stream;

          if (
            videoRef.current
          ) {
            videoRef.current.srcObject =
              stream;

            try {
              await videoRef.current.play();
            } catch {
              setError(
                "カメラ映像の再生に失敗しました。",
              );
            }
          }

          if (
            window.BarcodeDetector
          ) {
            try {
              detectorRef.current =
                new window.BarcodeDetector(
                  {
                    formats: [
                      "qr_code",
                    ],
                  },
                );
            } catch {
              detectorRef.current =
                null;
            }
          } else {
            detectorRef.current =
              null;
          }
        } catch (caughtError) {
          if (
            caughtError instanceof
            Error
          ) {
            setError(
              caughtError.message,
            );
          } else {
            setError(
              "カメラの起動に失敗しました。",
            );
          }
        } finally {
          setStartingCamera(
            false,
          );
        }
      },
      [
        stopCamera,
      ],
    );

  useEffect(() => {
    if (!authResolved) {
      return;
    }

    if (!isLoggedIn) {
      setStartingCamera(
        false,
      );

      return;
    }

    void startCamera();

    return () => {
      stopCamera();
    };
  }, [
    authResolved,
    isLoggedIn,
    startCamera,
    stopCamera,
  ]);

  useEffect(() => {
    if (
      !authResolved ||
      !isLoggedIn
    ) {
      return;
    }

    if (
      !videoRef.current ||
      scanningLocked
    ) {
      return;
    }

    let cancelled =
      false;

    const navigateToProduct =
      (
        productId: string,
      ) => {
        setScanningLocked(
          true,
        );

        stopCamera();

        const encodedProductId =
          encodeURIComponent(
            productId,
          );

        navigate(
          `/scan/result?productId=${encodedProductId}`,
          {
            replace: true,
          },
        );
      };

    const tryHandleRawValue =
      (
        rawValue: string,
      ): boolean => {
        const productId =
          extractProductIdFromQr(
            rawValue,
          );

        if (!productId) {
          return false;
        }

        navigateToProduct(
          productId,
        );

        return true;
      };

    const scanWithJsQR =
      (): boolean => {
        const video =
          videoRef.current;

        const canvas =
          canvasRef.current;

        if (
          !video ||
          !canvas
        ) {
          return false;
        }

        const width =
          video.videoWidth;

        const height =
          video.videoHeight;

        if (
          !width ||
          !height
        ) {
          return false;
        }

        canvas.width =
          width;

        canvas.height =
          height;

        const context =
          canvas.getContext(
            "2d",
            {
              willReadFrequently:
                true,
            },
          );

        if (!context) {
          return false;
        }

        context.drawImage(
          video,
          0,
          0,
          width,
          height,
        );

        const imageData =
          context.getImageData(
            0,
            0,
            width,
            height,
          );

        const result =
          jsQR(
            imageData.data,
            imageData.width,
            imageData.height,
            {
              inversionAttempts:
                "attemptBoth",
            },
          );

        if (
          !result?.data
        ) {
          return false;
        }

        return tryHandleRawValue(
          result.data,
        );
      };

    const scanLoop =
      async () => {
        if (
          cancelled ||
          !videoRef.current
        ) {
          return;
        }

        try {
          if (
            videoRef.current
              .readyState >= 2
          ) {
            let detected =
              false;

            if (
              detectorRef.current
            ) {
              try {
                const barcodes =
                  await detectorRef.current.detect(
                    videoRef.current,
                  );

                const rawValue =
                  barcodes[0]
                    ?.rawValue ??
                  "";

                if (rawValue) {
                  detected =
                    tryHandleRawValue(
                      rawValue,
                    );
                }
              } catch {
                detected =
                  false;
              }
            }

            if (!detected) {
              detected =
                scanWithJsQR();
            }

            if (
              !detected &&
              scannerError
            ) {
              setScannerError(
                "",
              );
            }
          }
        } catch {
          setScannerError(
            "QRコードの読み取りに失敗しました。",
          );
        }

        if (
          cancelled ||
          scanningLocked
        ) {
          return;
        }

        animationFrameRef.current =
          requestAnimationFrame(
            () => {
              void scanLoop();
            },
          );
      };

    animationFrameRef.current =
      requestAnimationFrame(
        () => {
          void scanLoop();
        },
      );

    return () => {
      cancelled =
        true;

      if (
        animationFrameRef.current !==
        null
      ) {
        cancelAnimationFrame(
          animationFrameRef.current,
        );

        animationFrameRef.current =
          null;
      }
    };
  }, [
    authResolved,
    isLoggedIn,
    navigate,
    scanningLocked,
    scannerError,
    stopCamera,
  ]);

  return {
    videoRef,
    canvasRef,

    error,
    scannerError,
    startingCamera,

    handleBack,
  };
}