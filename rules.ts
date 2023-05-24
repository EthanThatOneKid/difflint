/**
 * A set of lint rules parsed from language-agnostic source code.
 */
export interface Ruleset {
  /** The rules of a ruleset. */
  rules: Rule[];
}

/**
 * A lint rule parsed from language-agnostic source code.
 */
export interface Rule {
  /**
   * The location of the rule in the source code.
   */
  location: RuleLocation;

  /**
   * The locations in the source code where diff is expected.
   */
  targets: RuleLocation[];
}

/**
 * Source code grouped between a start and end line.
 */
export interface RuleLocation {
  /** File specifier of the rule. */
  file: string;

  /** The range of lines in the source code where the rule is applicable. */
  range: LineRange;
}

/**
 * The **inclusive** range of lines in the source code.
 */
interface LineRange {
  start: number;
  end: number;
}
