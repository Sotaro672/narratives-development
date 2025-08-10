/** @type {import('next').NextConfig} */
const nextConfig = {
  transpilePackages: ['@narratives/domain', '@narratives/graphql-fragments'],
  env: {
    NEXT_PUBLIC_SNS_GRAPHQL_ENDPOINT: process.env.NEXT_PUBLIC_SNS_GRAPHQL_ENDPOINT || 'http://localhost:8080/graphql',
    NEXT_PUBLIC_CATALOG_GRAPHQL_ENDPOINT: process.env.NEXT_PUBLIC_CATALOG_GRAPHQL_ENDPOINT || 'http://localhost:8082/graphql',
    NEXT_PUBLIC_TOKEN_REGISTRY_ENDPOINT: process.env.NEXT_PUBLIC_TOKEN_REGISTRY_ENDPOINT || 'http://localhost:8083/graphql',
    NEXT_PUBLIC_FIREBASE_API_KEY: process.env.NEXT_PUBLIC_FIREBASE_API_KEY,
    NEXT_PUBLIC_FIREBASE_AUTH_DOMAIN: process.env.NEXT_PUBLIC_FIREBASE_AUTH_DOMAIN,
    NEXT_PUBLIC_FIREBASE_PROJECT_ID: process.env.NEXT_PUBLIC_FIREBASE_PROJECT_ID,
  },
  async rewrites() {
    return [
      {
        source: '/api/sns/:path*',
        destination: `${process.env.NEXT_PUBLIC_SNS_GRAPHQL_ENDPOINT}/:path*`,
      },
      {
        source: '/api/catalog/:path*',
        destination: `${process.env.NEXT_PUBLIC_CATALOG_GRAPHQL_ENDPOINT}/:path*`,
      },
      {
        source: '/api/token-registry/:path*',
        destination: `${process.env.NEXT_PUBLIC_TOKEN_REGISTRY_ENDPOINT}/:path*`,
      },
    ];
  }
}

module.exports = nextConfig
