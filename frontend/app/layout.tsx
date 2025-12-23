import type { Metadata } from 'next'
import './globals.css'

export const metadata: Metadata = {
  title: 'Link Analytics Service',
  description: 'URL shortening service with real-time analytics',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  )
}

