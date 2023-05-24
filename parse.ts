import type { Rule, RuleLocation, Ruleset } from "./rules.ts";

/**
 * Parses the given string into a list of lint rules.
 *
 * Example:
 * ```
 * // bar.ts
 * //LINT.IF
 * //LINT.THEN foo.ts:name
 * ```
 *
 * ```
 * // foo.ts
 * //LINT.IF name
 * //LINT.THEN
 * ```
 */
export function parse(options: ParseOptions): Ruleset {
  const rules: Rule[] = [];
  const location: RuleLocation = {
    file: options.file,
    range: { start: 0, end: 0 },
  };

  let rule: Rule | undefined;
  const lines = options.code.split("\n");
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const prefix = getPrefix(line, options.prefixes);
    if (prefix === undefined) {
      continue;
    }

    const extension = getExtension(
      options.file,
      options.extensions[prefix.index],
    );
    if (extension === undefined) {
      continue;
    }

    const ruleText = line.slice(prefix.prefix.length).trim();
    if (ruleText === "IF") {
      if (rule !== undefined) {
        throw new Error("Unexpected IF");
      }

      rule = {
        location: { ...location, range: { start: i + 1, end: i + 1 } },
        targets: [],
      };
    } else if (ruleText === "THEN") {
      if (rule === undefined) {
        throw new Error("Unexpected THEN");
      }

      rule.location.range.end = i;
      rules.push(rule);
      rule = undefined;
    } else {
      if (rule === undefined) {
        throw new Error("Unexpected rule text");
      }

      rule.targets.push({
        file: ruleText,
        range: { start: i + 1, end: i + 1 },
      });
    }

    location.range.end = i + 1;
  }
}

// interface ParseState {
//   if?: string | undefined;
//   then?: string | undefined;
// }

enum ParseState {
  NONE,
  IF,
}

interface ParseLineOptions {
  /** Content of the line. */
  line: string;

  /** The index of the line in the source code. */
  index: number;

  /** The prefix that is used to identify the line. */
  prefix: string;

  /** Parse state. */
  state: ParseState;

  /** ID of the rule. */
  id?: string | undefined;
}

function parseLine(options: ParseLineOptions) {
}

// function getPrefix(line: string, prefixes: string[]): Prefix | undefined {
//   for (const [index, prefix] of prefixes.entries()) {
//     if (line.startsWith(prefix)) {
//       return { index, prefix };
//     }
//   }
// }

// function pars

/**
 * The key is the index of the prefix in the prefixes array.
 * The value is the list of extensions that the prefix applies to.
 */
export interface ParseOptions {
  /** Input source code file specifier. */
  file: string;

  /** Input source code to parse. */
  code: string;

  /** Prefixes that are used to identify a lint rule. */
  prefixes: {
    /** Text content that identifies a lint rule. */
    content: string;

    /** The valid regular expression patterns. */
    patterns: RegExp[];
  }[];

  /**
   * The extensions that each prefix applies to. The key is the index of the
   * prefix in the prefixes array.
   */
  extensions: Record<number, string[]>;
}
