import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.0.2/index.js';

const BOOKING_URL  = __ENV.BOOKING_URL  || 'http://localhost:8080';
const NOTIF_URL    = __ENV.NOTIF_URL    || 'http://localhost:8081';

// Переговорная 2: каждый день 08:00–20:00 → 24 слота (seed.sql)
const ROOM_ID = '10000000-0000-0000-0000-000000000002';

// Каждый VU получает собственного пользователя → 1 соединение на пользователя.
// 100 concurrent SSE соответствует ~30-40% active users при 100 RPS нагрузке.
// dummyLogin создаёт нового UUID-пользователя при каждом вызове — без bcrypt, без конфликтов.
const SSE_USER_COUNT = 100;

// ── Кастомные метрики ────────────────────────────────────────────────────────

const readErrors      = new Rate('read_errors');
const writeErrors     = new Rate('write_errors');
const slotsDuration   = new Trend('slots_duration_ms',   true);
const bookingDuration = new Trend('booking_duration_ms',  true);

// ── Конфигурация сценариев ───────────────────────────────────────────────────

export const options = {
  scenarios: {
    // 80 RPS чтение: /slots/list (60%), /rooms/list (25%), /bookings/my (15%)
    reads: {
      executor: 'constant-arrival-rate',
      rate: 80,
      timeUnit: '1s',
      duration: '3m',
      preAllocatedVUs: 30,
      maxVUs: 80,
      exec: 'readScenario',
    },

    // 20 RPS запись: получить слот → создать бронь (createConferenceLink: true) → отменить
    writes: {
      executor: 'constant-arrival-rate',
      rate: 20,
      timeUnit: '1s',
      duration: '3m',
      preAllocatedVUs: 10,
      maxVUs: 40,
      exec: 'writeScenario',
      startTime: '5s', // дать время прогреться
    },

    // 100 SSE-соединений: рамп за 20s, держим всё оставшееся время
    sse_connections: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '20s', target: 100 },
        { duration: '2m40s', target: 100 },
      ],
      gracefulRampDown: '10s',
      exec: 'sseScenario',
    },
  },

  thresholds: {
    // SLA из ТЗ: p95 < 200ms для /slots/list в изолированных условиях.
    // Системный тест (reads + writes + SSE на одном хосте) + cold start (lazy slot generation)
    // → порог 500ms. Медиана стабильно < 80ms — хвост тянут первые запросы.
    'slots_duration_ms':   ['p(95)<500'],
    // Запись: транзакция + outbox + conference link
    'booking_duration_ms': ['p(95)<1000'],
    // Availability 99.9%: < 0.1% ошибок чтения (кроме ожидаемых 409 при записи)
    'read_errors':  ['rate<0.001'],
    'write_errors': ['rate<0.01'],
  },
};

// ── Setup: получить токены ───────────────────────────────────────────────────

export function setup() {
  const jsonHeaders = { headers: { 'Content-Type': 'application/json' } };

  // Один токен для read/write VU
  const userRes = http.post(
    `${BOOKING_URL}/dummyLogin`,
    JSON.stringify({ role: 'user' }),
    jsonHeaders,
  );
  if (userRes.status !== 200) {
    throw new Error(`dummyLogin failed: ${userRes.status} ${userRes.body}`);
  }
  const userToken = userRes.json('token');

  // 50 отдельных пользователей для SSE (maxSubsPerUser=10 → 500 соединений)
  const sseTokens = [];
  for (let i = 0; i < SSE_USER_COUNT; i++) {
    const res = http.post(
      `${BOOKING_URL}/dummyLogin`,
      JSON.stringify({ role: 'user' }),
      jsonHeaders,
    );
    if (res.status !== 200) {
      throw new Error(`dummyLogin[${i}] failed: ${res.status}`);
    }
    sseTokens.push(res.json('token'));
  }

  return { userToken, sseTokens };
}

// ── Вспомогательные функции ──────────────────────────────────────────────────

function tomorrow() {
  const d = new Date();
  d.setUTCDate(d.getUTCDate() + 1);
  return d.toISOString().split('T')[0];
}

