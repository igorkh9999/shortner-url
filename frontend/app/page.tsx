'use client';

import { useState } from 'react';
import { createLink } from '@/lib/api';
import { useRouter } from 'next/navigation';

export default function Home() {
    const [url, setUrl] = useState('');
    const [shortUrl, setShortUrl] = useState<string | null>(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const router = useRouter();

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setError(null);
        setLoading(true);

        try {
            // Validate URL
            new URL(url);

            const result = await createLink(url, 'demo-user');
            setShortUrl(result.short_url);
        } catch (err: any) {
            setError(err.message || 'Invalid URL or server error');
        } finally {
            setLoading(false);
        }
    };

    const copyToClipboard = () => {
        if (shortUrl) {
            navigator.clipboard.writeText(shortUrl);
            alert('Copied to clipboard!');
        }
    };

    return (
        <div className="min-h-screen bg-gray-50">
            <nav className="bg-white shadow-sm">
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                    <div className="flex justify-between h-16">
                        <div className="flex items-center">
                            <h1 className="text-xl font-bold text-gray-900 cursor-pointer" onClick={() => router.push('/')}>
                                Link Analytics
                            </h1>
                        </div>
                        <div className="flex items-center space-x-4">
                            <a
                                href="/links"
                                className="text-gray-600 hover:text-gray-900 px-3 py-2 rounded-md text-sm font-medium"
                            >
                                My Links
                            </a>
                        </div>
                    </div>
                </div>
            </nav>

            <main className="max-w-2xl mx-auto py-12 px-4 sm:px-6 lg:px-8">
                <div className="bg-white rounded-lg shadow p-8">
                    <h2 className="text-2xl font-bold text-gray-900 mb-6">Shorten Your URL</h2>

                    <form onSubmit={handleSubmit} className="space-y-4">
                        <div>
                            <label htmlFor="url" className="block text-sm font-medium text-gray-700 mb-2">
                                Enter URL to shorten
                            </label>
                            <input
                                type="text"
                                id="url"
                                value={url}
                                onChange={(e) => setUrl(e.target.value)}
                                placeholder="https://example.com/very/long/url"
                                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                                required
                            />
                        </div>

                        <button
                            type="submit"
                            disabled={loading}
                            className="w-full bg-blue-600 text-white py-2 px-4 rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                        >
                            {loading ? 'Creating...' : 'Shorten URL'}
                        </button>
                    </form>

                    {error && <div className="mt-4 p-4 bg-red-50 border border-red-200 text-red-700 rounded-lg">{error}</div>}

                    {shortUrl && (
                        <div className="mt-6 p-4 bg-green-50 border border-green-200 rounded-lg">
                            <p className="text-sm text-gray-600 mb-2">Your short URL:</p>
                            <div className="flex items-center gap-2">
                                <input
                                    type="text"
                                    value={shortUrl}
                                    readOnly
                                    className="flex-1 px-4 py-2 bg-white border border-gray-300 rounded-lg"
                                />
                                <button
                                    onClick={copyToClipboard}
                                    className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
                                >
                                    Copy
                                </button>
                                <a
                                    href={`/analytics/${shortUrl.split('/').pop()}`}
                                    className="px-4 py-2 bg-gray-600 text-white rounded-lg hover:bg-gray-700 transition-colors"
                                >
                                    View Analytics
                                </a>
                                <a
                                    href={shortUrl}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    className="px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors"
                                >
                                    Test Link
                                </a>
                            </div>
                        </div>
                    )}
                </div>
            </main>
        </div>
    );
}
