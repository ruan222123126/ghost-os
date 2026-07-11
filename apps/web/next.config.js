const withBundleAnalyzer = require('@next/bundle-analyzer')({
  enabled: process.env.ANALYZE === 'true',
  openAnalyzer: false,
});

const disabledMarkstreamOptionalPeers = [
  '@antv/infographic',
  '@terrastruct/d2',
  'mermaid',
  'stream-markdown',
  'stream-monaco',
];

/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  distDir: process.env.NODE_ENV === 'development' ? '.next-dev' : '.next',
  webpack(config) {
    config.resolve.alias = {
      ...(config.resolve.alias || {}),
      ...Object.fromEntries(disabledMarkstreamOptionalPeers.map((name) => [name, false])),
    };
    return config;
  },
};

module.exports = withBundleAnalyzer(nextConfig);
