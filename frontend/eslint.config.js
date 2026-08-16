import js from '@eslint/js';
import prettierConfig from 'eslint-config-prettier';
import reactDom from 'eslint-plugin-react-dom';
import reactHooks from 'eslint-plugin-react-hooks';
import reactRefresh from 'eslint-plugin-react-refresh';
import reactX from 'eslint-plugin-react-x';
import simpleImportSort from 'eslint-plugin-simple-import-sort';
import globals from 'globals';
import tseslint from 'typescript-eslint';

import { defineConfig, globalIgnores } from 'eslint/config';

export default defineConfig([
  globalIgnores(['dist', 'coverage']),
  {
    files: ['**/*.{ts,tsx}'],
    plugins: {
      'react-dom': reactDom,
      'react-x': reactX,
      'simple-import-sort': simpleImportSort,
    },
    extends: [
      js.configs.recommended,
      tseslint.configs.recommended,
      reactHooks.configs.flat.recommended,
      reactRefresh.configs.vite,
      prettierConfig,
    ],
    languageOptions: {
      globals: globals.browser,
    },
    rules: {
      'react-x/no-missing-key': 'error',
      'react-dom/no-dangerously-set-innerhtml': 'error',
      'simple-import-sort/imports': [
        'warn',
        {
          groups: [
            ['^\\u0000'],
            ['^node:'],
            ['^react$', '^react/'],
            ['^@?\\w'],
            ['^@/'],
            ['^\\.\\.'],
            ['^\\.'],
            ['^.+\\.(css|scss|sass|less)$'],
          ],
        },
      ],
      'simple-import-sort/exports': 'warn',
    },
  },
]);
