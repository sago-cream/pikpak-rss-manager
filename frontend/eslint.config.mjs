import { completeConfigBase } from 'eslint-config-complete';
import reactHooks from 'eslint-plugin-react-hooks';

export default [
    ...completeConfigBase,
    reactHooks.configs.flat.recommended,

    {
        ignores: ['dist/**', 'node_modules/**', 'public/**'],
    },

    {
        rules: {
            // React uses nullable refs, conditional JSX, and writable API models.
            'complete/require-ascii': 'off', // Traditional Chinese is application copy.
            'complete/prefer-template': 'off', // Translation keys contain literal ${name}.
            'complete/no-template-curly-in-string-fix': 'off',
            'complete/require-variadic-function-argument': 'off',
            'complete/strict-undefined-functions': 'off',
            'no-alert': 'off', // Explicit native confirmation for destructive/cloud actions.
            '@typescript-eslint/no-floating-promises': [
                'error',
                { ignoreVoid: true },
            ],
            'no-param-reassign': ['error', { props: false }],
            'complete/prefer-readonly-parameter-types': 'off',
            'complete/no-mutable-return': 'off',
            'unicorn/no-null': 'off',
            'no-nested-ternary': 'off',
            'no-void': ['error', { allowAsStatement: true }],
            '@typescript-eslint/strict-boolean-expressions': [
                'error',
                {
                    allowString: true,
                    allowNumber: true,
                    allowNullableBoolean: true,
                    allowNullableObject: true,
                    allowNullableString: true,
                    allowNullableNumber: true,
                },
            ],
            '@typescript-eslint/prefer-nullish-coalescing': [
                'error',
                {
                    ignorePrimitives: { string: true },
                    ignoreConditionalTests: true,
                },
            ],
            '@stylistic/quotes': [
                'error',
                'single',
                {
                    avoidEscape: true,
                },
            ],
            'import-x/no-unassigned-import': [
                'error',
                {
                    allow: ['**/*.css'],
                },
            ],
        },
    },
    {
        files: ['src/**/*.tsx', 'tests/**/*.tsx'],
        rules: { '@typescript-eslint/strict-boolean-expressions': 'off' },
    },
    {
        files: ['tests/**/*'],
        rules: {
            '@typescript-eslint/no-non-null-assertion': 'off',
            '@typescript-eslint/require-await': 'off',
        },
    },
];
