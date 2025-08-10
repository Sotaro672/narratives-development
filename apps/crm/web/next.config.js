/** @type {import('next').NextConfig} */
const nextConfig = {
  transpilePackages: ['@narratives/domain', '@narratives/graphql-fragments'],
  env: {
    NEXT_PUBLIC_CRM_GRAPHQL_ENDPOINT: process.env.NEXT_PUBLIC_CRM_GRAPHQL_ENDPOINT || 'http://localhost:8081/graphql',
    NEXT_PUBLIC_FIREBASE_API_KEY: process.env.NEXT_PUBLIC_FIREBASE_API_KEY,
    NEXT_PUBLIC_FIREBASE_AUTH_DOMAIN: process.env.NEXT_PUBLIC_FIREBASE_AUTH_DOMAIN,
    NEXT_PUBLIC_FIREBASE_PROJECT_ID: process.env.NEXT_PUBLIC_FIREBASE_PROJECT_ID,
  }
}

module.exports = nextConfig
