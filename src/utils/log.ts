import * as fs from 'fs';
import * as path from 'path';

// Create logs directory if it doesn't exist
const logsDir = path.join(process.cwd(), 'logs');
if (!fs.existsSync(logsDir)) {
  fs.mkdirSync(logsDir, { recursive: true });
}

// Create log file paths with timestamps
const logFile = path.join(logsDir, `mcp-server.log`);
const errorFile = path.join(logsDir, `mcp-server-error.log`);

// Create write streams
const logStream = fs.createWriteStream(logFile, { flags: 'a' });
const errorStream = fs.createWriteStream(errorFile, { flags: 'a' });

// Helper function to format log messages
function formatMessage(...args: any[]): string {
  const timestamp = new Date().toISOString();
  const message = args.map(arg =>
    typeof arg === 'object' ? JSON.stringify(arg, null, 2) : String(arg)
  ).join(' ');
  return `[${timestamp}] ${message}\n`;
}

// Export custom logger functions if needed
export const fileLogger = {
  log: (...args: any[]) => {
    const message = formatMessage(...args);
    logStream.write(message);
  },
  error: (...args: any[]) => {
    const message = formatMessage('ERROR:', ...args);
    errorStream.write(message);
  },
  info: (...args: any[]) => {
    const message = formatMessage('INFO:', ...args);
    logStream.write(message);
  },
  warn: (...args: any[]) => {
    const message = formatMessage('WARN:', ...args);
    errorStream.write(message);
  },
  debug: (...args: any[]) => {
    const message = formatMessage('DEBUG:', ...args);
    logStream.write(message);
  },
  // Utility to get log file paths
  getLogPaths: () => ({
    logFile,
    errorFile,
    logsDir
  })
};

// Handle process exit to close streams
process.on('exit', () => {
  logStream.end();
  errorStream.end();
});

process.on('SIGINT', () => {
  logStream.end();
  errorStream.end();
  process.exit();
});

process.on('uncaughtException', (error) => {
  console.error('Uncaught Exception:', error);
  logStream.end();
  errorStream.end();
  process.exit(1);
});
