import { createRoute, z } from '@hono/zod-openapi';

// =================================================================
// Reusable Schemas
// =================================================================
export const UserSchema = z.object({
    id: z.number().int().openapi({ example: 1 }),
    sharing_enabled: z.boolean().openapi({ example: false }),
    remark: z.string().nullable().openapi({ example: 'My personal API key' }),
    last_synced_at: z.string().datetime().nullable().openapi({ example: '2025-10-18T10:00:00Z' }),
    created_at: z.string().datetime().openapi({ example: '2025-10-18T10:00:00Z' }),
    updated_at: z.string().datetime().openapi({ example: '2025-10-18T10:00:00Z' }),
});

// A schema for admin responses that INCLUDES the API key.
export const AdminUserResponseSchema = UserSchema.extend({
    api_key: z.string().uuid().openapi({ example: 'xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx' }),
});

export const CookieSchema = z.object({
    id: z.number().int().optional().openapi({ example: 101 }),
    user_id: z.number().int().optional().openapi({ example: 1 }),
    domain: z.string().openapi({ example: '.example.com' }),
    name: z.string().openapi({ example: 'session_id' }),
    value: z.string().openapi({ example: 'abc-123' }),
    path: z.string().openapi({ example: '/' }),
    expires: z.string().datetime().nullable().openapi({ example: '2026-10-18T10:00:00Z' }),
    http_only: z.boolean().openapi({ example: true }),
    secure: z.boolean().openapi({ example: true }),
    same_site: z.string().openapi({ example: 'Lax' }),
    is_sharable: z.boolean().openapi({ example: false }),
    last_updated_from_extension_at: z.string().datetime().openapi({ example: '2025-10-18T10:00:00Z' }),
});

const StandardResponseSchema = (dataType: z.ZodTypeAny) => z.object({
    code: z.number().int().openapi({ example: 200 }),
    message: z.string().openapi({ example: 'Success' }),
    data: dataType.optional(),
});

// =================================================================
// Route Definitions
// =================================================================

// --- Public Routes ---
export const healthCheckRoute = createRoute({
    method: 'get',
    path: '/api/v1/health',
    summary: 'Health Check',
    tags: ['Public'],
    responses: {
        200: {
            description: 'Service is healthy',
            content: { 'application/json': { schema: StandardResponseSchema(z.object({})) } },
        },
    },
});

// --- Auth Routes ---
export const authTestRoute = createRoute({
    method: 'get',
    path: '/auth/test',
    summary: 'Test API Key',
    tags: ['Auth'],
    security: [{ ApiKeyAuth: [] }],
    responses: {
        200: {
            description: 'Successful authentication',
            content: { 'application/json': { schema: StandardResponseSchema(z.object({
                user_id: z.number().int()
            }))}},
        },
    },
});

// --- User Routes ---
export const getUserSettingsRoute = createRoute({
    method: 'get',
    path: '/user/settings',
    summary: 'Get user settings',
    tags: ['User'],
    security: [{ ApiKeyAuth: [] }],
    responses: { 200: { description: 'User settings', content: { 'application/json': { schema: StandardResponseSchema(UserSchema) }}}},
});

export const updateUserSettingsRoute = createRoute({
    method: 'put',
    path: '/user/settings',
    summary: 'Update user settings',
    tags: ['User'],
    security: [{ ApiKeyAuth: [] }],
    request: { body: { content: { 'application/json': { schema: z.object({ sharing_enabled: z.boolean() }) }}}},
    responses: { 200: { description: 'Settings updated', content: { 'application/json': { schema: StandardResponseSchema(z.object({})) }}}},
});


// --- Cookie Routes ---
export const syncRoute = createRoute({
    method: 'post',
    path: '/sync',
    summary: 'Sync cookies',
    tags: ['Cookies'],
    security: [{ ApiKeyAuth: [] }],
    request: { body: { content: { 'application/json': { schema: z.array(CookieSchema) }}}},
    responses: { 200: { description: 'Sync successful', content: { 'application/json': { schema: StandardResponseSchema(z.object({})) }}}},
});

export const getAllCookiesRoute = createRoute({
    method: 'get',
    path: '/cookies/all',
    summary: 'Get all cookies for user',
    tags: ['Cookies'],
    security: [{ ApiKeyAuth: [] }],
    request: {
        query: z.object({
            format: z.enum(['json', 'header']).optional().openapi({
                param: { name: 'format', in: 'query' },
                description: 'json: returns full cookie objects. header: returns a map of domain to HTTP Cookie header string.',
                example: 'header',
            }),
        }),
    },
    responses: { 200: { description: 'All cookies for the user, format depends on the `format` query param.', content: { 'application/json': { schema: StandardResponseSchema(z.any()) }}}},
});

export const getDomainCookiesRoute = createRoute({
    method: 'get',
    path: '/cookies/{domain}',
    summary: 'Get cookies for a specific domain',
    tags: ['Cookies'],
    security: [{ ApiKeyAuth: [] }],
    request: {
        params: z.object({
            domain: z.string().openapi({
                param: { name: 'domain', in: 'path' },
                example: 'example.com',
            }),
        }),
        query: z.object({
            format: z.enum(['json', 'header']).optional().openapi({
                param: { name: 'format', in: 'query' },
                description: 'json: returns a map of cookie objects, keyed by cookie name. header: returns a single HTTP Cookie header string.',
                example: 'header',
            }),
        }),
    },
    responses: { 200: { description: 'Cookies for the domain, format depends on the `format` query param.', content: { 'application/json': { schema: StandardResponseSchema(z.any()) }}}},
});

