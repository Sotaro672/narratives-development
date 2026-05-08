// frontend/src/features/auth/types.ts

export type CreateAccountResult =
  | {
      ok: true;
      email: string;
    }
  | {
      ok: false;
      error: string;
    };

export type CreateAccountParams = {
  emailRaw: string;
  password: string;
  passwordConfirmation: string;
  agree: boolean;
};