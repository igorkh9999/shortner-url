import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const redirectLatency = new Trend('redirect_latency');

export const options = {
    stages: [
        { duration: '30s', target: 100 }, // Warm-up: Ramp to 100 RPS
        { duration: '1m', target: 500 }, // Ramp up: 500 RPS
        { duration: '1m', target: 1000 }, // Peak load: 1000 RPS
        { duration: '30s', target: 500 }, // Ramp down: 500 RPS
        { duration: '30s', target: 0 }, // Cool down
    ],
    thresholds: {
        errors: ['rate<0.01'], // <1% error rate
        http_req_duration: ['p(95)<100', 'p(99)<200'], // Latency targets
        http_req_failed: ['rate<0.01'], // <1% failed requests
        redirect_latency: ['p(95)<50'], // <50ms redirect time
    },
    noConnectionReuse: false, // Reuse connections for better performance
    userAgent: 'k6/load-test',
};

// Setup phase: create test links and warm cache
export function setup() {
    const links = [];
    const baseURL = __ENV.BASE_URL || 'http://localhost:8080';

    // Health check first
    console.log('Checking server health...');
    const healthRes = http.get(`${baseURL}/health`);
    check(healthRes, {
        'health check passed': (r) => r.status === 200,
    });

    console.log('Creating test links...');

    // Create 100 test links
    for (let i = 0; i < 100; i++) {
        const payload = JSON.stringify({
            url: `https://example.com/page${i}`,
            user_id: 'loadtest',
        });

        const params = {
            headers: { 'Content-Type': 'application/json' },
        };

        const res = http.post(`${baseURL}/api/links`, payload, params);

        if (res.status === 201) {
            const body = JSON.parse(res.body);
            links.push(body.short_code);
        } else {
            console.log(`Failed to create link ${i}: ${res.status} - ${res.body}`);
        }

        sleep(0.1); // Small delay between creations
    }

    console.log(`Created ${links.length} test links`);

    // Pre-warm cache by making a request to each link
    // This populates both L1 (in-memory) and L2 (Redis) caches
    console.log('Pre-warming cache...');
    for (let i = 0; i < links.length; i++) {
        const url = `${baseURL}/${links[i]}`;
        http.get(url, { redirects: 0 });
        // Small delay to avoid overwhelming the server
        if (i % 20 === 0) {
            sleep(0.1);
        }
    }
    // Give server a moment to process all warm-up requests and populate L1 cache
    sleep(2);
    console.log('Cache pre-warmed');

    return { links, baseURL };
}

export default function (data) {
    // Randomly select a short code
    if (data.links.length === 0) {
        console.error('No links available for testing');
        return;
    }

    const shortCode = data.links[Math.floor(Math.random() * data.links.length)];
    const url = `${data.baseURL}/${shortCode}`;

    // Test redirect endpoint with timing
    const startTime = Date.now();
    const res = http.get(url, {
        redirects: 0, // Don't follow redirects
        timeout: '5s',
        tags: { name: 'Redirect' },
    });
    const duration = Date.now() - startTime;

    // Record redirect latency
    redirectLatency.add(duration);

    // Checks
    const success = check(res, {
        'status is 302': (r) => r.status === 302,
        'has location header': (r) => r.headers['Location'] !== undefined,
        'redirect latency < 50ms': (r) => duration < 50,
    });

    errorRate.add(!success);

    // Log failures
    if (!success) {
        console.error(`Failed: ${shortCode}, Status=${res.status}, Duration=${duration}ms`);
    }

    sleep(0.1); // Small delay between requests
}

export function teardown(data) {
    console.log('Test completed');
    // Optionally check metrics endpoint
    const metricsRes = http.get(`${data.baseURL}/metrics`);
    if (metricsRes.status === 200) {
        console.log('Final metrics:', metricsRes.body);
    }
}
