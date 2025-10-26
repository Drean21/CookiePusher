import { swaggerUI } from '@hono/swagger-ui';
import { OpenAPIHono } from '@hono/zod-openapi';
import { ZodError } from 'zod';
import type { Cookie, User } from './models';
import { toApiUser } from './presenter';
import { respond } from './response';
import * as schema from './schema';
import { D1Store } from './store';

// --- Types and Context ---
export type Env = {
  DB: D1Database;
  // Secrets are injected directly into the environment and do not need to be in [vars]
  POOL_ACCESS_KEY: string;
  ADMIN_KEY: string;
};

type AppContext = {
  Variables: {
    store: D1Store;
    user: User;
  };
  Bindings: Env;
};

const app = new OpenAPIHono<AppContext>();

// --- Error Handler ---
app.onError((err: any, c) => {
    if (err?.name === 'ZodError' || err?.cause?.name === 'ZodError') {
        const zodError = (err.name === 'ZodError' ? err : err.cause) as ZodError;
        return c.json({ success: false, error: { name: 'ValidationError', details: zodError.flatten().fieldErrors, }, }, 400);
    }
    console.error('Unhandled Error:', err?.message, err?.stack);
    return c.json({ success: false, error: { name: err?.name || 'InternalServerError', message: 'An unexpected internal error occurred.', } }, 500);
});

// --- Middleware ---
app.use('*', (c, next) => {
  c.set('store', new D1Store(c.env.DB, c.env.ADMIN_KEY, c.env.POOL_ACCESS_KEY));
  return next();
});

const authMiddleware = async (c: any, next: any) => {
  const apiKey = c.req.header('x-api-key');
  if (!apiKey) return respond(c, 401, 'x-api-key header required');
  const store = c.get('store');
  const user = await store.getUserByAPIKey(apiKey);
  if (!user) return respond(c, 401, 'Invalid API Key');
  c.set('user', user);
  await next();
};

// The adminOnlyMiddleware is no longer needed as the admin role is being removed.

const poolAuthMiddleware = async (c: any, next: any) => {
    const poolKey = c.req.header('x-pool-key');
    if (!poolKey) return respond(c, 401, 'x-pool-key header required');
    if (poolKey !== c.env.POOL_ACCESS_KEY) {
        return respond(c, 401, 'Invalid Pool Access Key');
    }
    await next();
};

const adminAuthMiddleware = async (c: any, next: any) => {
    const adminKey = c.req.header('x-admin-key');
    if (!adminKey) return respond(c, 401, 'x-admin-key header required');
    if (adminKey !== c.env.ADMIN_KEY) {
        return respond(c, 403, 'Forbidden: Invalid Admin Key');
    }
    await next();
};

// --- OpenAPI Security Scheme Registration ---
// Register all security schemes on the main app instance.
// Hono will automatically associate them with routes that use them during documentation generation.
app.openAPIRegistry.registerComponent('securitySchemes', 'ApiKeyAuth', {
    type: 'apiKey',
    in: 'header',
    name: 'x-api-key',
});
app.openAPIRegistry.registerComponent('securitySchemes', 'PoolKeyAuth', {
    type: 'apiKey',
    in: 'header',
    name: 'x-pool-key',
    description: 'A separate key for services accessing the shared cookie pool.'
});

app.openAPIRegistry.registerComponent('securitySchemes', 'AdminKeyAuth', {
    type: 'apiKey',
    in: 'header',
    name: 'x-admin-key',
    description: 'A secret key for accessing admin-level endpoints.'
});


// =================================================================
// Route Definitions & Mounting
// =================================================================

// --- Public Routes ---
app.openapi(schema.healthCheckRoute, (c) => respond(c, 200, 'Service is healthy'));


