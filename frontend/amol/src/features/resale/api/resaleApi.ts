// frontend/amol/src/features/resale/api/resaleApi.ts
import { getDownloadURL, ref, uploadBytes } from "firebase/storage";

import { getApiBaseUrl } from "../../../lib/apiBaseUrl";
import { getFirebaseIdToken } from "../../../lib/authToken";
import { storage } from "../../../lib/firebase";

export type ResaleListing = {
  id?: string;
  status?: string;
  mintAddress?: string;
  tokenBlueprintId?: string;
  productId?: string;
  brandId?: string;
  productBlueprintId?: string;
  avatarId?: string;
  price?: number;
  condition?: string;
  description?: string;
  imageId?: string;

  productName?: string;
  tokenName?: string;
  brandName?: string;

  createdBy?: string;
  createdAt?: string;
  updatedBy?: string | null;
  updatedAt?: string | null;
};

export type ResaleConditionImage = {
  id: string;
  resaleId?: string;
  url: string;
  objectPath: string;
  fileName: string;
  fileSize: number;
  mimeType: string;
  displayOrder: number;
};

export type CreateResaleListingParams = {
  mintAddress: string;
  tokenBlueprintId: string;
  productId: string;
  brandId?: string;
  productBlueprintId?: string;
  avatarId?: string;
  price: number;
  condition: string;
  description: string;
  conditionImages: File[];
};

type ApiDataResponse<T> = {
  data?: T;
  error?: string;
};

function buildApiUrl(path: string): string {
  const baseUrl = getApiBaseUrl();

  if (!baseUrl) {
    return path;
  }

  return `${baseUrl}${path}`;
}

function createUploadImageID(): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }

  return `img_${Date.now()}_${Math.random().toString(36).slice(2)}`;
}

function sanitizeStorageFileName(fileName: string): string {
  const trimmed = fileName.trim();

  if (!trimmed) {
    return "image";
  }

  return trimmed.replace(/[^\w.\-()]/g, "_");
}

function nonEmptyOrUndefined(value: string | undefined): string | undefined {
  const normalized = value?.trim();
  return normalized ? normalized : undefined;
}

async function readApiJson<T>(res: Response): Promise<T> {
  return (await res.json().catch(() => ({}))) as T;
}

