import { z } from 'zod';
import { CookieSchema } from './schema'; // We will define these in schema.ts

// This is the internal representation of a user, matching the database schema.
export interface User {
    id: number;
    api_key: string;
    sharing_enabled: 0 | 1;
    remark: string | null;
    last_synced_at: string | null;
    created_at: string;
    updated_at: string;
}

// The Cookie type is inferred from the Zod schema, which is currently aligned with the database structure.
export type Cookie = z.infer<typeof CookieSchema>;
