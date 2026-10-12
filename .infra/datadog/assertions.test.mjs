import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { test } from 'node:test';
import vm from 'node:vm';

const expiry = readFileSync(new URL('./expiry.js', import.meta.url), 'utf8');
const dnssec = readFileSync(new URL('./dnssec.js', import.meta.url), 'utf8');
const now = Date.parse('2026-10-12T08:17:15Z');
class Clock extends Date { static now() { return now; } }
function run(code, body) {
  vm.runInNewContext(code, { dd: { response: { body: JSON.stringify(body) }, assert }, Date: Clock }, { timeout: 1000 });
}
function domain(days) {
  return { objectClassName: 'domain', ldhName: 'dario.cat', events: [{ eventAction: 'expiration', eventDate: new Date(now + days * 86400000).toISOString() }] };
}

test('registry warnings transition independently at 90, 60 and 30 days', () => {
  for (const days of [90, 60, 30]) {
    const code = expiry.replaceAll('__MINIMUM_DAYS__', String(days));
    run(code, domain(days + 1));
    assert.throws(() => run(code, domain(days)), /expiration is within/);
    assert.throws(() => run(code, domain(days - 1)), /expiration is within/);
  }
});

test('RDAP fails closed for wrong domains, absent/invalid dates and expired registrations', () => {
  const code = expiry.replaceAll('__MINIMUM_DAYS__', '90');
  for (const bad of [
    {}, { ...domain(365), ldhName: 'example.org' },
    { ...domain(365), events: [] },
    { ...domain(365), events: [{ eventAction: 'expiration', eventDate: 'invalid' }] },
    { ...domain(365), events: [{ eventAction: 'expiration', eventDate: '2028' }] },
    { ...domain(365), events: [{ eventAction: 'expiration', eventDate: '2028-06-01T07:05:04' }] },
    ...['2028-02-30T07:05:04Z', '2100-02-29T07:05:04Z', '2028-04-31T07:05:04Z', '2028-06-01T24:00:00Z', '2028-06-01T07:05:04+24:00'].map(eventDate => ({ ...domain(365), events: [{ eventAction: 'expiration', eventDate }] })),
    { ...domain(365), events: [{ eventAction: 'registrar expiration', eventDate: domain(365).events[0].eventDate }] },
    domain(-1),
  ]) assert.throws(() => run(code, bad));
  const multiple = domain(365);
  multiple.events.push(domain(10).events[0]);
  assert.throws(() => run(code, multiple), /expiration is within/);
  run(code, { ...domain(365), events: [{ eventAction: 'expiration', eventDate: '2028-02-29T07:05:04+01:00' }] });
});

const authenticated = {
  Status: 0, AD: true, CD: false,
  Question: [{ name: 'dario.cat.', type: 1 }],
  Answer: [{ name: 'dario.cat.', type: 1, data: '104.21.49.46' }],
};

test('DNSSEC accepts current authenticated A records without pinning a Cloudflare IP', () => {
  run(dnssec, authenticated);
  run(dnssec, { ...authenticated, Answer: [{ name: 'dario.cat', type: 1, data: '188.114.96.12' }] });
});

test('DNSSEC rejects validation errors, unsigned/incomplete answers and different questions', () => {
  for (const bad of [
    {}, { ...authenticated, AD: false }, { ...authenticated, CD: true },
    { ...authenticated, Status: 2 }, { ...authenticated, Answer: [] },
    { ...authenticated, Question: [{ name: 'example.org.', type: 1 }] },
    { ...authenticated, Answer: [{ name: 'dario.cat.', type: 46, data: 'signed denial' }] },
    { ...authenticated, Answer: [{ name: 'example.org.', type: 1, data: '104.21.49.46' }] },
    ...['999.999.999.999', '256.1.1.1', '001.2.3.4', '1.2.3', 'IPv4'].map(data => ({ ...authenticated, Answer: [{ name: 'dario.cat.', type: 1, data }] })),
  ]) assert.throws(() => run(dnssec, bad));
});
