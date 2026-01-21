/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./internal/**/*.templ",
    "./internal/**/*.go",
  ],
  theme: {
    extend: {
      colors: {
        primary: 'var(--color-bg-primary)',
        text: {
          primary: 'var(--color-text-primary)',
          secondary: 'var(--color-text-secondary)',
        },
        border: 'var(--color-border)',
        accent: 'var(--color-accent)',
        marigold: 'var(--color-marigold)',
        'marigold-dark': 'var(--color-marigold-dark)',
        'pink-hot': 'var(--color-pink-hot)',
        'purple-deep': 'var(--color-purple-deep)',
      },
      fontFamily: {
        sans: ['Inter', 'sans-serif'],
        serif: ['Merriweather', 'serif'],
      },
    },
  },
  plugins: [],
}
