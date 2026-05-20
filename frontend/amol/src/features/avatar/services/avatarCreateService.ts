// frontend/src/features/avatar/services/avatarCreateService.ts

import type { Auth } from "firebase/auth";
import { getDownloadURL, ref, uploadBytes } from "firebase/storage";

import { storage } from "../../../lib/firebase";
import { createAvatar, getMyAvatar, updateAvatar } from "../api/avatarApi";
import type {
  AvatarCreateResult,
  AvatarUpdateResult,
  MyAvatarResponse,
  PickIconResult,
} from "../types/avatarCreateTypes";

type AvatarCreateServiceParams = {
  auth: Auth;
  backendUrl: string;
};

type SaveAvatarParams = {
  avatarNameRaw: string;
  profileRaw: string;
  externalLinkRaw: string;
  iconFile: File | null;
};

type UpdateAvatarParams = SaveAvatarParams & {
  avatarId: string;
};

export class AvatarCreateService {
  private readonly auth: Auth;
  private readonly backendUrl: string;

  constructor({ auth, backendUrl }: AvatarCreateServiceParams) {
    this.auth = auth;
    this.backendUrl = backendUrl;
  }

  s(value: string | null | undefined): string {
    return (value ?? "").trim();
  }

  backTo(from: string | null): string {
    const f = this.s(from);
    if (f) return f;
    return "/lists";
  }

  isValidUrlOrEmpty(value: string): boolean {
    const v = this.s(value);
    if (!v) return true;

    try {
      const url = new URL(v);
      return (
        (url.protocol === "http:" || url.protocol === "https:") && !!url.host
      );
    } catch {
      return false;
    }
  }

  pickIconWeb(file: File | null): PickIconResult | null {
    if (!file) return null;

    if (!file.type.startsWith("image/")) {
      return {
        file: null,
        fileName: null,
        mimeType: null,
        previewUrl: null,
        error: "画像ファイルを選択してください。",
      };
    }

    return {
      file,
      fileName: file.name || null,
      mimeType: file.type || null,
      previewUrl: URL.createObjectURL(file),
    };
  }

  private ensureMimeType(file: File): string {
    const mime = this.s(file.type);
    if (mime) return mime;

    const name = this.s(file.name).toLowerCase();

    if (name.endsWith(".png")) return "image/png";
    if (name.endsWith(".jpg") || name.endsWith(".jpeg")) return "image/jpeg";
    if (name.endsWith(".webp")) return "image/webp";
    if (name.endsWith(".gif")) return "image/gif";

    return "application/octet-stream";
  }

  private ensureSupportedImage(file: File): string {
    const mimeType = this.ensureMimeType(file).toLowerCase();

    switch (mimeType) {
      case "image/png":
      case "image/jpeg":
      case "image/jpg":
      case "image/webp":
      case "image/gif":
        return mimeType;
      default:
        throw new Error(
          "対応していない画像形式です。png, jpg, webp, gif を選択してください。",
        );
    }
  }

  private async getIdToken(): Promise<string> {
    const user = this.auth.currentUser;

    if (!user) {
      throw new Error("サインインが必要です。");
    }

    const token = await user.getIdToken(true);

    if (!token) {
      throw new Error("認証トークンが取得できませんでした。再ログインしてください。");
    }

    return token;
  }

  private avatarIconStoragePath(avatarId: string): string {
    const id = this.s(avatarId);

    if (!id) {
      throw new Error("avatarId が取得できませんでした。");
    }

    return `avatar-icons/${id}/icon`;
  }

  private async uploadAvatarIconToFirebaseStorage({
    avatarId,
    iconFile,
  }: {
    avatarId: string;
    iconFile: File;
  }): Promise<string> {
    const mimeType = this.ensureSupportedImage(iconFile);
    const objectPath = this.avatarIconStoragePath(avatarId);
    const storageRef = ref(storage, objectPath);

    await uploadBytes(storageRef, iconFile, {
      contentType: mimeType,
      customMetadata: {
        avatarId,
        fileName: iconFile.name || "icon",
      },
    });

    return getDownloadURL(storageRef);
  }

