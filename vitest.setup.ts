import { config } from 'dotenv';
import { resolve } from 'path';

// Load .env.test first (highest priority)
config({ path: resolve(process.cwd(), '.env.test') });

// Then load .env (fallback)
config({ path: resolve(process.cwd(), '.env') });

console.log('Vitest setup: Environment variables loaded from .env files');
