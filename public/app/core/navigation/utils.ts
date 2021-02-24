import { createLogger } from '@grafana/ui';

export const navigationLogger = createLogger('Router');

export function queryStringToJSON(queryString: string) {
  const params: Array<[string, string | boolean]> = [];
  new URLSearchParams(queryString).forEach((v, k) => params.push([k, parseValue(v)]));
  return Object.fromEntries(new Map(params));
}

function parseValue(value: string) {
  if (value === 'true') {
    return true;
  }
  if (value === 'false') {
    return false;
  }
  return value;
}