module.exports = {
  content: [
    "./internal/interfaces/http/templates/**/*.html",
  ],
  theme: {
    extend: {
      fontFamily: {
        sans: ['Outfit', 'sans-serif'],
      },
      colors: {
        club: {
          red: '#ffffff',
          dark: '#0a0a0a',
          panel: '#111111',
          gold: '#e0e0e0'
        }
      }
    },
  },
  plugins: [],
}
