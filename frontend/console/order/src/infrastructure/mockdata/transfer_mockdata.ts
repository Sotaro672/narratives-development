// frontend/order/src/infrastructure/mockdata/transfer_mockdata.ts

import type {
  Transfer,
  TransferStatus,
  TransferErrorType,
} from "../../../../shell/src/shared/types/transfer";

/**
 * モック用 Transfer データ
 * frontend/shell/src/shared/types/transfer.ts に準拠。
 *
 * - Solana アドレス風 Base58 形式
 * - Status は requested / fulfilled / error のいずれか
 */
export const TRANSFERS: Transfer[] = [
  {
    id: "transfer_001",
    mintAddress: "9vQXzX2hMZp8yA5y2V4g3uK5BdJw6YtNs8C7kJ4r2aBc",
    fromAddress: "C3RrV9yRKwH3Uo9b8YtL5kQ2aNdP6xW4gT7mE8sN9hFd",
    toAddress: "H8sJ3nW4qR7yU2mV9xB5zL6aN3kT8eR1pQ4wD5uF2oGh",
    requestedAt: "2024-03-21T09:30:00Z",
    transferredAt: "2024-03-21T09:35:00Z",
    status: "fulfilled",
    errorType: null,
  },
  {
    id: "transfer_002",
    mintAddress: "G6nQxA7sL3mE9zR1vT5dP2fC8hY4bU9tK7jN3wV5xSgH",
    fromAddress: "A4pT8yQ9nC7uV3dL5fE1rK6sZ2gW9xB8tN4jY5hM7oJq",
    toAddress: "L9vB4fN6xD2kY7wT3mC8rS1uQ5zG9aJ2hE6nP4tU8oHq",
    requestedAt: "2024-03-22T10:00:00Z",
    transferredAt: null,
    status: "requested",
    errorType: null,
  },
  {
    id: "transfer_003",
    mintAddress: "E2bN9tM4wC6yH1xV7jP3rQ8dF5zK9aT4sU2gL6eW8oYq",
    fromAddress: "Q3rV8yU9nC7sM5dL2fT1aK6xZ4wB9gN8tP4jY5hR7oJe",
    toAddress: "B6fN3xL9dY7wR1kT8mC2rS5uQ4zG9aJ2hE6nP4tV8oHq",
    requestedAt: "2024-03-22T11:15:00Z",
    transferredAt: null,
    status: "error",
    errorType: "network_error",
  },
  {
    id: "transfer_004",
    mintAddress: "J1rP8aM7wV6yC9xN5tF3dK4sQ2gB8uT9jE1nH6oL4eYq",
    fromAddress: "W5nQ8yT9uV7cM3dL1fR6aK4sZ2gB9xN8tJ4pY5hE7oJr",
    toAddress: "K2vB9fN6xL3kY7wT1mC8rS5uQ4zG9aJ2hE6nP4tU8oHp",
    requestedAt: "2024-03-23T08:45:00Z",
    transferredAt: "2024-03-23T08:50:00Z",
    status: "fulfilled",
    errorType: null,
  },
  {
    id: "transfer_005",
    mintAddress: "F9kT7aN5wB3yL1xV6jP2rQ8dC4zK9aT3sU2gL6eW8oYq",
    fromAddress: "A8pT9yQ7nC5uV3dL1fE6rK4sZ2gW9xB8tN4jY5hM7oJq",
    toAddress: "M4vB8fN6xD2kY7wT9mC3rS5uQ1zG9aJ2hE6nP4tU8oHq",
    requestedAt: "2024-03-23T09:00:00Z",
    transferredAt: null,
    status: "error",
    errorType: "timeout",
  },
];

/**
 * UI 用補助: TransferStatus 表示ラベル
 */
export function getTransferStatusLabel(status: TransferStatus): string {
  switch (status) {
    case "fulfilled":
      return "転送完了";
    case "requested":
      return "転送要求中";
    case "error":
      return "エラー";
    default:
      return "不明";
  }
}

/**
 * UI 用補助: TransferErrorType 表示ラベル
 */
export function getTransferErrorLabel(
  errorType?: TransferErrorType | null
): string {
  if (!errorType) return "—";
  switch (errorType) {
    case "insufficient_balance":
      return "残高不足";
    case "invalid_address":
      return "宛先不正";
    case "network_error":
      return "ネットワークエラー";
    case "timeout":
      return "タイムアウト";
    case "unknown":
      return "不明エラー";
    default:
      return "—";
  }
}
