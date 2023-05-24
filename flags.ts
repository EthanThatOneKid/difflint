import { flags } from "./deps.ts";
import type { ParseOptions } from "./parse.ts";

export function fromArgs(args: string[]): Flags {
  return flags.parse(args);
}

export const DEFAULT_FLAGS: Flags = {
  files: {
    include: ["**/*.ts", "**/*.tsx", "**/*.js", "**/*.jsx", "**/*.jsonc"],
    exclude: [],
  },
  parseOptions: {
    file: "",
    code: "",
    prefixes: [],
  },
};

/**
 * The flags that are used to configure difflint.
 */
export interface Flags {
  files: FileFlags;
  parseOptions: ParseFlags;
}

export type ParseFlags = ParseOptions;

/**
 * The key is the overwritten or custom file name excluding the dot.
 */
interface FileFlags {
  include: string[];
  exclude: string[];
}
