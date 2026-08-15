// frontend/console/shell/src/shared/types/user.ts

/**
 * Backend BFFのUser responseを正とする。
 */
export interface User {
  id: string;
  first_name: string;
  first_name_kana: string;
  last_name_kana: string;
  last_name: string;
  createdAt: string;
  updatedAt: string;
}