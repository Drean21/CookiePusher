import { v4 as uuidv4 } from 'uuid';
import { queryOne } from './db';

async function ensureInitialUser() {
    console.log('Checking for existing users...');
    
    try {
        const userExists = await queryOne<{ count: number }>(
            "SELECT COUNT(*) as count FROM users"
        );

        if (userExists && userExists.count > 0) {
            console.log('At least one user already exists. No action needed.');
            console.log('If you need to create a new user, please do so manually or extend this script.');
            return;
        }

        console.log('No users found. Creating an initial user...');
        const now = new Date().toISOString();
        const newApiKey = uuidv4();
        
        const newUser = await queryOne<{ id: number }>(
            'INSERT INTO users (api_key, sharing_enabled, created_at, updated_at) VALUES (?1, ?2, ?3, ?4) RETURNING id',
            [newApiKey, 0, now, now]
        );

        if (!newUser || newUser.id === undefined) {
            throw new Error('Failed to retrieve ID of newly created user.');
        }

        console.log('Successfully created initial user:');
        console.log(`  - ID: ${newUser.id}`);
        console.log('  - API Key: ' + newApiKey);
        console.log('\nIMPORTANT: Please save this API key securely. It will not be shown again.');

    } catch (error) {
        console.error('An error occurred during initial user setup:', error);
        process.exit(1);
    }
}

ensureInitialUser();
