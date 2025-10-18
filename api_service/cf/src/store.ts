import type { D1Database } from '@cloudflare/workers-types';
import { v4 as uuidv4 } from 'uuid';
import type { Cookie, User } from './models';

/**
 * D1Store provides an interface for interacting with the D1 database.
 * It encapsulates all the SQL queries and logic for data management.
 */
export class D1Store {
  private db: D1Database;
  private adminKey: string;
  private poolKey: string;

  constructor(db: D1Database, adminKey: string, poolKey: string) {
    this.db = db;
    this.adminKey = adminKey;
    this.poolKey = poolKey;
  }

  // --- User Methods ---

  async getUserByAPIKey(apiKey: string): Promise<User | null> {
    // Updated to select the new `last_synced_at` field.
    return this.db
      .prepare('SELECT id, api_key, sharing_enabled, remark, last_synced_at, created_at, updated_at FROM users WHERE api_key = ?1')
      .bind(apiKey)
      .first<User>();
  }

  async getUserByID(userId: number): Promise<User | null> {
    return this.db
      .prepare('SELECT id, api_key, sharing_enabled, remark, last_synced_at, created_at, updated_at FROM users WHERE id = ?1')
      .bind(userId)
      .first<User>();
  }
  
  async userExists(): Promise<boolean> {
      const result = await this.db
          .prepare("SELECT COUNT(*) as count FROM users")
          .first<{ count: number }>();
      return result ? result.count > 0 : false;
  }

  private generateSafeApiKey(): string {
      let apiKey: string;
      do {
          apiKey = uuidv4();
      } while (apiKey === this.adminKey || apiKey === this.poolKey);
      return apiKey;
  }

  async createUser(remark: string | null | undefined): Promise<User> {
      const now = new Date().toISOString();
      const newUser = {
          api_key: this.generateSafeApiKey(),
          sharing_enabled: 0 as 0 | 1,
          remark: remark ?? null,
          last_synced_at: null,
          created_at: now,
          updated_at: now,
      };

      const result = await this.db
          .prepare(
              'INSERT INTO users (api_key, sharing_enabled, remark, last_synced_at, created_at, updated_at) VALUES (?1, ?2, ?3, ?4, ?5, ?6) RETURNING id'
          )
          .bind(
              newUser.api_key,
              newUser.sharing_enabled,
              newUser.remark,
              newUser.last_synced_at,
              newUser.created_at,
              newUser.updated_at
          )
          .first<{ id: number }>();
      
      return { id: result!.id, ...newUser };
  }

  async updateUserSharing(userId: number, enabled: boolean): Promise<void> {
      await this.db
          .prepare('UPDATE users SET sharing_enabled = ?1, updated_at = ?2 WHERE id = ?3')
          .bind(enabled ? 1 : 0, new Date().toISOString(), userId)
          .run();
  }
  
  async updateUserRemark(userId: number, remark: string | null): Promise<void> {
      await this.db
          .prepare('UPDATE users SET remark = ?1, updated_at = ?2 WHERE id = ?3')
          .bind(remark, new Date().toISOString(), userId)
          .run();
  }

  async updateUserRemarkByAPIKey(apiKey: string, remark: string | null): Promise<void> {
      await this.db
          .prepare('UPDATE users SET remark = ?1, updated_at = ?2 WHERE api_key = ?3')
          .bind(remark, new Date().toISOString(), apiKey)
          .run();
  }

  async adminUpdateUserAPIKey(userId: number): Promise<User | null> {
      const newAPIKey = this.generateSafeApiKey();
      await this.db
        .prepare('UPDATE users SET api_key = ?1, updated_at = ?2 WHERE id = ?3')
        .bind(newAPIKey, new Date().toISOString(), userId)
        .run();
      return this.getUserByID(userId);
  }

  async adminUpdateUserAPIKeyByAPIKey(apiKey: string): Promise<User | null> {
       const newAPIKey = this.generateSafeApiKey();
       await this.db
           .prepare('UPDATE users SET api_key = ?1, updated_at = ?2 WHERE api_key = ?3')
           .bind(newAPIKey, new Date().toISOString(), apiKey)
           .run();
       return this.getUserByAPIKey(newAPIKey);
  }

  async deleteUsers(ids: number[]): Promise<void> {
      if (ids.length === 0) return;
      const placeholders = ids.map(() => '?').join(',');
      await this.db
          .prepare(`DELETE FROM users WHERE id IN (${placeholders})`)
          .bind(...ids)
          .run();
  }

  // --- Cookie Methods ---

  async syncCookies(userId: number, cookies: Cookie[]): Promise<void> {
    const now = new Date().toISOString();
    const cookiesJson = JSON.stringify(cookies);

    const updateUserStmt = this.db
      .prepare('UPDATE users SET cookies_json = ?1, last_synced_at = ?2, updated_at = ?2 WHERE id = ?3')
      .bind(cookiesJson, now, userId);

    const deleteCookiesStmt = this.db.prepare('DELETE FROM cookies WHERE user_id = ?1').bind(userId);
    
    const statements = [updateUserStmt, deleteCookiesStmt];

    if (cookies.length > 0) {
      const insertStmt = this.db.prepare(
        `INSERT INTO cookies (user_id, domain, name, value, path, expires, http_only, secure, same_site, is_sharable, last_updated_from_extension_at)
         VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11)`
      );
      
      cookies.forEach(cookie => {
        statements.push(
          insertStmt.bind(
            userId,
            cookie.domain,
            cookie.name,
            cookie.value,
            cookie.path,
            cookie.expires,
            cookie.http_only ? 1 : 0,
            cookie.secure ? 1 : 0,
            cookie.same_site,
            cookie.is_sharable ? 1 : 0,
            now // Use the same 'now' timestamp for consistency
          )
        );
      });
    }
    
    await this.db.batch(statements);
  }

  async getAllCookies(userId: number): Promise<Cookie[]> {
    const user = await this.db
      .prepare('SELECT cookies_json FROM users WHERE id = ?1')
      .bind(userId)
      .first<{ cookies_json: string | null }>();

    if (!user?.cookies_json) {
      return [];
    }

    try {
      return JSON.parse(user.cookies_json) as Cookie[];
    } catch (e) {
      console.error(`Failed to parse cookies_json for user ${userId}:`, e);
      return []; // Return empty array on parsing error
    }
  }
  
  async getCookiesByDomain(userId: number, domain: string): Promise<Cookie[]> {
      const allCookies = await this.getAllCookies(userId);
      return allCookies.filter(cookie => cookie.domain === domain || cookie.domain.endsWith(`.${domain}`));
  }

  async getSharableCookiesByDomain(domain: string): Promise<Cookie[]> {
      // Use LIKE to match the domain and its subdomains
      const { results } = await this.db
          .prepare(
              'SELECT c.* FROM cookies c JOIN users u ON c.user_id = u.id WHERE (c.domain = ?1 OR c.domain LIKE ?2) AND u.sharing_enabled = 1 AND c.is_sharable = 1'
          )
          .bind(domain, `%${domain}`)
          .all<Cookie>();
      return results;
  }
}