  async fetchMine(): Promise<MyAvatarResponse | null> {
    const idToken = await this.getIdToken();

    return getMyAvatar({
      backendUrl: this.backendUrl,
      idToken,
    });
  }

  async save({
    avatarNameRaw,
    profileRaw,
    externalLinkRaw,
    iconFile,
  }: SaveAvatarParams): Promise<AvatarCreateResult> {
    try {
      const user = this.auth.currentUser;

      if (!user) {
        return {
          ok: false,
          message: "サインインが必要です。",
        };
      }

      const userId = this.s(user.uid);

      if (!userId) {
        return {
          ok: false,
          message: "userId が取得できませんでした。",
        };
      }

      const avatarName = this.s(avatarNameRaw);

      if (!avatarName) {
        return {
          ok: false,
          message: "アバター名を入力してください。",
        };
      }

      const externalLink = this.s(externalLinkRaw);

      if (!this.isValidUrlOrEmpty(externalLink)) {
        return {
          ok: false,
          message: "外部リンクは http(s) のURLを入力してください。",
        };
      }

      const profile = this.s(profileRaw);
      const idToken = await this.getIdToken();

      const created = await createAvatar({
        backendUrl: this.backendUrl,
        idToken,
        payload: {
          userId,
          userUid: userId,
          avatarName,
          ...(profile ? { profile } : {}),
          ...(externalLink ? { externalLink } : {}),
        },
      });

      const avatarId = this.s(created.avatarId);

      if (!avatarId) {
        return {
          ok: false,
          message: "avatarId が取得できませんでした。",
        };
      }

      if (iconFile) {
        const avatarIcon = await this.uploadAvatarIconToFirebaseStorage({
          avatarId,
          iconFile,
        });

        await updateAvatar({
          backendUrl: this.backendUrl,
          idToken,
          avatarId,
          payload: {
            avatarName,
            ...(profile ? { profile } : {}),
            ...(externalLink ? { externalLink } : {}),
            avatarIcon,
          },
        });
      }

      return {
        ok: true,
        message: "アバターを作成しました。",
        nextRoute: "/lists",
        createdAvatarId: avatarId,
      };
    } catch (error) {
      return {
        ok: false,
        message: error instanceof Error ? error.message : String(error),
      };
    }
  }

  async update({
    avatarId,
    avatarNameRaw,
    profileRaw,
    externalLinkRaw,
    iconFile,
  }: UpdateAvatarParams): Promise<AvatarUpdateResult> {
    try {
      const id = this.s(avatarId);

      if (!id) {
        return {
          ok: false,
          message: "avatarId が取得できませんでした。",
        };
      }

      const avatarName = this.s(avatarNameRaw);

      if (!avatarName) {
        return {
          ok: false,
          message: "アバター名を入力してください。",
        };
      }

      const externalLink = this.s(externalLinkRaw);

      if (!this.isValidUrlOrEmpty(externalLink)) {
        return {
          ok: false,
          message: "外部リンクは http(s) のURLを入力してください。",
        };
      }

      const profile = this.s(profileRaw);
      const idToken = await this.getIdToken();

      let avatarIcon: string | undefined;

      if (iconFile) {
        avatarIcon = await this.uploadAvatarIconToFirebaseStorage({
          avatarId: id,
          iconFile,
        });
      }

      await updateAvatar({
        backendUrl: this.backendUrl,
        idToken,
        avatarId: id,
        payload: {
          avatarName,
          ...(profile ? { profile } : {}),
          ...(externalLink ? { externalLink } : {}),
          ...(avatarIcon ? { avatarIcon } : {}),
        },
      });

      return {
        ok: true,
        message: "アバターを保存しました。",
        avatarId: id,
      };
    } catch (error) {
      return {
        ok: false,
        message: error instanceof Error ? error.message : String(error),
      };
    }
  }
}