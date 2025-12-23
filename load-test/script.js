import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

const errorRate = new Rate('errors');

export const options = {
  stages: [
    { duration: '30s', target: 100 },   // Ramp to 100 RPS
    { duration: '30s', target: 500 },   // Ramp to 500 RPS
    { duration: '30s', target: 1000 },  // Ramp to 1000 RPS
    { duration: '30s', target: 1000 },  // Hold at 1000 RPS
  ],
  thresholds: {
    http_req_duration: ['p(95)<100', 'p(99)<200'], // 95% < 100ms, 99% < 200ms
    http_req_failed: ['rate<0.01'], // Error rate < 1%
    errors: ['rate<0.01'],
  },
};

// Setup phase: create test links
export function setup() {
  const links = [];
  const baseURL = __ENV.BASE_URL || 'http://localhost:8080';
  
  console.log('Creating test links...');
  
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
  return { links, baseURL };
}

export default function(data) {
  // Randomly select a short code
  if (data.links.length === 0) {
    console.error('No links available for testing');
    return;
  }
  
  const shortCode = data.links[Math.floor(Math.random() * data.links.length)];
  const url = `${data.baseURL}/${shortCode}`;
  
  // Test redirect endpoint
  const res = http.get(url, {
    redirects: 0, // Don't follow redirects
    tags: { name: 'Redirect' },
  });
  
  const success = check(res, {
    'status is 302': (r) => r.status === 302,
    'redirect latency < 50ms': (r) => r.timings.duration < 50,
  });
  
  if (!success) {
    errorRate.add(1);
  } else {
    errorRate.add(0);
  }
  
  sleep(0.1); // Small delay between requests
}

