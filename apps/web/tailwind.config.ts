/** @type {import('tailwindcss').Config} */
export default {
  content: {
    relative: true,
    files: ['./index.html', './src/**/*.{ts,tsx}', './.storybook/**/*.{ts,tsx}'],
  },
  theme: {
    extend: {
      colors: {
        accent: 'rgb(var(--action) / <alpha-value>)',
        surface: 'var(--surface)',
        'surface-muted': 'var(--surface-muted)',
        canvas: 'var(--canvas)',
        ink: 'var(--ink)',
        muted: 'var(--muted)',
        line: 'var(--line)',
        navigation: 'var(--navigation)',
        'navigation-text': 'var(--navigation-text)',
        'navigation-muted': 'var(--navigation-muted)',
        'severity-p1': '#DC2626',
        'severity-p2': '#C2410C',
        'severity-p3': '#92400E',
        'severity-p4': '#2563EB',
        resolved: '#15803D',
      },
      fontFamily: {
        sans: ['"IBM Plex Sans"', 'system-ui', 'sans-serif'],
        mono: ['"IBM Plex Mono"', 'monospace'],
      },
    },
  },
  plugins: [],
};
