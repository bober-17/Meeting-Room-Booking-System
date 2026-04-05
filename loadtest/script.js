import http from 'k6/http';
import { check, group } from 'k6';
import { Rate, Trend } from 'k6/metrics';
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.0.2/index.js';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Комната 2 — расписание каждый день 08:00–20:00 (из seed.sql)
const ROOM_ID = '10000000-0000-0000-0000-000000000002';

const errorRate = new Rate('errors');
const slotsDuration = new Trend('slots_duration', true);

export const options = {
  scenarios: {
    // Основной эндпоинт: список доступных слотов — 100 RPS согласно условиям задания
    list_slots: {
      executor: 'constant-arrival-rate',
      rate: 100,
      timeUnit: '1s',
      duration: '1m',
      preAllocatedVUs: 20,
      maxVUs: 50,
      exec: 'listSlots',
    },
    // Сценарий бронирования: создание + отмена — 10 RPS
    bookings: {
      executor: 'constant-arrival-rate',
      rate: 10,
      timeUnit: '1s',
      duration: '1m',
      preAllocatedVUs: 5,
      maxVUs: 20,
      exec: 'bookAndCancel',
      startTime: '5s', // дать время прогреться слотам
    },
  },
  thresholds: {
    // SLI из задания: p95 < 200ms для основного эндпоинта
    'slots_duration': ['p(95)<200'],
    // SLI успешности: 99.9%
    'http_req_failed': ['rate<0.001'],
    'errors': ['rate<0.001'],
  },
};

export function setup() {
  const adminRes = http.post(
    `${BASE_URL}/dummyLogin`,
    JSON.stringify({ role: 'admin' }),
    { headers: { 'Content-Type': 'application/json' } },
  );
  const userRes = http.post(
    `${BASE_URL}/dummyLogin`,
    JSON.stringify({ role: 'user' }),
    { headers: { 'Content-Type': 'application/json' } },
  );
  return {
    adminToken: adminRes.json('token'),
    userToken: userRes.json('token'),
  };
}

// Завтрашняя дата в UTC — гарантирует наличие будущих слотов
function tomorrow() {
  const d = new Date();
  d.setUTCDate(d.getUTCDate() + 1);
  return d.toISOString().split('T')[0];
}

export function listSlots(data) {
  const res = http.get(
    `${BASE_URL}/rooms/${ROOM_ID}/slots/list?date=${tomorrow()}`,
    { headers: { Authorization: `Bearer ${data.userToken}` } },
  );

  slotsDuration.add(res.timings.duration);

  const ok = check(res, {
    'slots: status 200': (r) => r.status === 200,
    'slots: p95 < 200ms': (r) => r.timings.duration < 200,
  });
  errorRate.add(!ok);
}

export function bookAndCancel(data) {
  const userHeaders = {
    headers: {
      Authorization: `Bearer ${data.userToken}`,
      'Content-Type': 'application/json',
    },
  };

  // Получить доступные слоты
  const slotsRes = http.get(
    `${BASE_URL}/rooms/${ROOM_ID}/slots/list?date=${tomorrow()}`,
    { headers: { Authorization: `Bearer ${data.userToken}` } },
  );
  if (slotsRes.status !== 200) {
    errorRate.add(1);
    return;
  }

  const slots = slotsRes.json('slots');
  if (!slots || slots.length === 0) return;

  // Выбрать случайный слот
  const slot = slots[Math.floor(Math.random() * slots.length)];

  group('create and cancel booking', () => {
    const bookRes = http.post(
      `${BASE_URL}/bookings/create`,
      JSON.stringify({ slotId: slot.id, createConferenceLink: false }),
      userHeaders,
    );

    // 409 (слот уже занят другим VU) — ожидаемо при параллельной нагрузке
    const bookOk = check(bookRes, {
      'booking: created or conflict': (r) => r.status === 201 || r.status === 409,
    });
    errorRate.add(!bookOk);

    if (bookRes.status === 201) {
      const bookingID = bookRes.json('booking').id;
      const cancelRes = http.post(
        `${BASE_URL}/bookings/${bookingID}/cancel`,
        null,
        userHeaders,
      );
      check(cancelRes, { 'cancel: status 200': (r) => r.status === 200 });
    }
  });
}

export function handleSummary(data) {
  return {
    '/loadtest/results/report.txt': textSummary(data, { indent: ' ', enableColors: false }),
    stdout: textSummary(data, { indent: ' ', enableColors: true }),
  };
}
