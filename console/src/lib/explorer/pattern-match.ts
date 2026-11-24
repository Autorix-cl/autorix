const MAX_PATTERN_LENGTH = 256;
// Reject nested quantifiers and overlapping repetition groups that can trigger catastrophic backtracking
const UNSAFE_REGEX_RE = /(\([^)]*[+*?][^)]*\)[+*{]|([+*?])\s*([+*?]|\{))/;

export function safePatternMatch(pattern: string, input: string): boolean {
  if (!pattern || pattern.length > MAX_PATTERN_LENGTH) return false;
  if (UNSAFE_REGEX_RE.test(pattern)) return false;
  try {
    const regex = new RegExp(pattern);
    return regex.test(input);
  } catch {
    return false;
  }
}
