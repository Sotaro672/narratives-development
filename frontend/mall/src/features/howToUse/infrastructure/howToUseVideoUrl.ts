// frontend/mall/src/features/howToUse/infrastructure/howToUseVideoUrl.ts

const storageBucket = import.meta.env.VITE_FIREBASE_STORAGE_BUCKET;
const HOW_TO_USE_ROOT_PATH = "how-to-use";

export function getHowToUseVideoUrl(path: string): string {
  if (!storageBucket) {
    throw new Error("VITE_FIREBASE_STORAGE_BUCKET is not configured.");
  }

  const normalizedPath = path.replace(/^\/+|\/+$/g, "");

  if (!normalizedPath) {
    throw new Error("How-to-use video path is required.");
  }

  const objectPath = `${HOW_TO_USE_ROOT_PATH}/${normalizedPath}`;
  const encodedPath = encodeURIComponent(objectPath);

  return `https://firebasestorage.googleapis.com/v0/b/${storageBucket}/o/${encodedPath}?alt=media`;
}