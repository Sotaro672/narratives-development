// frontend/order/src/infrastructure/mockdata/user_mockdata.ts

import type { User } from "../../../../shell/src/shared/types/user";

/**
 * モック用 User データ
 * frontend/shell/src/shared/types/user.ts に準拠。
 *
 * - createdAt / updatedAt / deletedAt は ISO8601 UTC 形式
 * - first_name_kana / last_name_kana は日本語表記例
 */

export const USERS: User[] = [
  {
    id: "user_001",
    first_name: "美咲",
    first_name_kana: "ミサキ",
    last_name: "佐藤",
    last_name_kana: "サトウ",
    email: "misaki.sato@example.com",
    phone_number: "+819012345678",
    createdAt: "2024-01-05T09:00:00Z",
    updatedAt: "2024-03-10T10:00:00Z",
    deletedAt: "2099-12-31T00:00:00Z",
  },
  {
    id: "user_002",
    first_name: "翔太",
    first_name_kana: "ショウタ",
    last_name: "田中",
    last_name_kana: "タナカ",
    email: "shota.tanaka@example.com",
    phone_number: "+819012345679",
    createdAt: "2024-02-01T11:30:00Z",
    updatedAt: "2024-03-15T14:00:00Z",
    deletedAt: "2099-12-31T00:00:00Z",
  },
  {
    id: "user_003",
    first_name: "花子",
    first_name_kana: "ハナコ",
    last_name: "鈴木",
    last_name_kana: "スズキ",
    email: "hanako.suzuki@example.com",
    phone_number: "+819012345680",
    createdAt: "2024-03-01T08:45:00Z",
    updatedAt: "2024-04-20T09:15:00Z",
    deletedAt: "2099-12-31T00:00:00Z",
  },
  {
    id: "user_004",
    first_name: "健一",
    first_name_kana: "ケンイチ",
    last_name: "山本",
    last_name_kana: "ヤマモト",
    email: "kenichi.yamamoto@example.com",
    phone_number: "+819012345681",
    createdAt: "2024-03-12T10:00:00Z",
    updatedAt: "2024-04-10T10:30:00Z",
    deletedAt: "2099-12-31T00:00:00Z",
  },
  {
    id: "user_005",
    first_name: "真理",
    first_name_kana: "マリ",
    last_name: "中村",
    last_name_kana: "ナカムラ",
    email: "mari.nakamura@example.com",
    phone_number: "+819012345682",
    createdAt: "2024-03-25T07:20:00Z",
    updatedAt: "2024-04-01T09:00:00Z",
    deletedAt: "2099-12-31T00:00:00Z",
  },
];

/**
 * ID検索ヘルパー
 */
export function getUserById(id: string): User | undefined {
  return USERS.find((u) => u.id === id);
}

/**
 * 全ユーザーのメール一覧を取得
 */
export function getAllUserEmails(): string[] {
  return USERS.map((u) => u.email ?? "").filter(Boolean);
}