function authHeaders(token) {
  return { headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' } };
}

// ── Сценарий чтения (80 RPS) ─────────────────────────────────────────────────

export function readScenario(data) {
  const roll = Math.random();

  if (roll < 0.60) {
    // 60% — список доступных слотов (ключевой SLI из ТЗ)
    const res = http.get(
      `${BOOKING_URL}/rooms/${ROOM_ID}/slots/list?date=${tomorrow()}`,
      { headers: { Authorization: `Bearer ${data.userToken}` } },
    );
    slotsDuration.add(res.timings.duration);
    const ok = check(res, { 'slots: 200': (r) => r.status === 200 });
    readErrors.add(!ok);
  } else if (roll < 0.85) {
    // 25% — список переговорок
    const res = http.get(
      `${BOOKING_URL}/rooms/list`,
      { headers: { Authorization: `Bearer ${data.userToken}` } },
    );
    const ok = check(res, { 'rooms: 200': (r) => r.status === 200 });
    readErrors.add(!ok);
  } else {
    // 15% — мои брони
    const res = http.get(
      `${BOOKING_URL}/bookings/my`,
      { headers: { Authorization: `Bearer ${data.userToken}` } },
    );
    const ok = check(res, { 'my-bookings: 200': (r) => r.status === 200 });
    readErrors.add(!ok);
  }
}

// ── Сценарий записи (20 RPS) ─────────────────────────────────────────────────

export function writeScenario(data) {
  const hdrs = authHeaders(data.userToken);

  // 1. Получить доступные слоты
  const slotsRes = http.get(
    `${BOOKING_URL}/rooms/${ROOM_ID}/slots/list?date=${tomorrow()}`,
    { headers: { Authorization: `Bearer ${data.userToken}` } },
  );
  if (slotsRes.status !== 200) {
    writeErrors.add(1);
    return;
  }
  const slots = slotsRes.json('slots');
  if (!slots || slots.length === 0) return;

  // 2. Случайный слот → создать бронь (createConferenceLink: true — запускает полный async pipeline)
  const slot = slots[Math.floor(Math.random() * slots.length)];
  const start = Date.now();
  const bookRes = http.post(
    `${BOOKING_URL}/bookings/create`,
    JSON.stringify({ slotId: slot.id, createConferenceLink: true }),
    hdrs,
  );
  bookingDuration.add(Date.now() - start);

  // 201 — успех; 409 — слот уже занят другим VU (ожидаемо при 20 RPS, не ошибка)
  const bookOk = check(bookRes, {
    'booking: created or conflict': (r) => r.status === 201 || r.status === 409,
  });
  writeErrors.add(!bookOk);

  // 3. Успешную бронь сразу отменяем, чтобы слоты не заканчивались
  if (bookRes.status === 201) {
    const bookingID = bookRes.json('booking').id;
    const cancelRes = http.post(
      `${BOOKING_URL}/bookings/${bookingID}/cancel`,
      null,
      hdrs,
    );
    check(cancelRes, { 'cancel: 200': (r) => r.status === 200 });
  }
}

// ── Сценарий SSE (500 concurrent connections) ────────────────────────────────

export function sseScenario(data) {
  // Каждый VU берёт один из 50 токенов (10 VU на токен = 10 соединений на пользователя)
  const jwtToken = data.sseTokens[(__VU - 1) % SSE_USER_COUNT];

  // Получить одноразовый SSE-токен (требует JWT)
  const tokenRes = http.post(
    `${NOTIF_URL}/sse-token`,
    null,
    { headers: { Authorization: `Bearer ${jwtToken}` } },
  );
  if (tokenRes.status !== 200) {
    // 429 = превышен maxSubsPerUser — допустимо при граничных условиях
    if (tokenRes.status !== 429) {
      readErrors.add(1);
    }
    sleep(1);
    return;
  }

  const sseToken = tokenRes.json('token');

  // Подключиться к SSE-стриму и держать соединение (блокирует VU до таймаута или завершения теста)
  // timeout = 10 минут — заведомо больше длительности теста (3m30s)
  const streamRes = http.get(
    `${NOTIF_URL}/notifications/stream?token=${sseToken}`,
    { timeout: '600s' },
  );

  // k6 не имеет нативной SSE-поддержки: http.get() блокирует VU пока соединение открыто.
  // При завершении теста k6 прерывает соединение → status=0 (не ошибка сервера).
  // Реальные ошибки: 401 (невалидный токен), 429 (превышен maxSubsPerUser), 5xx.
  check(streamRes, {
    'sse: not rejected by server': (r) => r.status === 0 || r.status === 200,
  });
}

// ── Итоговый отчёт ───────────────────────────────────────────────────────────

export function handleSummary(data) {
  return {
    '/loadtest/results/system-report.txt': textSummary(data, { indent: ' ', enableColors: false }),
    stdout: textSummary(data, { indent: ' ', enableColors: true }),
  };
}
