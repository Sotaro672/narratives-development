//services\solana-bubblegum\src\index.ts
import {
  app,
} from "./app.js";

function resolvePort(): number {
  const raw =
    process.env.PORT ?? "8080";

  const port =
    Number(raw);

  if (
    !Number.isInteger(port) ||
    port <= 0 ||
    port > 65535
  ) {
    throw new Error(
      `index: invalid PORT value=${raw}`,
    );
  }

  return port;
}

const port =
  resolvePort();

const server =
  app.listen(
    port,
    "0.0.0.0",
    () => {
      console.log(
        `[solana-bubblegum] listening on 0.0.0.0:${port}`,
      );
    },
  );

function shutdown(
  signal: string,
): void {
  console.log(
    `[solana-bubblegum] received ${signal}`,
  );

  server.close(
    (error) => {
      if (error) {
        console.error(
          "[solana-bubblegum] shutdown failed",
          error,
        );

        process.exitCode = 1;

        return;
      }

      console.log(
        "[solana-bubblegum] shutdown complete",
      );

      process.exitCode = 0;
    },
  );
}

process.once(
  "SIGTERM",
  () => shutdown("SIGTERM"),
);

process.once(
  "SIGINT",
  () => shutdown("SIGINT"),
);