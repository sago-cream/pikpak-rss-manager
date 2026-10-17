import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';

const backend = process.env.VITE_BACKEND_URL ?? 'http://127.0.0.1:8080';
const proxy = {
    target: backend,
    changeOrigin: true,
    configure(server) {
        server.on('proxyReq', (request) => {
            request.setHeader('Origin', new URL(backend).origin);
        });
    },
};
export default defineConfig({
    base: '/ui/',
    plugins: [react()],
    server: {
        host: '127.0.0.1',
        port: 5173,
        strictPort: true,
        proxy: { '/api': proxy, '/healthz': proxy },
    },
    build: { outDir: '../internal/web/ui/dist', emptyOutDir: true },
});
