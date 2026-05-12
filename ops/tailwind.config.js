/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        bg: {
          base: '#0d1117',
          surface: '#161b22',
          elevated: '#1c2128',
        },
        border: {
          DEFAULT: '#30363d',
          hover: '#484f58',
        },
        accent: {
          blue: '#58a6ff',
          green: '#238636',
          red: '#f85149',
          orange: '#d29922',
        },
        text: {
          primary: '#e6edf3',
          secondary: '#8b949e',
          muted: '#6e7681',
        }
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'Fira Code', 'monospace'],
      }
    },
  },
  plugins: [],
}
