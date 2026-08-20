import { FlatCompat } from "@eslint/eslintrc";

const compat = new FlatCompat({
  baseDirectory: import.meta.dirname,
});

const eslintConfig = [
  ...compat.extends("next/core-web-vitals", "next/typescript"),
  {
    // e2e/**: Playwright test files, not React source — its fixture
    // callback is named `use` (Playwright's own convention), which
    // react-hooks/rules-of-hooks otherwise misreads as React's use() hook.
    ignores: [".next/**", "node_modules/**", "coverage/**", "e2e/**", "playwright-report/**", "test-results/**"],
  },
];

export default eslintConfig;
