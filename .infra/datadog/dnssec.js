// Runs in the Datadog API assertion sandbox, not on dario.cat.
const answer = JSON.parse(dd.response.body);
dd.assert(answer.Status === 0 && answer.AD === true && answer.CD === false, "Resolver must authenticate the DNS answer without disabling validation");
dd.assert(Array.isArray(answer.Question) && answer.Question.some(question => question.name.toLowerCase().replace(/\.$/, "") === "dario.cat" && question.type === 1), "Resolver must answer the dario.cat A query");
function ipv4(value) {
  return typeof value === "string" && /^(\d{1,3}\.){3}\d{1,3}$/.test(value) && value.split(".").every(octet => Number(octet) <= 255 && String(Number(octet)) === octet);
}
dd.assert(Array.isArray(answer.Answer) && answer.Answer.some(record => record.name.toLowerCase().replace(/\.$/, "") === "dario.cat" && record.type === 1 && ipv4(record.data)), "Authenticated A answer is missing");
