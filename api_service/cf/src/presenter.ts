import type { User as DbUser } from './models';

// This is the user object shape the API will expose to the frontend.
export interface ApiUser {
    id: number;
    sharing_enabled: boolean; // Note: this is a boolean
    remark: string | null;
    created_at: string;
    updated_at: string;
}

/**
 * Converts a User object from the database format (with 0/1 for booleans)
 * to the API format (with true/false).
 * @param user The user object from the database.
 * @returns A user object formatted for API responses.
 */
export function toApiUser(user: DbUser): ApiUser {
    return {
        id: user.id,
        sharing_enabled: !!user.sharing_enabled, // Converts 0 to false, 1 to true
        remark: user.remark,
        created_at: user.created_at,
        updated_at: user.updated_at,
    };
}
