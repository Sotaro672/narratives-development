// services/solana-bubblegum/src/app.ts

import express, {
  type NextFunction,
  type Request,
  type Response,
} from "express";

export const app = express();

app.disable("x-powered-by");

app.use(
  express.json({
    limit: "32kb",
  }),
);

app.get(
  "/healthz",
  (_req: Request, res: Response) => {
    res.status(200).json({
      status: "ok",
      service: "solana-bubblegum",
    });
  },
);

app.use(
  (
    _req: Request,
    res: Response,
  ) => {
    res.status(404).json({
      error: "not found",
    });
  },
);

app.use(
  (
    error: unknown,
    _req: Request,
    res: Response,
    _next: NextFunction,
  ) => {
    console.error(
      "[http]",
      error,
    );

    if (
      error instanceof SyntaxError
    ) {
      res.status(400).json({
        error: "invalid JSON body",
      });

      return;
    }

    res.status(500).json({
      error: "internal server error",
    });
  },
);