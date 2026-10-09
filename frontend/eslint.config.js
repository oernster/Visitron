// What the front end is checked for; why each rule is here.
//
// The Go side has gofmt, vet and staticcheck standing over it, plus a suite that
// refuses to pass below its own bar. The front end had the type checker and nothing
// else, across a third of the source: enough to know that a name exists, never enough
// to know that a hook was told the truth about what it depends on.

import js from '@eslint/js'
import tseslint from 'typescript-eslint'
import reactHooks from 'eslint-plugin-react-hooks'

export default tseslint.config(
  {
    ignores: ['dist/**', 'wailsjs/**', 'node_modules/**'],
  },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    files: ['**/*.{ts,tsx}'],
    plugins: { 'react-hooks': reactHooks },
    rules: {
      // An effect that reads a value it never declared runs once against whatever
      // that value happened to be, then never again. Checked rather than assumed: a
      // planted effect reading a prop outside its dependency list is refused by name.
      //
      // Worth being exact about its reach, since the obvious story is the wrong one.
      // Three scrolling regions in this application did stop being reachable from the
      // keyboard for a mistake of this shape. The value they read was a ref though,
      // and this rule leaves refs alone by design, so it would not have caught that
      // one. It catches the rest of the family, which is why it is here rather than
      // because it would have saved that particular afternoon.
      'react-hooks/exhaustive-deps': 'error',
      'react-hooks/rules-of-hooks': 'error',

      // An unused name is either a leftover or a mistake about what a function was
      // given. Arguments deliberately ignored say so with a leading underscore.
      '@typescript-eslint/no-unused-vars': [
        'error',
        { argsIgnorePattern: '^_', varsIgnorePattern: '^_' },
      ],

      // any is how a typed codebase stops being one, quietly and in one file at a
      // time. There is none today.
      '@typescript-eslint/no-explicit-any': 'error',

      // == against null is the one coercion worth keeping, because it catches
      // undefined as well and every line here that wants it means both.
      eqeqeq: ['error', 'always', { null: 'ignore' }],
    },
  },
)
