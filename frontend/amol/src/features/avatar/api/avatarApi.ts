// frontend/amol/src/features/avatar/api/avatarApi.ts

import { buildApiUrl } from "../../../lib/apiBaseUrl";

import type {
  CreateAvatarPayload,
  CreateAvatarResponse,
  MyAvatarResponse,
  UpdateAvatarPayload,
  UpdateAvatarResponse,
} from "../../shared/types/avatar";

type AuthedRequestParams = {
  backendUrl: string;
  idToken: string;
};

type GetPublicAvatarParams =
  AuthedRequestParams & {
    avatarId: string;
  };

async function readApiError(
  response: Response,
): Promise<string> {
  const contentType =
    response.headers.get("content-type") || "";

  if (
    contentType.includes(
      "application/json",
    )
  ) {
    const body = (await response.json().catch(
      () => null,
    )) as
      | {
          error?: string;
          message?: string;
        }
      | null;

    if (body?.error) {
      return body.error;
    }

    if (body?.message) {
      return body.message;
    }
  }

  const text = await response
    .text()
    .catch(() => "");

  return (
    text ||
    `API request failed (${response.status})`
  );
}

function unwrapData<T>(
  body: unknown,
): T {
  if (
    body &&
    typeof body === "object" &&
    "data" in body &&
    (body as { data?: unknown }).data
  ) {
    return (body as { data: T }).data;
  }

  return body as T;
}

function buildAuthHeaders(
  idToken: string,
): HeadersInit {
  return {
    Accept: "application/json",
    Authorization: `Bearer ${idToken}`,
  };
}

export async function getMyAvatar({
  backendUrl,
  idToken,
}: AuthedRequestParams): Promise<MyAvatarResponse | null> {
  const response = await fetch(
    buildApiUrl(
      backendUrl,
      "/mall/me/avatars",
    ),
    {
      method: "GET",
      headers: buildAuthHeaders(idToken),
    },
  );

  if (response.status === 404) {
    return null;
  }

  if (!response.ok) {
    throw new Error(
      await readApiError(response),
    );
  }

  const body: unknown =
    await response.json();

  return unwrapData<MyAvatarResponse>(
    body,
  );
}

export async function getPublicAvatar({
  backendUrl,
  idToken,
  avatarId,
}: GetPublicAvatarParams): Promise<MyAvatarResponse | null> {
  const normalizedAvatarId =
    avatarId.trim();

  if (!normalizedAvatarId) {
    throw new Error(
      "avatarIdが指定されていません。",
    );
  }

  const response = await fetch(
    buildApiUrl(
      backendUrl,
      `/mall/avatars/${encodeURIComponent(
        normalizedAvatarId,
      )}`,
    ),
    {
      method: "GET",
      headers: buildAuthHeaders(idToken),
    },
  );

  if (response.status === 404) {
    return null;
  }

  if (!response.ok) {
    throw new Error(
      await readApiError(response),
    );
  }

  const body: unknown =
    await response.json();

  const avatar =
    unwrapData<MyAvatarResponse>(body);

  return {
    ...avatar,
    avatarId:
      avatar.avatarId ||
      normalizedAvatarId,
  };
}

export async function createAvatar({
  backendUrl,
  idToken,
  payload,
}: AuthedRequestParams & {
  payload: CreateAvatarPayload;
}): Promise<CreateAvatarResponse> {
  const response = await fetch(
    buildApiUrl(
      backendUrl,
      "/mall/avatars",
    ),
    {
      method: "POST",
      headers: {
        ...buildAuthHeaders(idToken),
        "Content-Type":
          "application/json",
      },
      body: JSON.stringify(payload),
    },
  );

  if (!response.ok) {
    throw new Error(
      await readApiError(response),
    );
  }

  const body: unknown =
    await response.json();

  return unwrapData<CreateAvatarResponse>(
    body,
  );
}

export async function updateAvatar({
  backendUrl,
  idToken,
  avatarId,
  payload,
}: AuthedRequestParams & {
  avatarId: string;
  payload: UpdateAvatarPayload;
}): Promise<UpdateAvatarResponse> {
  void avatarId;

  const response = await fetch(
    buildApiUrl(
      backendUrl,
      "/mall/me/avatars",
    ),
    {
      method: "PATCH",
      headers: {
        ...buildAuthHeaders(idToken),
        "Content-Type":
          "application/json",
      },
      body: JSON.stringify(payload),
    },
  );

  if (!response.ok) {
    throw new Error(
      await readApiError(response),
    );
  }

  const body: unknown =
    await response
      .json()
      .catch(() => ({ avatarId }));

  return unwrapData<UpdateAvatarResponse>(
    body,
  );
}