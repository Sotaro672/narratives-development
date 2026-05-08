// frontend/src/features/contact/components/ContactForm.tsx
import { ChangeEvent, RefObject } from "react";

import Input from "../../../components/ui/Input";
import Textbox from "../../../components/ui/Textbox";
import MediaUploader from "../../../components/ui/MediaUploader";
import type { ContactAttachmentItem } from "../types";

type ContactFormProps = {
  shouldShowGuestEmailInput: boolean;
  name: string;
  guestEmail: string;
  company: string;
  message: string;
  submitting: boolean;
  attachments: ContactAttachmentItem[];
  carouselIndex: number;
  mediaInputRef: RefObject<HTMLInputElement>;
  carouselRef: RefObject<HTMLDivElement>;
  onNameChange: (value: string) => void;
  onGuestEmailChange: (value: string) => void;
  onCompanyChange: (value: string) => void;
  onMessageChange: (value: string) => void;
  onFilesSelected: (event: ChangeEvent<HTMLInputElement>) => void;
  onRemoveAttachment: (id: string) => void;
  onCarouselScroll: () => void;
  onMoveToSlide: (index: number) => void;
};

export default function ContactForm({
  shouldShowGuestEmailInput,
  name,
  guestEmail,
  company,
  message,
  submitting,
  attachments,
  carouselIndex,
  mediaInputRef,
  carouselRef,
  onNameChange,
  onGuestEmailChange,
  onCompanyChange,
  onMessageChange,
  onFilesSelected,
  onRemoveAttachment,
  onCarouselScroll,
  onMoveToSlide,
}: ContactFormProps) {
  return (
    <div className="landing-page-form contact-form">
      <div className="landing-page-form__group">
        <Input
          id="contact-name"
          label="お名前"
          placeholder="お名前を入力してください"
          value={name}
          onChange={(event) => onNameChange(event.target.value)}
          required
          disabled={submitting}
        />
      </div>

      {shouldShowGuestEmailInput ? (
        <div className="landing-page-form__group">
          <Input
            id="contact-email"
            type="email"
            label="メールアドレス"
            placeholder="example@example.com"
            value={guestEmail}
            onChange={(event) => onGuestEmailChange(event.target.value)}
            required
            disabled={submitting}
          />
        </div>
      ) : null}

      <div className="landing-page-form__group">
        <Input
          id="contact-company"
          label="会社名"
          placeholder="会社名を入力してください"
          value={company}
          onChange={(event) => onCompanyChange(event.target.value)}
          disabled={submitting}
        />
      </div>

      <div className="landing-page-form__group">
        <Textbox
          id="contact-message"
          label="お問い合わせ内容"
          placeholder="お問い合わせ内容を入力してください"
          rows={6}
          value={message}
          onChange={(event) => onMessageChange(event.target.value)}
          required
          disabled={submitting}
        />
      </div>

      <div className="landing-page-form__group">
        <MediaUploader
          label="添付ファイル画像"
          hint="アップロードできるのは画像のみです。"
          emptyText="添付ファイルはまだ選択されていません。"
          accept="image/*"
          multiple
          items={attachments}
          currentIndex={carouselIndex}
          inputRef={mediaInputRef}
          carouselRef={carouselRef}
          onFilesSelected={onFilesSelected}
          onRemoveItem={onRemoveAttachment}
          onCarouselScroll={onCarouselScroll}
          onMoveToSlide={onMoveToSlide}
          selectButtonLabel="ファイルを選択"
          disabled={submitting}
        />
      </div>

      <div className="landing-page-form__group">
        <p className="landing-page-definition-list__term">ご案内</p>
        <p className="landing-page-card__text">
          お問い合わせ内容によっては、ご回答までにお時間をいただく場合があります。
        </p>
      </div>
    </div>
  );
}