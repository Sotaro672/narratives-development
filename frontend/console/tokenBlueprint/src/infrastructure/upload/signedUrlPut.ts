// frontend/console/tokenBlueprint/src/infrastructure/upload/signedUrlPut.ts

export async function putFileToSignedUrl(
  uploadUrl: string,
  file: File,
  signedContentType?: string,
): Promise<void> {
  const url = String(uploadUrl || "").trim();
  if (!url) throw new Error("uploadUrl is empty");
  if (!file) throw new Error("file is empty");

  const ct =
    String(signedContentType || "").trim() ||
    file.type ||
    "application/octet-stream";

  const res = await fetch(url, {
    method: "PUT",
    headers: { "Content-Type": ct },
    body: file,
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(text || `GCS PUT failed: ${res.status}`);
  }
}
