// frontend/mall/src/features/howToUse/infrastructure/howToUseVideoUrl.ts

const storageBucket = import.meta.env.VITE_FIREBASE_STORAGE_BUCKET;

export function getHowToUseVideoUrl(path: string): string {
  if (!storageBucket) {
    throw new Error("VITE_FIREBASE_STORAGE_BUCKET is not configured.");
  }

  const encodedPath = encodeURIComponent(`how-to-use/${path}`);

  return `https://firebasestorage.googleapis.com/v0/b/${storageBucket}/o/${encodedPath}?alt=media`;
}