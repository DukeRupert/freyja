/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./web/templates/**/*.html",
    "./web/static/js/**/*.js",
  ],
  theme: {
    extend: {
      colors: {
        teal: {
          50: '#F0F7F7',
          100: '#D6EBEB',
          600: '#2F8C8C',
          700: '#2A7D7D',
          800: '#1F5F5F',
          900: '#164545',
        },
        amber: {
          50: '#FAF8F3',
          100: '#F5EFE6',
          200: '#EBE0CC',
          600: '#C69345',
          700: '#B5873A',
          800: '#9A7030',
          900: '#7D5C26',
        },
      },
    },
  },
  plugins: [],
}
