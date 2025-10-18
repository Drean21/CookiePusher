import { execSync } from 'child_process';

// A helper to execute a single query and get the first result
export async function queryOne<T = unknown>(query: string, params: any[] = []): Promise<T | null> {
    const localFlag = process.env.CF_REMOTE === 'true' ? '--remote' : '--local';
    const dbName = 'cookie-syncer-db';
    
    // wrangler d1 execute requires params to be a stringified JSON array
    const paramsJson = JSON.stringify(params);

    try {
        // Use --json flag to get structured output
        const cmd = `wrangler d1 execute ${dbName} ${localFlag} --json --command="${query}" --params='${paramsJson}'`;
        const output = execSync(cmd, { encoding: 'utf-8', stdio: 'pipe' });
        
        // The output from wrangler is an array of results for batch operations. We take the first.
        const results = JSON.parse(output);
        if (!results || results.length === 0) {
            throw new Error("Wrangler command returned no results.");
        }
        const firstResult = results[0];

        if (firstResult.results && firstResult.results.length > 0) {
            return firstResult.results[0] as T;
        }
        // This is for INSERT...RETURNING id, where the result is not in the `results` array
        if (firstResult.id !== undefined) {
             return firstResult as T;
        }
        
        return null;
    } catch (e: any) {
        console.error(`Failed to execute D1 query via Wrangler: ${e.message}`);
        if (e.stderr) {
            console.error(`Wrangler stderr: ${e.stderr}`);
        }
        throw new Error(`D1 execution failed. Is Wrangler logged in and configured correctly?`);
    }
}
