const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api';

export async function createLink(url: string, userId: string) {
    const response = await fetch(`${API_BASE}/links`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url, user_id: userId }),
    });

    if (!response.ok) {
        const error = await response.text();
        throw new Error(error || 'Failed to create link');
    }

    return response.json();
}

export async function getLinks(userId: string) {
    const response = await fetch(`${API_BASE}/links?user_id=${userId}`);

    if (!response.ok) {
        throw new Error('Failed to fetch links');
    }

    return response.json();
}

export async function getLink(shortCode: string) {
    const response = await fetch(`${API_BASE}/links/${shortCode}`);

    if (!response.ok) {
        throw new Error('Failed to fetch link');
    }

    return response.json();
}

export async function getAnalytics(shortCode: string, period: '24h' | '7d' | '30d' = '24h') {
    const response = await fetch(`${API_BASE}/analytics/${shortCode}?period=${period}`);

    if (!response.ok) {
        throw new Error('Failed to fetch analytics');
    }

    return response.json();
}

export async function trackClick(shortCode: string) {
    const response = await fetch(`${API_BASE}/track/${shortCode}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
    });

    if (!response.ok) {
        throw new Error('Failed to track click');
    }

    return response.json();
}