async function fetchWithAuth<T>(path: string, init?: RequestInit): Promise<T> {
  const token = await getFirebaseIdToken();

  const headers = new Headers(init?.headers);
  headers.set("Authorization", `Bearer ${token}`);

  if (init?.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const res = await fetch(buildApiUrl(path), {
    ...init,
    headers,
  });

  const json = await readApiJson<T & { error?: string }>(res);

  if (!res.ok) {
    throw new Error(json.error || "APIリクエストに失敗しました。");
  }

  return json;
}

async function uploadResaleConditionImage(params: {
  resaleId: string;
  file: File;
  displayOrder: number;
}): Promise<ResaleConditionImage> {
  const imageID = createUploadImageID();
  const safeFileName = sanitizeStorageFileName(params.file.name);
  const objectPath = `resale-condition-images/${params.resaleId}/${imageID}/${safeFileName}`;
  const storageRef = ref(storage, objectPath);
  const mimeType = params.file.type || "application/octet-stream";

  await uploadBytes(storageRef, params.file, {
    contentType: mimeType,
  });

  const url = await getDownloadURL(storageRef);

  return {
    id: imageID,
    resaleId: params.resaleId,
    url,
    objectPath,
    fileName: params.file.name,
    fileSize: params.file.size,
    mimeType,
    displayOrder: params.displayOrder,
  };
}

async function createResaleConditionImage(
  image: ResaleConditionImage,
): Promise<ResaleConditionImage | null> {
  if (!image.resaleId) {
    throw new Error("resaleId is required");
  }

  const json = await fetchWithAuth<ApiDataResponse<ResaleConditionImage>>(
    `/mall/me/resales/${encodeURIComponent(image.resaleId)}/images`,
    {
      method: "POST",
      body: JSON.stringify({
        id: image.id,
        url: image.url,
        displayOrder: image.displayOrder,
      }),
    },
  );

  return json.data ?? null;
}

async function setPrimaryResaleImage(params: {
  resaleId: string;
  imageId: string;
}): Promise<ResaleListing | null> {
  const json = await fetchWithAuth<ApiDataResponse<ResaleListing>>(
    `/mall/me/resales/${encodeURIComponent(params.resaleId)}/primary-image`,
    {
      method: "PUT",
      body: JSON.stringify({
        imageId: params.imageId,
      }),
    },
  );

  return json.data ?? null;
}

export async function createResaleListing(
  params: CreateResaleListingParams,
): Promise<ResaleListing | null> {
  const json = await fetchWithAuth<ApiDataResponse<ResaleListing>>(
    "/mall/me/resales",
    {
      method: "POST",
      body: JSON.stringify({
        mintAddress: params.mintAddress,
        tokenBlueprintId: params.tokenBlueprintId,
        productId: params.productId,
        brandId: nonEmptyOrUndefined(params.brandId),
        productBlueprintId: nonEmptyOrUndefined(params.productBlueprintId),
        avatarId: nonEmptyOrUndefined(params.avatarId),
        price: params.price,
        condition: params.condition,
        description: params.description,
      }),
    },
  );

  const created = json.data ?? null;
  const resaleId = created?.id;

  if (!resaleId) {
    return created;
  }

  const uploadedImages = await Promise.all(
    params.conditionImages.map((file, index) =>
      uploadResaleConditionImage({
        resaleId,
        file,
        displayOrder: index,
      }),
    ),
  );

  await Promise.all(uploadedImages.map(createResaleConditionImage));

  if (uploadedImages.length === 0) {
    return created;
  }

  const updated = await setPrimaryResaleImage({
    resaleId,
    imageId: uploadedImages[0].id,
  });

  return updated ?? created;
}

export type ListMyResaleListingsParams = {
  page?: number;
  perPage?: number;
};

export type ListMyResaleListingsResponse = {
  items?: ResaleListing[];
  totalCount?: number;
  totalPages?: number;
  page?: number;
  perPage?: number;
};

export async function listMyResaleListings(
  params: ListMyResaleListingsParams = {},
): Promise<ListMyResaleListingsResponse> {
  const searchParams = new URLSearchParams();

  searchParams.set("page", String(params.page ?? 1));
  searchParams.set("perPage", String(params.perPage ?? 50));

  return fetchWithAuth<ListMyResaleListingsResponse>(
    `/mall/me/resales?${searchParams.toString()}`,
    {
      method: "GET",
    },
  );
}

export type ListResaleListingsByAvatarIdParams = {
  avatarId: string;
  page?: number;
  perPage?: number;
};

export async function listResaleListingsByAvatarId(
  params: ListResaleListingsByAvatarIdParams,
): Promise<ListMyResaleListingsResponse> {
  const avatarId = params.avatarId.trim();

  if (!avatarId) {
    return {
      items: [],
      totalCount: 0,
      totalPages: 0,
      page: params.page ?? 1,
      perPage: params.perPage ?? 50,
    };
  }

  const searchParams = new URLSearchParams();

  searchParams.set("page", String(params.page ?? 1));
  searchParams.set("perPage", String(params.perPage ?? 50));

  return fetchWithAuth<ListMyResaleListingsResponse>(
    `/mall/resales/avatar/${encodeURIComponent(avatarId)}?${searchParams.toString()}`,
    {
      method: "GET",
    },
  );
}

export async function listMyResaleConditionImages(
  resaleId: string,
): Promise<ResaleConditionImage[]> {
  const id = resaleId.trim();

  if (!id) {
    return [];
  }

  const result = await fetchWithAuth<{
    data?: ResaleConditionImage[] | null;
    items?: ResaleConditionImage[];
    error?: string;
  }>(`/mall/me/resales/${encodeURIComponent(id)}/images`, {
    method: "GET",
  });

  return result.data ?? result.items ?? [];
}

export async function listPublicResaleConditionImages(
  resaleId: string,
): Promise<ResaleConditionImage[]> {
  const id = resaleId.trim();

  if (!id) {
    return [];
  }

  const result = await fetchWithAuth<{
    data?: ResaleConditionImage[] | null;
    items?: ResaleConditionImage[];
    error?: string;
  }>(`/mall/resales/${encodeURIComponent(id)}/images`, {
    method: "GET",
  });

  return result.data ?? result.items ?? [];
}