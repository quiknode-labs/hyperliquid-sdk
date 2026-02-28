import { defineConfig } from 'tsup';
import { cpSync, mkdirSync } from 'fs';
import { resolve } from 'path';

export default defineConfig({
  entry: ['src/index.ts'],
  format: ['cjs', 'esm'],
  dts: true,
  splitting: false,
  sourcemap: true,
  clean: true,
  outDir: 'dist',
  target: 'es2022',
  minify: false,
  treeshake: true,
  external: ['ws', '@grpc/grpc-js', '@grpc/proto-loader'],
  esbuildOptions(options) {
    options.banner = {
      js: '/* Hyperliquid SDK - https://hyperliquidapi.com */',
    };
  },
  onSuccess: async () => {
    // Copy proto files to dist
    const srcProto = resolve(__dirname, 'src/proto');
    const distProto = resolve(__dirname, 'dist/proto');
    try {
      mkdirSync(distProto, { recursive: true });
      cpSync(srcProto, distProto, { recursive: true });
      console.log('Copied proto files to dist/proto');
    } catch (e) {
      console.log('Proto files copy skipped:', e);
    }
  },
});
