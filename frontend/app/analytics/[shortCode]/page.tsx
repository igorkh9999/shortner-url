'use client';

import { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import { getAnalytics, getLink } from '@/lib/api';
import { Analytics, Link } from '@/types';
import { RealtimeCounter } from '@/components/RealtimeCounter';
import { ClicksChart } from '@/components/ClicksChart';
import { ReferrersTable } from '@/components/ReferrersTable';
import { StatsCard } from '@/components/StatsCard';

export default function AnalyticsPage() {
    const params = useParams();
    const shortCode = params.shortCode as string;

    const [link, setLink] = useState<Link | null>(null);
    const [analytics, setAnalytics] = useState<Analytics | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [period, setPeriod] = useState<'24h' | '7d' | '30d'>('24h');

    useEffect(() => {
        fetchData();
    }, [shortCode, period]);

    const fetchData = async () => {
        try {
            setLoading(true);
            const [linkData, analyticsData] = await Promise.all([getLink(shortCode), getAnalytics(shortCode, period)]);
            setLink(linkData);
            setAnalytics(analyticsData);
        } catch (err: any) {
            setError(err.message || 'Failed to load analytics');
        } finally {
            setLoading(false);
        }
    };

    if (loading) {
        return (
            <div className="min-h-screen bg-gray-50 flex items-center justify-center">
                <div className="text-gray-500">Loading analytics...</div>
            </div>
        );
    }

    if (error || !link || !analytics) {
        return (
            <div className="min-h-screen bg-gray-50 flex items-center justify-center">
                <div className="text-red-600">{error || 'Failed to load analytics'}</div>
            </div>
        );
    }

    return (
        <div className="min-h-screen bg-gray-50">
            <nav className="bg-white shadow-sm">
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                    <div className="flex justify-between h-16">
                        <div className="flex items-center">
                            <h1 className="text-xl font-bold text-gray-900">Link Analytics</h1>
                        </div>
                        <div className="flex items-center space-x-4">
                            <a href="/" className="text-gray-600 hover:text-gray-900 px-3 py-2 rounded-md text-sm font-medium">
                                Home
                            </a>
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

            <main className="max-w-7xl mx-auto py-12 px-4 sm:px-6 lg:px-8">
                {/* Link Info */}
                <div className="bg-white rounded-lg shadow p-6 mb-6">
                    <h2 className="text-2xl font-bold text-gray-900 mb-4">Link Details</h2>
                    <div className="space-y-2">
                        <div>
                            <span className="text-sm font-medium text-gray-500">Short Code: </span>
                            <span className="text-sm text-gray-900">{link.short_code}</span>
                        </div>
                        <div>
                            <span className="text-sm font-medium text-gray-500">Original URL: </span>
                            <a
                                href={link.original_url}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="text-sm text-blue-600 hover:underline"
                            >
                                {link.original_url}
                            </a>
                        </div>
                        <div>
                            <span className="text-sm font-medium text-gray-500">Created: </span>
                            <span className="text-sm text-gray-900">{new Date(link.created_at).toLocaleString()}</span>
                        </div>
                    </div>
                </div>

                {/* Period Selector */}
                <div className="mb-6 flex gap-2">
                    <button
                        onClick={() => setPeriod('24h')}
                        className={`px-4 py-2 rounded-lg ${
                            period === '24h' ? 'bg-blue-600 text-white' : 'bg-white text-gray-700 hover:bg-gray-50'
                        }`}
                    >
                        24 Hours
                    </button>
                    <button
                        onClick={() => setPeriod('7d')}
                        className={`px-4 py-2 rounded-lg ${
                            period === '7d' ? 'bg-blue-600 text-white' : 'bg-white text-gray-700 hover:bg-gray-50'
                        }`}
                    >
                        7 Days
                    </button>
                    <button
                        onClick={() => setPeriod('30d')}
                        className={`px-4 py-2 rounded-lg ${
                            period === '30d' ? 'bg-blue-600 text-white' : 'bg-white text-gray-700 hover:bg-gray-50'
                        }`}
                    >
                        30 Days
                    </button>
                </div>

                {/* Stats Grid */}
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-6">
                    <RealtimeCounter shortCode={shortCode} initialCount={analytics.total_clicks} />
                    <StatsCard title="Unique Visitors" value={analytics.unique_visitors.toLocaleString()} />
                    <StatsCard
                        title={period === '24h' ? 'Clicks/Hour' : 'Clicks/Day'}
                        value={
                            analytics.click_rate
                                ? analytics.click_rate.toFixed(1)
                                : period === '24h'
                                ? analytics.total_clicks.toLocaleString()
                                : Math.round(analytics.total_clicks / (period === '7d' ? 7 : 30)).toLocaleString()
                        }
                        subtitle={`Average over ${period}`}
                    />
                    {analytics.peak_hour ? (
                        <StatsCard
                            title={period === '24h' ? 'Peak Hour' : 'Peak Day'}
                            value={analytics.peak_hour.count.toLocaleString()}
                            subtitle={new Date(analytics.peak_hour.timestamp).toLocaleString()}
                        />
                    ) : (
                        <StatsCard title={period === '24h' ? 'Peak Hour' : 'Peak Day'} value="N/A" subtitle="No data available" />
                    )}
                </div>

                {/* Chart */}
                <div className="mb-6">
                    <ClicksChart data={analytics.clicks_over_time || []} period={period} />
                </div>

                {/* Top Referrers */}
                <div>
                    <ReferrersTable referrers={analytics.top_referrers || []} />
                </div>
            </main>
        </div>
    );
}
