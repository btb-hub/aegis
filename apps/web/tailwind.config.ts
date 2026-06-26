/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}', './.storybook/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        accent: '#2563EB',
        surface: '#fafafa',
        'surface-muted': '#f4f4f5',
        'severity-p1': '#DC2626',
        'severity-p2': '#EA580C',
        'severity-p3': '#D97706',
        'severity-p4': '#9333EA',
        resolved: '#16A34A',
      },
      fontFamily: {
        sans: ['"IBM Plex Sans"', 'system-ui', 'sans-serif'],
        mono: ['"IBM Plex Mono"', 'monospace'],
      },
    },
  },
  plugins: [],
};
