// Runs in the Datadog API assertion sandbox, not on dario.cat.
const domain = JSON.parse(dd.response.body);
dd.assert(domain.objectClassName === "domain" && typeof domain.ldhName === "string" && domain.ldhName.toLowerCase() === "dario.cat", "RDAP must describe dario.cat");
dd.assert(Array.isArray(domain.events), "RDAP events are missing");
const expirations = domain.events.filter(event => event && event.eventAction === "expiration");
dd.assert(expirations.length > 0, "Registry expiration is missing");
function expirationDate(value) {
  if (typeof value !== "string") return NaN;
  const parts = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.\d+)?(?:Z|([+-])(\d{2}):(\d{2}))$/.exec(value);
  if (!parts) return NaN;
  const [year, month, day, hour, minute, second] = parts.slice(1, 7).map(Number);
  const leap = year % 4 === 0 && (year % 100 !== 0 || year % 400 === 0);
  const monthDays = [31, leap ? 29 : 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31];
  if (month < 1 || month > 12 || day < 1 || day > monthDays[month - 1] || hour > 23 || minute > 59 || second > 59) return NaN;
  if (parts[7] && (Number(parts[8]) > 23 || Number(parts[9]) > 59)) return NaN;
  return Date.parse(value);
}
const dates = expirations.map(event => expirationDate(event.eventDate));
dd.assert(dates.every(date => Number.isFinite(date)), "Registry expiration is invalid");
// Use the earliest if the registry supplies multiple expiration events.
const daysRemaining = (Math.min(...dates) - Date.now()) / 86400000;
dd.assert(daysRemaining > __MINIMUM_DAYS__, "Registry expiration is within __MINIMUM_DAYS__ days: " + daysRemaining.toFixed(2) + " days remain");