// --- User-Authenticated Routes ---
const userApp = new OpenAPIHono<AppContext>();
userApp.use('*', authMiddleware);
// (Handlers are restored here)
userApp.openapi(schema.authTestRoute, (c) => respond(c, 200, 'Token is valid', { user_id: c.get('user').id }));
userApp.openapi(schema.getUserSettingsRoute, (c) => respond(c, 200, 'User settings retrieved successfully', toApiUser(c.get('user'))));
userApp.openapi(schema.updateUserSettingsRoute, async (c) => {
    const { sharing_enabled } = c.req.valid('json');
    await c.get('store').updateUserSharing(c.get('user').id, sharing_enabled);
    return respond(c, 200, 'Settings updated successfully');
});
userApp.openapi(schema.syncRoute, async (c) => {
    await c.get('store').syncCookies(c.get('user').id, c.req.valid('json'));
    return respond(c, 200, 'Sync successful');
});
userApp.openapi(schema.getAllCookiesRoute, async (c) => {
    const { format } = c.req.valid('query');
    const cookies = await c.get('store').getAllCookies(c.get('user').id);
    if (format === 'json') {
        const domainMap: Record<string, Record<string, string>> = {};
        for (const cookie of cookies) {
            if (!domainMap[cookie.domain]) {
                domainMap[cookie.domain] = {};
            }
            domainMap[cookie.domain][cookie.name] = cookie.value;
        }
        return respond(c, 200, 'Successfully retrieved all cookies', domainMap);
    }
    const groupedByDomain: Record<string, string> = {};
    for (const cookie of cookies) {
        if (!groupedByDomain[cookie.domain]) groupedByDomain[cookie.domain] = '';
        groupedByDomain[cookie.domain] += `${cookie.name}=${cookie.value}; `;
    }
    for (const domain in groupedByDomain) {
        groupedByDomain[domain] = groupedByDomain[domain].slice(0, -2);
    }
    return respond(c, 200, 'Successfully retrieved all cookies as header strings', groupedByDomain);
});
userApp.openapi(schema.getDomainCookiesRoute, async (c) => {
    const { domain } = c.req.valid('param');
    const { format } = c.req.valid('query');
    const cookies = await c.get('store').getCookiesByDomain(c.get('user').id, domain);
    if (format === 'json') {
        const cookieMap: Record<string, string> = {};
        for (const cookie of cookies) {
            cookieMap[cookie.name] = cookie.value;
        }
        return respond(c, 200, `Successfully retrieved cookies for domain ${domain}`, cookieMap);
    }
    const headerString = cookies.map(c => `${c.name}=${c.value}`).join('; ');
    return respond(c, 200, `Successfully retrieved cookies for domain ${domain} as header string`, headerString);
});
userApp.openapi(schema.getCookieValueRoute, async (c) => {
    const { domain, name } = c.req.valid('param');
    // With the removal of getCookieByName, we fetch by domain and filter in-memory.
    const cookies = await c.get('store').getCookiesByDomain(c.get('user').id, domain);
    const cookie = cookies.find(c => c.name === name);

    if (!cookie) return respond(c, 404, 'Cookie not found');
    return respond(c, 200, 'Successfully retrieved cookie value', cookie.value);
});
// --- Pool-Authenticated Routes ---
const poolApp = new OpenAPIHono<AppContext>();
poolApp.use('*', poolAuthMiddleware);
poolApp.openapi(schema.getSharableCookiesRoute, async (c) => {
    const { domain } = c.req.valid('param');
    const { format } = c.req.valid('query');
    const cookies = await c.get('store').getSharableCookiesByDomain(domain);
    const cookiesByUser: Record<number, Cookie[]> = {};
    for (const cookie of cookies) {
        const userId = cookie.user_id as number;
        if (!cookiesByUser[userId]) {
            cookiesByUser[userId] = [];
        }
        cookiesByUser[userId].push(cookie);
    }

    if (format === 'json') {
        const result = Object.keys(cookiesByUser).map(userIdStr => {
            const userId = parseInt(userIdStr, 10);
            const userCookies = cookiesByUser[userId];
            const domainMap: Record<string, Record<string, string>> = {};

            for (const cookie of userCookies) {
                if (!domainMap[cookie.domain]) {
                    domainMap[cookie.domain] = {};
                }
                domainMap[cookie.domain][cookie.name] = cookie.value;
            }

            return {
                user_id: userId,
                cookies: domainMap,
            };
        });

        return respond(c, 200, `Successfully retrieved sharable cookies for domain ${domain}`, result);
    }
    const result: string[] = [];
    for (const userId in cookiesByUser) {
        result.push(cookiesByUser[userId].map(c => `${c.name}=${c.value}`).join('; '));
    }
    return respond(c, 200, `Successfully retrieved sharable cookies for domain ${domain} as header strings`, result);
});

// --- Mount all route groups ---
// --- Admin Routes ---
const adminApp = new OpenAPIHono<AppContext>();
adminApp.use('*', adminAuthMiddleware);

adminApp.openapi(schema.adminCreateUserRoute, async (c) => {
    const usersToCreate = c.req.valid('json');
    const store = c.get('store');
    const createdUsers: User[] = [];

    if (usersToCreate.length === 0) {
        // If the body is an empty array, create one user with a default remark.
        const newUser = await store.createUser('Default user');
        createdUsers.push(newUser);
    } else {
        for (const userData of usersToCreate) {
            const newUser = await store.createUser(userData.remark);
            createdUsers.push(newUser);
        }
    }
    
    // For the admin response, we return the full user object, including the API key.
    return respond(c, 201, 'Users created successfully', createdUsers);
});

adminApp.openapi(schema.adminUpdateUserRoute, async (c) => {
    const { id } = c.req.valid('param');
    const { remark } = c.req.valid('json');
    await c.get('store').updateUserRemark(parseInt(id, 10), remark);
    return respond(c, 200, 'User updated successfully');
});

adminApp.openapi(schema.adminUpdateUserByApiKeyRoute, async (c) => {
    const { apiKey } = c.req.valid('param');
    const { remark } = c.req.valid('json');
    await c.get('store').updateUserRemarkByAPIKey(apiKey, remark);
    return respond(c, 200, 'User updated successfully');
});

adminApp.openapi(schema.adminRefreshUserApiKeyRoute, async (c) => {
    const { id } = c.req.valid('param');
    const updatedUser = await c.get('store').adminUpdateUserAPIKey(parseInt(id, 10));
    if (!updatedUser) return respond(c, 404, 'User not found');
    return respond(c, 200, 'API key refreshed successfully', updatedUser);
});

adminApp.openapi(schema.adminRefreshUserApiKeyByApiKeyRoute, async (c) => {
    const { apiKey } = c.req.valid('param');
    const updatedUser = await c.get('store').adminUpdateUserAPIKeyByAPIKey(apiKey);
    if (!updatedUser) return respond(c, 404, 'User not found');
    return respond(c, 200, 'API key refreshed successfully', updatedUser);
});

// IMPORTANT: More specific routes must be mounted BEFORE more general routes.
app.route('/api/v1/admin', adminApp);
app.route('/api/v1/pool', poolApp);
app.route('/api/v1', userApp);


// =================================================================
// Documentation Generation
// =================================================================
app.doc('/doc', {
    openapi: '3.0.0',
    info: {
        version: '1.0.0',
        title: 'Cookie Syncer API',
    },
});

app.get('/swagger', swaggerUI({ url: '/doc' }));

export default app;
