import tailwindcss from '@tailwindcss/vite';
import react from '@vitejs/plugin-react';
import path from 'path';
import {defineConfig, loadEnv} from 'vite';

export default defineConfig(({mode}) => {
  const env = loadEnv(mode, '.', '');
  return {
    plugins: [react(), tailwindcss()],
    define: {
      'process.env.GEMINI_API_KEY': JSON.stringify(env.GEMINI_API_KEY),
    },
    resolve: {
      alias: {
        '@': path.resolve(__dirname, '.'),
      },
    },
    server: {
      hmr: process.env.DISABLE_HMR !== 'true',
      proxy: {
        '/auth/': 'http://localhost:8090',
        '/account/': 'http://localhost:8090',
        '/stripe/': 'http://localhost:8090',
        '/v1/': 'http://localhost:8090',
        '/crypto/': 'http://localhost:8090',
        '/health': 'http://localhost:8090',
        '/stats/': 'http://localhost:8090',
      },
    },
  };
});
