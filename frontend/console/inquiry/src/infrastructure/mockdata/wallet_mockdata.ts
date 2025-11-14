// frontend/inquiry/src/infrastructure/mockdata/wallet_mockdata.ts
// ------------------------------------------------------
// Mock data for Wallet
// Mirrors frontend/shell/src/shared/types/wallet.ts
// ------------------------------------------------------

import type { Wallet } from "../../../../shell/src/shared/types/wallet";

/**
 * Mock Wallet data representing Solana-like wallets and owned token mints.
 */
export const WALLETS: Wallet[] = [
  {
    walletAddress: "7kPsu1D3mEFkZcYVw9qEq5S2XzXvHk5NfZyApHh3oE6T",
    tokens: [
      "3vK8H8wUn2CcpZVebnBv6KDYkDkT9mZQnS3kSZPpETrH",
      "F5cKtkLr6sZ1xMzL3sNnN4oJXkD5m2C6BqTzV5Wk1gPe",
    ],
    lastUpdatedAt: "2025-11-09T13:40:00Z",
    status: "active",
    createdAt: "2025-01-01T00:00:00Z",
    updatedAt: "2025-11-09T13:40:00Z",
  },
  {
    walletAddress: "4hZrLk2eEr5a8bFq1nB2Rk6sPw7Tt3Yv9cWzXoQpJgFh",
    tokens: ["5xE7H9fK3aN2rVb1M5sYkZ3tQp8JdXg4LhWmC7NfRzT"],
    lastUpdatedAt: "2025-11-08T21:30:00Z",
    status: "active",
    createdAt: "2025-02-15T00:00:00Z",
    updatedAt: "2025-11-08T21:30:00Z",
  },
  {
    walletAddress: "8nTyWf6mJzPqRrT1vYkXe2ZcNqLd3sHfGp4WqKx9LmR",
    tokens: [],
    lastUpdatedAt: "2025-10-10T12:00:00Z",
    status: "inactive",
    createdAt: "2024-12-10T00:00:00Z",
    updatedAt: "2025-10-10T12:00:00Z",
  },
  {
    walletAddress: "9pJxTm5QwLrXnKs3yRbZc8WvHd2gFjVbXkRpQzTmYhE",
    tokens: [
      "2aN5fC8wPkL3dXnTqH6zRjV4bWmS7pLyGxKdZ9hYqEoP",
      "9rQzWt3sLpMvYxJnKhTqGwFeDcSaRfUoPiLoMkNjBvC",
    ],
    lastUpdatedAt: "2025-11-07T18:45:00Z",
    status: "active",
    createdAt: "2025-03-20T00:00:00Z",
    updatedAt: "2025-11-07T18:45:00Z",
  },
  {
    walletAddress: "3fGhRt2yWxNzJbKvPqLdCmSaErTyUiOpLkJhVgCfDxZ",
    tokens: [],
    lastUpdatedAt: "2025-09-12T10:10:00Z",
    status: "inactive",
    createdAt: "2024-11-01T00:00:00Z",
    updatedAt: "2025-09-12T10:10:00Z",
  },
];

/**
 * Default export for convenience.
 */
export default WALLETS;
