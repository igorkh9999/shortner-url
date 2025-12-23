'use client';

import { Line } from 'react-chartjs-2';
import {
    Chart as ChartJS,
    CategoryScale,
    LinearScale,
    PointElement,
    LineElement,
    Title,
    Tooltip,
    Legend,
    TimeScale,
} from 'chart.js';
import 'chartjs-adapter-date-fns';

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend, TimeScale);

interface Props {
    data: Array<{ timestamp: string; count: number }>;
    period?: '24h' | '7d' | '30d';
}

export function ClicksChart({ data, period = '24h' }: Props) {
    // Handle empty or null data
    const safeData = data || [];

    const chartData = {
        labels: safeData.map((d) => new Date(d.timestamp)),
        datasets: [
            {
                label: 'Clicks',
                data: safeData.map((d) => d.count),
                borderColor: 'rgb(59, 130, 246)',
                backgroundColor: 'rgba(59, 130, 246, 0.1)',
                tension: 0.4,
                fill: true,
            },
        ],
    };

    // Show message if no data
    if (safeData.length === 0) {
        return (
            <div className="bg-white rounded-lg shadow p-6">
                <h3 className="text-lg font-semibold text-gray-900 mb-4">Clicks Over Time</h3>
                <div className="flex items-center justify-center h-[300px]">
                    <p className="text-gray-500">No click data available for this period</p>
                </div>
            </div>
        );
    }

    // Determine time unit based on period
    const timeUnit = period === '24h' ? 'hour' : 'day';

    const options = {
        responsive: true,
        maintainAspectRatio: false,
        scales: {
            x: {
                type: 'time' as const,
                time: {
                    unit: timeUnit as 'hour' | 'day',
                    displayFormats: {
                        hour: 'MMM dd HH:mm',
                        day: 'MMM dd',
                    },
                },
                title: {
                    display: true,
                    text: 'Time',
                },
            },
            y: {
                beginAtZero: true,
                title: {
                    display: true,
                    text: 'Clicks',
                },
            },
        },
        plugins: {
            legend: {
                display: false,
            },
            tooltip: {
                mode: 'index' as const,
                intersect: false,
            },
        },
    };

    return (
        <div className="bg-white rounded-lg shadow p-6">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Clicks Over Time</h3>
            <div style={{ height: '300px' }}>
                <Line data={chartData} options={options} />
            </div>
        </div>
    );
}
