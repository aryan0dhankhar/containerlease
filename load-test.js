import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
  vus: 10,
  duration: '30s',
  thresholds: {
    http_req_duration: ['p(95)<500'],
    http_req_failed: ['rate<0.1'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function() {
  // 1. Health check (no auth required)
  let res = http.get(`${BASE_URL}/health`);
  check(res, {
    'health status 200': (r) => r.status === 200,
  });

  // 2. Register a new user (with unique email per VU)
  const email = `user-${__VU}-${__ITER}@test.local`;
  res = http.post(`${BASE_URL}/api/auth/register`, JSON.stringify({
    email: email,
    username: `user-${__VU}-${__ITER}`,
    password: 'TestPass123',
    tenantId: `tenant-${__VU}`,
  }), {
    headers: { 'Content-Type': 'application/json' },
  });
  check(res, {
    'register status 201': (r) => r.status === 201,
    'register returns token': (r) => r.json() && r.json().token,
  });

  let token = '';
  if (res.status === 201) {
    token = res.json().token;
  }

  sleep(1);

  // 3. Login with credentials (if register succeeded)
  if (token) {
    res = http.post(`${BASE_URL}/api/auth/login`, JSON.stringify({
      email: email,
      password: 'TestPass123',
    }), {
      headers: { 'Content-Type': 'application/json' },
    });
    check(res, {
      'login status 200': (r) => r.status === 200,
      'login returns token': (r) => r.json() && r.json().token,
    });
    token = res.json().token || token;
  }

  sleep(1);

  // 4. List containers (with auth)
  if (token) {
    res = http.get(`${BASE_URL}/api/containers`, {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    });
    check(res, {
      'list containers status 200': (r) => r.status === 200,
      'list containers returns array': (r) => Array.isArray(r.json()?.containers),
    });
  }

  sleep(1);

  // 5. Provision a container
  if (token) {
    res = http.post(`${BASE_URL}/api/provision`, JSON.stringify({
      imageType: 'ubuntu',
      durationMinutes: 10,
      cpuMilli: 500,
      memoryMB: 512,
    }), {
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
    });
    check(res, {
      'provision status 201': (r) => r.status === 201,
      'provision returns container id': (r) => r.json() && r.json().id,
    });
  }

  sleep(2);
}
