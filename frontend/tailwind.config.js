/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        primary: {
          50: 'color-mix(in oklch, var(--primary) 4%, white)',
          100: 'color-mix(in oklch, var(--primary) 8%, white)',
          200: 'color-mix(in oklch, var(--primary) 14%, white)',
          300: 'color-mix(in oklch, var(--primary) 24%, white)',
          400: 'color-mix(in oklch, var(--primary) 42%, white)',
          500: 'var(--primary)',
          600: 'color-mix(in oklch, var(--primary) 90%, black)',
          700: 'color-mix(in oklch, var(--primary) 78%, black)',
          800: 'color-mix(in oklch, var(--primary) 64%, black)',
          900: 'color-mix(in oklch, var(--primary) 50%, black)',
          950: 'color-mix(in oklch, var(--primary) 38%, black)'
        },
        accent: {
          50: 'color-mix(in oklch, var(--accent) 18%, white)',
          100: 'color-mix(in oklch, var(--accent) 32%, white)',
          200: 'color-mix(in oklch, var(--accent) 48%, white)',
          300: 'color-mix(in oklch, var(--accent) 64%, white)',
          400: 'color-mix(in oklch, var(--accent) 82%, white)',
          500: 'var(--accent)',
          600: 'color-mix(in oklch, var(--accent) 88%, black)',
          700: 'color-mix(in oklch, var(--accent) 76%, black)',
          800: 'color-mix(in oklch, var(--accent) 64%, black)',
          900: 'color-mix(in oklch, var(--accent) 52%, black)',
          950: 'color-mix(in oklch, var(--accent) 40%, black)'
        },
        dark: {
          50: 'color-mix(in oklch, var(--foreground) 92%, white)',
          100: 'color-mix(in oklch, var(--foreground) 78%, white)',
          200: 'color-mix(in oklch, var(--foreground) 62%, white)',
          300: 'color-mix(in oklch, var(--foreground) 48%, white)',
          400: 'color-mix(in oklch, var(--muted-foreground) 78%, transparent)',
          500: 'var(--muted-foreground)',
          600: 'color-mix(in oklch, var(--muted) 82%, var(--foreground))',
          700: 'var(--muted)',
          800: 'var(--card)',
          900: 'var(--background)',
          950: 'color-mix(in oklch, var(--background) 82%, black)'
        }
      },
      fontFamily: {
        sans: [
          'system-ui',
          '-apple-system',
          'BlinkMacSystemFont',
          'Segoe UI',
          'Roboto',
          'Helvetica Neue',
          'Arial',
          'PingFang SC',
          'Hiragino Sans GB',
          'Microsoft YaHei',
          'sans-serif'
        ],
        mono: ['ui-monospace', 'SFMono-Regular', 'Menlo', 'Monaco', 'Consolas', 'monospace']
      },
      boxShadow: {
        glass: 'var(--surface-shadow)',
        'glass-sm': '0 1px 2px oklch(0 0 0 / 4%), 0 8px 20px oklch(0 0 0 / 4%)',
        glow: '0 0 0 1px color-mix(in oklch, var(--border) 80%, transparent)',
        'glow-lg': '0 0 0 1px color-mix(in oklch, var(--border) 80%, transparent)',
        card: 'var(--surface-shadow)',
        'card-hover': 'var(--surface-shadow-hover)',
        'inner-glow': 'inset 0 1px 0 rgba(255, 255, 255, 0.1)'
      },
      backgroundImage: {
        'gradient-radial': 'radial-gradient(var(--tw-gradient-stops))',
        'gradient-primary': 'linear-gradient(135deg, var(--primary) 0%, color-mix(in oklch, var(--primary) 72%, var(--muted)) 100%)',
        'gradient-dark': 'linear-gradient(135deg, var(--card) 0%, var(--background) 100%)',
        'gradient-glass':
          'var(--surface-glass)',
        'mesh-gradient':
          'var(--app-shell-gradient)'
      },
      animation: {
        'fade-in': 'fadeIn 0.3s ease-out',
        'slide-up': 'slideUp 0.3s ease-out',
        'slide-down': 'slideDown 0.3s ease-out',
        'slide-in-right': 'slideInRight 0.3s ease-out',
        'scale-in': 'scaleIn 0.2s ease-out',
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        shimmer: 'shimmer 2s linear infinite',
        glow: 'glow 2s ease-in-out infinite alternate'
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' }
        },
        slideUp: {
          '0%': { opacity: '0', transform: 'translateY(10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideDown: {
          '0%': { opacity: '0', transform: 'translateY(-10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideInRight: {
          '0%': { opacity: '0', transform: 'translateX(20px)' },
          '100%': { opacity: '1', transform: 'translateX(0)' }
        },
        scaleIn: {
          '0%': { opacity: '0', transform: 'scale(0.95)' },
          '100%': { opacity: '1', transform: 'scale(1)' }
        },
        shimmer: {
          '0%': { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' }
        },
        glow: {
          '0%': { boxShadow: '0 0 20px rgba(0, 0, 0, 0.08)' },
          '100%': { boxShadow: '0 0 30px rgba(0, 0, 0, 0.12)' }
        }
      },
      backdropBlur: {
        xs: '2px'
      },
      borderRadius: {
        '4xl': '2rem'
      }
    }
  },
  plugins: []
}
