const datetimeLocalLength = 16;

export function datetimeLocalToRFC3339(value?: string | null): string | undefined {
  if (!value) return undefined;

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;

  return date.toISOString();
}

export function rfc3339ToDatetimeLocal(value?: string | null): string | undefined {
  if (!value) return undefined;

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;

  const offsetMs = date.getTimezoneOffset() * 60 * 1000;
  return new Date(date.getTime() - offsetMs)
    .toISOString()
    .slice(0, datetimeLocalLength);
}
