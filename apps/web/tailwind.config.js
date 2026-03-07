/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    './app/**/*.{js,ts,jsx,tsx,mdx}',
    './components/**/*.{js,ts,jsx,tsx,mdx}',
    './lib/**/*.{js,ts,jsx,tsx,mdx}',
  ],
  theme: {
    extend: {
      colors: {
        app: {
          bg: 'rgb(var(--bg) / <alpha-value>)',
          panel: 'rgb(var(--panel) / <alpha-value>)',
          border: 'rgb(var(--border) / <alpha-value>)',
          text: 'rgb(var(--text) / <alpha-value>)',
          muted: 'rgb(var(--text-muted) / <alpha-value>)',
          muted2: 'rgb(var(--ui-muted-2) / <alpha-value>)',
          accent: 'rgb(var(--accent) / <alpha-value>)',
          field: 'rgb(var(--ui-field-bg) / <alpha-value>)',
          fieldBorder: 'rgb(var(--ui-field-border) / <alpha-value>)',
          fieldBorderHover: 'rgb(var(--ui-field-border-hover) / <alpha-value>)',
          ring: 'rgb(var(--ui-field-ring) / <alpha-value>)',
        },
      },
      boxShadow: {
        panel: 'var(--ui-shadow-panel)',
        lift: '0 14px 30px rgb(0 0 0 / 0.28), 0 6px 14px rgb(5 10 20 / 0.18)',
      },
      animation: {
        rise: 'rise 320ms ease-out both',
        riseSoft: 'riseSoft 240ms cubic-bezier(0.16, 1, 0.3, 1) both',
      },
    },
  },
  plugins: [],
};
