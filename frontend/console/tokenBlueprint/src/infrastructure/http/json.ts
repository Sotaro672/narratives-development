// frontend/console/tokenBlueprint/src/infrastructure/http/json.ts

export async function handleJsonResponse<T>(res: Response): Promise<T> {
  const text = await res.text();

  if (!res.ok) {
    try {
      const data = JSON.parse(text);
      const msg = (data && (data.error || data.message)) || res.statusText;
      throw new Error(msg || `HTTP ${res.status}`);
    } catch {
      throw new Error(text || `HTTP ${res.status}`);
    }
  }

  if (!text) {
    return undefined as unknown as T;
  }

  try {
    return JSON.parse(text) as T;
  } catch {
    return text as unknown as T;
  }
}
