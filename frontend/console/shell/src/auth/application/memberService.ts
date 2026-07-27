// frontend/console/shell/src/auth/application/memberService.ts
/// <reference types="vite/client" />

import type { MemberDTO } from "../../shared/types/member";
import {
  fetchCurrentMemberRaw,
  updateCurrentMemberProfileRaw,
} from "../infrastructure/repository/authRepositoryHTTP";

type UnknownRecord = Record<string, unknown>;

function isRecord(value: unknown): value is UnknownRecord {
  return (
    typeof value === "object" &&
    value !== null &&
    !Array.isArray(value)
  );
}

function toStringValue(value: unknown): string {
  return typeof value === "string"
    ? value.trim()
    : "";
}

function toNullableString(
  value: unknown,
): string | null {
  if (typeof value !== "string") {
    return null;
  }

  const normalized = value.trim();

  return normalized || null;
}

function toStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) {
    return [];
  }

  return value
    .filter(
      (item): item is string =>
        typeof item === "string",
    )
    .map((item) => item.trim())
    .filter((item) => item.length > 0);
}

function toAssignedBrands(
  value: unknown,
): string[] | null {
  const assignedBrands = toStringArray(value);

  return assignedBrands.length > 0
    ? assignedBrands
    : null;
}

// -------------------------------
// 共通: 生JSON → MemberDTO変換
// -------------------------------

function mapRawToMemberDTO(
  raw: unknown,
  fallbackEmail?: string | null,
): MemberDTO {
  if (!isRecord(raw)) {
    throw new Error(
      "現在のメンバーレスポンスの形式が不正です。",
    );
  }

  const firstName = toStringValue(raw.firstName);
  const lastName = toStringValue(raw.lastName);

  const displayNameFromResponse = toStringValue(
    raw.displayName,
  );

  const displayNameFromNameParts = [
    lastName,
    firstName,
  ]
    .filter((value) => value.length > 0)
    .join(" ");

  const responseEmail = toStringValue(raw.email);
  const normalizedFallbackEmail =
    typeof fallbackEmail === "string"
      ? fallbackEmail.trim()
      : "";

  return {
    // Backend responseのidはFirestore membersのdocId
    id: toStringValue(raw.id),

    // Firebase Authentication UIDはBackend responseを正とする
    uid: toStringValue(raw.uid),

    firstName,
    lastName,
    firstNameKana: toStringValue(
      raw.firstNameKana,
    ),
    lastNameKana: toStringValue(
      raw.lastNameKana,
    ),

    email:
      responseEmail ||
      normalizedFallbackEmail,

    permissions: toStringArray(
      raw.permissions,
    ),

    assignedBrands: toAssignedBrands(
      raw.assignedBrands,
    ),

    companyId: toStringValue(raw.companyId),
    status: toStringValue(raw.status),

    createdAt: toStringValue(raw.createdAt),
    updatedAt: toNullableString(
      raw.updatedAt,
    ),
    updatedBy: toNullableString(
      raw.updatedBy,
    ),

    // Backend responseのdisplayNameを正とし、
    // 存在しない場合のみ姓名から生成する
    displayName:
      displayNameFromResponse ||
      displayNameFromNameParts,
  };
}

// -------------------------------
// 現在メンバー取得
// -------------------------------

export async function fetchCurrentMember(): Promise<MemberDTO | null> {
  const raw = await fetchCurrentMemberRaw();

  if (!raw) {
    return null;
  }

  return mapRawToMemberDTO(raw);
}

// -------------------------------
// プロファイル更新
// -------------------------------

export type UpdateMemberProfileInput = {
  // PATCH /members/{docId}用
  id: string;
  firstName: string;
  lastName: string;
  firstNameKana: string;
  lastNameKana: string;
  email?: string | null;
};

type UpdateMemberProfilePayload = {
  firstName: string;
  lastName: string;
  firstNameKana: string;
  lastNameKana: string;
  email?: string | null;
};

export async function updateCurrentMemberProfile(
  input: UpdateMemberProfileInput,
): Promise<MemberDTO | null> {
  const payload: UpdateMemberProfilePayload = {
    firstName: input.firstName,
    lastName: input.lastName,
    firstNameKana: input.firstNameKana,
    lastNameKana: input.lastNameKana,
  };

  if (input.email !== undefined) {
    payload.email = input.email;
  }

  const raw = await updateCurrentMemberProfileRaw(
    input.id,
    payload,
  );

  if (!raw) {
    return null;
  }

  return mapRawToMemberDTO(
    raw,
    input.email ?? null,
  );
}