export const getCookieValueRoute = createRoute({
    method: 'get',
    path: '/cookies/{domain}/{name}',
    summary: 'Get a single cookie\'s value',
    tags: ['Cookies'],
    security: [{ ApiKeyAuth: [] }],
    request: {
        params: z.object({
            domain: z.string().openapi({
                param: { name: 'domain', in: 'path' },
                example: 'example.com',
            }),
            name: z.string().openapi({
                param: { name: 'name', in: 'path' },
                example: 'session_id',
            }),
        }),
    },
    responses: { 200: { description: 'Cookie value', content: { 'application/json': { schema: StandardResponseSchema(z.string()) }}}},
});


// --- Admin Routes ---
export const adminCreateUserRoute = createRoute({
    method: 'post',
    path: '/users',
    summary: '[Admin] Create one or more new users',
    tags: ['Admin'],
    security: [{ AdminKeyAuth: [] }],
    request: {
        body: {
            content: {
                'application/json': {
                    schema: z.array(z.object({
                        remark: z.string().nullable().optional().openapi({ example: 'New user for project X' }),
                    })).openapi({
                        description: 'An array of user objects to create. Send an empty array to create one default user.'
                    }),
                },
            },
        },
    },
    responses: { 201: { description: 'Users created successfully', content: { 'application/json': { schema: StandardResponseSchema(z.array(AdminUserResponseSchema)) }}}},
});

export const adminUpdateUserRoute = createRoute({
    method: 'put',
    path: '/users/{id}',
    summary: '[Admin] Update a user\'s details',
    tags: ['Admin'],
    security: [{ AdminKeyAuth: [] }],
    request: {
        params: z.object({ id: z.string().openapi({ param: { name: 'id', in: 'path' }, example: '1' }) }),
        body: { content: { 'application/json': { schema: z.object({ remark: z.string().nullable() }) } } }
    },
    responses: { 200: { description: 'User updated successfully', content: { 'application/json': { schema: StandardResponseSchema(z.object({})) }}}},
});

export const adminUpdateUserByApiKeyRoute = createRoute({
    method: 'put',
    path: '/users/by-key/{apiKey}',
    summary: '[Admin] Update a user\'s details by API Key',
    tags: ['Admin'],
    security: [{ AdminKeyAuth: [] }],
    request: {
        params: z.object({ apiKey: z.string().openapi({ param: { name: 'apiKey', in: 'path' }, example: 'xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx' }) }),
        body: { content: { 'application/json': { schema: z.object({ remark: z.string().nullable() }) } } }
    },
    responses: { 200: { description: 'User updated successfully', content: { 'application/json': { schema: StandardResponseSchema(z.object({})) }}}},
});

export const adminRefreshUserApiKeyRoute = createRoute({
    method: 'post',
    path: '/users/{id}/refresh-key',
    summary: '[Admin] Refresh a user\'s API key',
    tags: ['Admin'],
    security: [{ AdminKeyAuth: [] }],
    request: {
        params: z.object({ id: z.string().openapi({ param: { name: 'id', in: 'path' }, example: '1' }) }),
    },
    responses: { 200: { description: 'API key refreshed successfully', content: { 'application/json': { schema: StandardResponseSchema(AdminUserResponseSchema) }}}},
});

export const adminRefreshUserApiKeyByApiKeyRoute = createRoute({
    method: 'post',
    path: '/users/by-key/{apiKey}/refresh-key',
    summary: '[Admin] Refresh a user\'s API key by API Key',
    tags: ['Admin'],
    security: [{ AdminKeyAuth: [] }],
    request: {
        params: z.object({ apiKey: z.string().openapi({ param: { name: 'apiKey', in: 'path' }, example: 'xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx' }) }),
    },
    responses: { 200: { description: 'API key refreshed successfully', content: { 'application/json': { schema: StandardResponseSchema(AdminUserResponseSchema) }}}},
});


export const getSharableCookiesRoute = createRoute({
    method: 'get',
    path: '/cookies/{domain}',
    summary: 'Get sharable cookies for a domain',
    tags: ['Pool'],
    security: [{ PoolKeyAuth: [] }], // A new security scheme for the pool
    request: {
        params: z.object({
            domain: z.string().openapi({
                param: { name: 'domain', in: 'path' },
                example: 'example.com',
            }),
        }),
        query: z.object({
            format: z.enum(['json', 'header']).optional().openapi({
                param: { name: 'format', in: 'query' },
                description: 'json: returns cookies as a nested map, grouped by user, domain, and cookie name. header: returns an array of HTTP Cookie header strings, one per user.',
                example: 'header',
            }),
        }),
    },
    responses: { 200: { description: 'Sharable cookies for the domain, format depends on the `format` query param.', content: { 'application/json': { schema: StandardResponseSchema(z.any()) }}}},
});
