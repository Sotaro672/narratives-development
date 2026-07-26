// frontend/console/scripts/generate-brand-image-policy.mjs

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const currentFile = fileURLToPath(import.meta.url);
const currentDir = path.dirname(currentFile);

// currentDir:
// frontend/console/scripts
//
// consoleRootDir:
// frontend/console
//
// frontendRootDir:
// frontend
const consoleRootDir = path.resolve(currentDir, "..");
const frontendRootDir = path.resolve(currentDir, "../..");

const policyPath = path.join(
  consoleRootDir,
  "shell/src/features/brand/config/brandImagePolicy.json",
);

const frontendOutputPath = path.join(
  consoleRootDir,
  "shell/src/features/brand/config/brandImagePolicy.generated.ts",
);

const storageRulesTemplatePath = path.join(
  frontendRootDir,
  "storage.rules.template",
);

const storageRulesOutputPath = path.join(
  frontendRootDir,
  "storage.rules",
);

function ensureFileExists(filePath, label) {
  if (!fs.existsSync(filePath)) {
    throw new Error(
      `${label} が見つかりません。\n` +
        `確認対象: ${filePath}`,
    );
  }
}

ensureFileExists(
  policyPath,
  "brandImagePolicy.json",
);

ensureFileExists(
  storageRulesTemplatePath,
  "storage.rules.template",
);

const policyText = fs.readFileSync(
  policyPath,
  "utf8",
);

let policy;

try {
  policy = JSON.parse(policyText);
} catch (error) {
  throw new Error(
    `brandImagePolicy.jsonのJSON形式が不正です。\n` +
      `${error instanceof Error ? error.message : String(error)}`,
  );
}

const allowedMimeTypes = policy.allowedMimeTypes;

const iconMaxBytes =
  policy.targets?.brandIcon?.maxBytes;

const backgroundMaxBytes =
  policy.targets?.brandBackgroundImage?.maxBytes;

if (
  !Array.isArray(allowedMimeTypes) ||
  allowedMimeTypes.length === 0 ||
  !allowedMimeTypes.every(
    (mimeType) =>
      typeof mimeType === "string" &&
      mimeType !== "",
  )
) {
  throw new Error(
    "allowedMimeTypesには1件以上のMIME文字列を指定してください。",
  );
}

if (
  !Number.isInteger(iconMaxBytes) ||
  iconMaxBytes <= 0
) {
  throw new Error(
    "targets.brandIcon.maxBytesには正の整数を指定してください。",
  );
}

if (
  !Number.isInteger(backgroundMaxBytes) ||
  backgroundMaxBytes <= 0
) {
  throw new Error(
    "targets.brandBackgroundImage.maxBytesには正の整数を指定してください。",
  );
}

const frontendSource = `// このファイルは自動生成です。直接編集しないでください。

export const BRAND_IMAGE_ALLOWED_MIME_TYPES = ${JSON.stringify(
  allowedMimeTypes,
  null,
  2,
)} as const;

export type BrandImageMimeType =
  (typeof BRAND_IMAGE_ALLOWED_MIME_TYPES)[number];

export const BRAND_IMAGE_MAX_BYTES = {
  brandIcon: ${iconMaxBytes},
  brandBackgroundImage: ${backgroundMaxBytes},
} as const;

export type BrandImageTarget =
  keyof typeof BRAND_IMAGE_MAX_BYTES;
`;

fs.mkdirSync(
  path.dirname(frontendOutputPath),
  {
    recursive: true,
  },
);

fs.writeFileSync(
  frontendOutputPath,
  frontendSource,
  "utf8",
);

const mimeRule = allowedMimeTypes
  .map(
    (mimeType) =>
      `request.resource.contentType == '${mimeType}'`,
  )
  .join("\n          || ");

const rulesTemplate = fs.readFileSync(
  storageRulesTemplatePath,
  "utf8",
);

const requiredPlaceholders = [
  "__ALLOWED_MIME_CONDITION__",
  "__BRAND_ICON_MAX_BYTES__",
  "__BRAND_BACKGROUND_MAX_BYTES__",
];

for (const placeholder of requiredPlaceholders) {
  if (!rulesTemplate.includes(placeholder)) {
    throw new Error(
      `storage.rules.templateに${placeholder}がありません。`,
    );
  }
}

const storageRules = rulesTemplate
  .replaceAll(
    "__ALLOWED_MIME_CONDITION__",
    mimeRule,
  )
  .replaceAll(
    "__BRAND_ICON_MAX_BYTES__",
    String(iconMaxBytes),
  )
  .replaceAll(
    "__BRAND_BACKGROUND_MAX_BYTES__",
    String(backgroundMaxBytes),
  );

fs.mkdirSync(
  path.dirname(storageRulesOutputPath),
  {
    recursive: true,
  },
);

fs.writeFileSync(
  storageRulesOutputPath,
  storageRules,
  "utf8",
);

console.log("Brand image policy files generated.");
console.log(`Policy: ${policyPath}`);
console.log(`Frontend: ${frontendOutputPath}`);
console.log(`Storage Rules Template: ${storageRulesTemplatePath}`);
console.log(`Storage Rules: ${storageRulesOutputPath}`);