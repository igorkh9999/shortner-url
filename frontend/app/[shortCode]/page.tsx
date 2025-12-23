'use client';

import { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import { getLink, trackClick } from '@/lib/api';

export default function RedirectPage() {
    const params = useParams();
    const shortCode = params.shortCode as string;
    const [error, setError] = useState<string | null>(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        const redirect = async () => {
            try {
                setLoading(true);
                const link = await getLink(shortCode);

                // Redirect to original URL
                if (link.original_url) {
                    // Track the click (fire and forget - don't wait for response)
                    trackClick(shortCode).catch((err) => {
                        // Ignore errors - analytics tracking is best effort
                        console.error('Failed to track click:', err);
                    });

                    // Redirect to original URL immediately
                    window.location.href = link.original_url;
                } else {
                    setError('Invalid link');
                    setLoading(false);
                }
            } catch (err: any) {
                console.error('Redirect error:', err);
                setError(err.message || 'Link not found');
                setLoading(false);
            }
        };

        if (shortCode) {
            redirect();
        }
    }, [shortCode]);

    if (loading) {
        return (
            <div className="min-h-screen bg-gray-50 flex items-center justify-center">
                <div className="text-center">
                    <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
                    <p className="text-gray-600">Redirecting...</p>
                </div>
            </div>
        );
    }

    if (error) {
        return (
            <div className="min-h-screen bg-gray-50 flex items-center justify-center">
                <div className="text-center">
                    <div className="text-red-600 text-xl font-semibold mb-4">Error</div>
                    <p className="text-gray-600 mb-4">{error}</p>
                    <a href="/" className="text-blue-600 hover:text-blue-800 underline">
                        Go to Home
                    </a>
                </div>
            </div>
        );
    }

    return null;
}
