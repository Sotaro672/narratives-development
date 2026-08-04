// frontend/amol/src/lib/http/httpError.ts

export type HttpErrorParams = {
  message: string;
  status: number;
  url: string;
  body?: unknown;
};

export class HttpError extends Error {
  readonly status: number;
  readonly url: string;
  readonly body: unknown;

  constructor({
    message,
    status,
    url,
    body,
  }: HttpErrorParams) {
    super(message);

    this.name = "HttpError";
    this.status = status;
    this.url = url;
    this.body = body;
  }
}