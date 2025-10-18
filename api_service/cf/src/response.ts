import { Context } from 'hono';

/**
 * Standardized API response structure, mirroring the Go backend.
 */
interface APIResponse {
  code: number;
  message: string;
  data?: any;
}

/**
 * A helper function to create standardized JSON responses.
 * @param c Hono context
 * @param statusCode HTTP status code
 * @param message A descriptive message
 * @param data The payload to be sent (optional)
 * @returns A Response object.
 */
export function respond(c: Context, statusCode: number, message: string, data?: any) {
  const response: APIResponse = {
    code: statusCode,
    message,
  };
  if (data !== undefined) {
    response.data = data;
  }
  return c.json(response, statusCode as any);
}
