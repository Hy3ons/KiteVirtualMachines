import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

const rootDir = path.dirname(fileURLToPath(import.meta.url))

// https://vite.dev/config/
export default defineConfig({
  envDir: path.join(rootDir, '.vite-env-disabled'),
  plugins: [react()],
  build: {
    chunkSizeWarningLimit: 1200,
    rolldownOptions: {
      output: {
        codeSplitting: {
          groups: [
            {
              name: 'terminal',
              test: /node_modules[\\/]@xterm[\\/]/,
              priority: 30,
            },
            {
              name: 'antd',
              test: /node_modules[\\/](@ant-design|antd|rc-|@rc-component)[\\/]/,
              priority: 20,
            },
            {
              name: 'react',
              test: /node_modules[\\/](react|react-dom|react-router-dom|zustand)[\\/]/,
              priority: 10,
            },
          ],
        },
      },
    },
  },
})
