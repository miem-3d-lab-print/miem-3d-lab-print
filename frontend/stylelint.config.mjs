export default {
  extends: ['stylelint-config-standard-scss'],
  plugins: ['stylelint-order'],
  ignoreFiles: ['**/node_modules/**', '**/dist/**', '**/coverage/**', '**/*.min.css'],
  rules: {
    'custom-property-pattern': null,
    'selector-class-pattern': null,
    'order/properties-alphabetical-order': [true, { severity: 'warning' }],
  },
};
